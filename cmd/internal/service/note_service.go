package service

import (
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
	"net/http"
	"regexp"
	"simplenotes/internal/domain/entity"
	"simplenotes/internal/domain/sqlite"
	"strings"
	"time"
)

type NoteResponse struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Content   string   `json:"content"`
	Aliases   []string `json:"aliases"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

type NoteRequest struct {
	Name    string   `json:"name" validate:"required,min=2,max=80"`
	Type    string   `json:"type" validate:"required,oneof=PDF IMAGE TEXT VIDEO AUDIO"`
	Aliases []string `json:"aliases" validate:"required,max=50"`
	Content string   `json:"content" validate:"required"`
}

const (
	MinAliasLength = 2
	MaxAliasLength = 30
)

var validate = validator.New()
var whitespaceRegex = regexp.MustCompile(`\s+`)

func GetAllNotes() ([]*NoteResponse, *APIError) {
	var notes []*entity.Note
	var aliases []*entity.NoteAlias

	err := sqlite.DB.Find(&notes).Error
	if err != nil {
		log.Errorf("failed to fetch notes: %v", err)
		return nil, ErrorInternalServer
	}

	err = sqlite.DB.Find(&aliases).Error
	if err != nil {
		log.Errorf("failed to fetch aliases: %v", err)
		return nil, ErrorInternalServer
	}

	mappedAliases := toNotesAliasesMap(aliases)
	resp := make([]*NoteResponse, len(notes))
	for i, note := range notes {
		noteAliases, ok := mappedAliases[note.ID]
		if !ok {
			noteAliases = []string{}
		}

		resp[i] = toNoteResponse(note, noteAliases)
	}
	return resp, nil
}

func CreateNote(req *NoteRequest) (*NoteResponse, *APIError) {
	if err := validate.Struct(req); err != nil {
		return nil, NewError(http.StatusBadRequest, err.Error())
	}

	req.Aliases = sanitizeAliases(req.Aliases)
	apierr := validateAliases(req.Aliases)
	if apierr != nil {
		return nil, apierr
	}

	var note *entity.Note
	err := sqlite.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		note, err = trxInsertNoteWithAliases(tx, req)
		return err
	})

	if err != nil {
		log.Errorf("failed to create note: %v", err)
		return nil, ErrorInternalServer
	}

	// Shhh 🤫, they will never notice this
	return toNoteResponse(note, req.Aliases), nil
}

func DeleteNote(noteId int) *APIError {
	var note *entity.Note
	err := sqlite.DB.First(&note, noteId).Error
	isMissing := errors.Is(err, gorm.ErrRecordNotFound)
	if err != nil && !isMissing {
		log.Errorf("failed to fetch note: %v", err)
		return ErrorInternalServer
	}

	if isMissing {
		return ErrorNotFound
	}

	err = sqlite.DB.Delete(note, noteId).Error
	if err != nil {
		log.Errorf("failed to delete note: %v", err)
		return ErrorInternalServer
	}
	return nil
}

// toNotesAliasesMap returns a map that associates each note ID with its corresponding list of aliases.
// It takes a slice of NoteAlias entities as input and returns a map where the key is the note ID,
// and the value is a slice of strings representing all aliases of that note.
func toNotesAliasesMap(aliases []*entity.NoteAlias) map[int][]string {
	out := make(map[int][]string)
	for _, alias := range aliases {
		noteID := alias.NoteID
		out[noteID] = append(out[noteID], alias.Value)
	}
	return out
}

// trxInsertNoteWithAliases saves the whole note (with its aliases) in a single transaction
func trxInsertNoteWithAliases(tx *gorm.DB, req *NoteRequest) (*entity.Note, error) {
	timestamp := time.Now().UTC().UnixMilli()
	note := &entity.Note{
		Name:      req.Name,
		Type:      req.Type,
		Content:   req.Content,
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
	}

	if err := tx.Create(note).Error; err != nil {
		return nil, err
	}

	if len(req.Aliases) <= 0 {
		return note, nil
	}

	aliases := make([]*entity.NoteAlias, len(req.Aliases))
	for i, alias := range req.Aliases {
		aliases[i] = &entity.NoteAlias{
			NoteID:    note.ID,
			Value:     strings.ToLower(alias),
			CreatedAt: timestamp,
		}
	}

	if err := tx.Create(&aliases).Error; err != nil {
		return nil, err
	}
	return note, nil
}

func validateAliases(vals []string) *APIError {
	for _, val := range vals {
		if err := validateAlias(val); err != nil {
			return err
		}
	}

	if hasDuplicates(vals) {
		return ErrorDuplicateAlias
	}
	return nil
}

func validateAlias(val string) *APIError {
	size := len(val)

	if size < MinAliasLength || size > MaxAliasLength {
		return NewAliasLengthError(val, MinAliasLength, MaxAliasLength)
	}
	return nil
}

func hasDuplicates(vals []string) bool {
	seen := make(map[string]bool)

	for _, val := range vals {
		if seen[val] {
			return true
		}
		seen[val] = true
	}
	return false
}

func sanitizeAliases(vals []string) []string {
	out := make([]string, len(vals))
	for i, val := range vals {
		noSpaces := whitespaceRegex.ReplaceAllString(val, "")
		out[i] = strings.ToLower(noSpaces)
	}
	return out
}

func toNoteResponse(note *entity.Note, aliases []string) *NoteResponse {
	return &NoteResponse{
		ID:        note.ID,
		Name:      note.Name,
		Type:      note.Type,
		Content:   note.Content,
		Aliases:   aliases,
		CreatedAt: FormatEpoch(note.CreatedAt),
		UpdatedAt: FormatEpoch(note.UpdatedAt),
	}
}

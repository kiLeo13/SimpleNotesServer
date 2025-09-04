package service

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/gommon/log"
	"net/http"
	"regexp"
	"simplenotes/internal/domain/entity"
	"strings"
)

const (
	MinAliasLength = 2
	MaxAliasLength = 30
)

var whitespaceRegex = regexp.MustCompile(`\s+`)

type NoteResponse struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags"`
	CreatedByID int      `json:"created_by_id"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type NoteRequest struct {
	Name    string   `json:"name" validate:"required,min=2,max=80"`
	Tags    []string `json:"tags" validate:"required,max=50"`
	Content string   `json:"content" validate:"required"`
}

type NoteRepository interface {
	FindAll() ([]*entity.Note, error)
	FindByID(id int) (*entity.Note, error)
	Save(note *entity.Note) error
	Delete(note *entity.Note) error
}

type DefaultNoteService struct {
	NoteRepo NoteRepository
	Validate *validator.Validate
}

func NewNoteService(noteRepo NoteRepository, validate *validator.Validate) *DefaultNoteService {
	return &DefaultNoteService{NoteRepo: noteRepo, Validate: validate}
}

func (n *DefaultNoteService) GetAllNotes() ([]*NoteResponse, *APIError) {
	notes, err := n.NoteRepo.FindAll()
	if err != nil {
		log.Errorf("failed to fetch notes: %v", err)
		return nil, InternalServerError
	}

	resp := make([]*NoteResponse, len(notes))
	for i, note := range notes {
		resp[i] = toNoteResponse(note)
	}
	return resp, nil
}

func (n *DefaultNoteService) CreateNote(req *NoteRequest) (*NoteResponse, *APIError) {
	if err := n.Validate.Struct(req); err != nil {
		return nil, NewError(http.StatusBadRequest, err.Error())
	}

	req.Tags = sanitizeAliases(req.Tags)
	apierr := validateAliases(req.Tags)
	if apierr != nil {
		return nil, apierr
	}

	now := NowUTC()
	note := &entity.Note{
		Name:      req.Name,
		Content:   req.Content,
		Tags:      strings.Join(req.Tags, " "),
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := n.NoteRepo.Save(note)
	if err != nil {
		log.Errorf("failed to create note: %v", err)
		return nil, InternalServerError
	}
	return toNoteResponse(note), nil
}

func (n *DefaultNoteService) DeleteNote(noteId int) *APIError {
	note, err := n.NoteRepo.FindByID(noteId)
	if err != nil {
		log.Errorf("failed to fetch note: %v", err)
		return InternalServerError
	}

	if note == nil {
		return NotFoundError
	}

	err = n.NoteRepo.Delete(note)
	if err != nil {
		log.Errorf("failed to delete note: %v", err)
		return InternalServerError
	}
	return nil
}

func validateAliases(vals []string) *APIError {
	for _, val := range vals {
		if err := validateAlias(val); err != nil {
			return err
		}
	}

	if hasDuplicates(vals) {
		return DuplicateAliasError
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

func toNoteResponse(note *entity.Note) *NoteResponse {
	return &NoteResponse{
		ID:          note.ID,
		Name:        note.Name,
		Content:     note.Content,
		Tags:        strings.Split(note.Tags, " "),
		CreatedByID: note.CreatedByID,
		CreatedAt:   FormatEpoch(note.CreatedAt),
		UpdatedAt:   FormatEpoch(note.UpdatedAt),
	}
}

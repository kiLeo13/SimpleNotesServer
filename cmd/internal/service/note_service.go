package service

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"io"
	"mime/multipart"
	"path/filepath"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/infrastructure/aws/storage"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
	"strings"
)

const MaxNoteFileSizeBytes = 30 * 1024 * 1024

var ValidNoteFileTypes = []string{"txt", "md", "pdf", "png", "jpg", "jpeg", "jfif", "webp", "gif", "mp4", "mp3"}

type NoteResponse struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Content     string   `json:"content,omitempty"`
	Tags        []string `json:"tags"`
	Visibility  string   `json:"visibility"`
	NoteType    string   `json:"note_type"`
	ContentSize int      `json:"content_size"`
	CreatedByID int      `json:"created_by_id"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type NoteRequest struct {
	Name       string   `form:"name" validate:"required,min=2,max=80"`
	Visibility string   `form:"visibility" validate:"required,oneof=PUBLIC CONFIDENTIAL"`
	Tags       []string `form:"tags" validate:"required,max=50,nodupes,dive,required,min=2,max=30,nospaces"`
}

type UpdateNoteRequest struct {
	Name       *string  `form:"name" validate:"omitempty,min=2,max=80"`
	Visibility *string  `form:"visibility" validate:"omitempty,oneof=PUBLIC CONFIDENTIAL"`
	Tags       []string `form:"tags" validate:"omitempty,max=50,nodupes,dive,required,min=2,max=30,nospaces"`
}

type NoteRepository interface {
	FindAll() ([]*entity.Note, error)
	FindByID(id int) (*entity.Note, error)
	Save(note *entity.Note) error
	Delete(note *entity.Note) error
}

type DefaultNoteService struct {
	NoteRepo NoteRepository
	UserRepo UserRepository
	S3       storage.S3Client
	Validate *validator.Validate
}

func NewNoteService(noteRepo NoteRepository, userRepo UserRepository, s3 storage.S3Client, validate *validator.Validate) *DefaultNoteService {
	return &DefaultNoteService{NoteRepo: noteRepo, UserRepo: userRepo, S3: s3, Validate: validate}
}

func (n *DefaultNoteService) GetAllNotes() ([]*NoteResponse, apierror.ErrorResponse) {
	notes, err := n.NoteRepo.FindAll()
	if err != nil {
		log.Errorf("failed to fetch notes: %v", err)
		return nil, apierror.InternalServerError
	}

	resp := make([]*NoteResponse, len(notes))
	for i, note := range notes {
		resp[i] = toNoteResponse(note, false)
	}
	return resp, nil
}

func (n *DefaultNoteService) GetNoteByID(id int) (*NoteResponse, apierror.ErrorResponse) {
	note, err := n.NoteRepo.FindByID(id)
	if err != nil {
		log.Errorf("failed to fetch note: %v", err)
		return nil, apierror.InternalServerError
	}

	if note == nil {
		return nil, apierror.NotFoundError
	}
	return toNoteResponse(note, true), nil
}

func (n *DefaultNoteService) CreateNote(req *NoteRequest, fileHeader *multipart.FileHeader, issuerId string) (*NoteResponse, apierror.ErrorResponse) {
	issuer, err := n.UserRepo.FindBySub(issuerId)
	if err != nil {
		log.Errorf("failed to check if user %s is admin: %v", issuerId, err)
		return nil, apierror.InternalServerError
	}

	// I don't know how the user can even be nil here, but better safe than sorry?
	if issuer == nil || !issuer.IsAdmin {
		return nil, apierror.UserNotAdmin
	}

	utils.Sanitize(req)
	if valerr := n.Validate.Struct(req); valerr != nil {
		return nil, apierror.FromValidationError(valerr)
	}

	if apierr := checkNoteFile(fileHeader); apierr != nil {
		return nil, apierr
	}

	ext := filepath.Ext(fileHeader.Filename)
	filename, apierr := handleNoteUpload(n.S3, fileHeader)
	if apierr != nil {
		return nil, apierr
	}

	tags := strings.Join(req.Tags, " ")
	now := utils.NowUTC()
	note := &entity.Note{
		Name:        req.Name,
		Content:     filename,
		CreatedByID: issuer.ID,
		Tags:        strings.ToLower(tags),
		NoteType:    toNoteType(ext),
		ContentSize: int(fileHeader.Size), // I really hope never exceed the 32-bits content length lol
		Visibility:  req.Visibility,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = n.NoteRepo.Save(note)
	if err != nil {
		log.Errorf("failed to create note: %v", err)
		return nil, apierror.InternalServerError
	}
	return toNoteResponse(note, true), nil
}

func (n *DefaultNoteService) UpdateNote(id int, userSub string, req *UpdateNoteRequest) (*NoteResponse, apierror.ErrorResponse) {
	utils.Sanitize(req)
	if valerr := n.Validate.Struct(req); valerr != nil {
		return nil, apierror.FromValidationError(valerr)
	}

	user, err := n.UserRepo.FindBySub(userSub)
	if err != nil {
		log.Errorf("failed to fetch user: %v", err)
		return nil, apierror.InternalServerError
	}

	if user == nil || !user.IsAdmin {
		return nil, apierror.UserNotAdmin
	}

	note, err := n.NoteRepo.FindByID(id)
	if err != nil {
		log.Errorf("failed to fetch note: %v", err)
		return nil, apierror.InternalServerError
	}

	if note == nil {
		return nil, apierror.NotFoundError
	}

	// Now, we can finally PATCH our data :D
	tags := strings.Join(req.Tags, " ")
	if req.Name != nil {
		note.Name = *req.Name
	}
	if req.Visibility != nil {
		note.Visibility = *req.Visibility
	}
	if req.Tags != nil {
		note.Tags = strings.ToLower(tags)
	}

	note.UpdatedAt = utils.NowUTC()
	err = n.NoteRepo.Save(note)
	if err != nil {
		log.Errorf("failed to update note: %v", err)
		return nil, apierror.InternalServerError
	}
	return toNoteResponse(note, false), nil
}

func (n *DefaultNoteService) DeleteNote(noteId int, issuerId string) apierror.ErrorResponse {
	issuer, err := n.UserRepo.FindBySub(issuerId)
	if err != nil {
		log.Errorf("failed to check if user %s is admin: %v", issuerId, err)
		return apierror.InternalServerError
	}

	if issuer == nil || !issuer.IsAdmin {
		return apierror.UserNotAdmin
	}

	note, err := n.NoteRepo.FindByID(noteId)
	if err != nil {
		log.Errorf("failed to fetch note: %v", err)
		return apierror.InternalServerError
	}

	if note == nil {
		return apierror.NotFoundError
	}

	err = deleteBucketObject(n.S3, note.Content)
	if err != nil {
		log.Errorf("failed to delete note: %v", err)
		return apierror.InternalServerError
	}

	err = n.NoteRepo.Delete(note)
	if err != nil {
		log.Errorf("failed to delete note: %v", err)
		return apierror.InternalServerError
	}
	return nil
}

// handleNoteUpload tries to upload the note to S3.
// EXCEPT if the file is a text file, which in such cases this method
// will return immediately with the content of the text file.
//
// If the file IS NOT a text file, it uploads to the S3 bucket and returns the filename.
func handleNoteUpload(s3 storage.S3Client, fileheader *multipart.FileHeader) (string, apierror.ErrorResponse) {
	ext := filepath.Ext(fileheader.Filename)
	bytes, apierr := readNoteFile(fileheader)
	if apierr != nil {
		return "", apierr
	}

	if isText(ext) {
		return string(bytes), nil
	}

	filename := uuid.NewString() + ext
	err := s3.UploadFile(bytes, storage.PathAttachments+filename)
	if err != nil {
		log.Errorf("failed to upload file: %v", err)
		return "", apierror.InternalServerError
	}
	return filename, nil
}

func checkNoteFile(fileHeader *multipart.FileHeader) apierror.ErrorResponse {
	if fileHeader.Size > MaxNoteFileSizeBytes {
		return apierror.NewNoteContentTooLargeError(MaxNoteFileSizeBytes)
	}

	if strings.TrimSpace(fileHeader.Filename) == "" {
		return apierror.MissingFileNameError
	}

	if ext, ok := utils.CheckFileExt(fileHeader.Filename, ValidNoteFileTypes); !ok {
		return apierror.NewInvalidFileExtError(ext)
	}
	return nil
}

func readNoteFile(fileHeader *multipart.FileHeader) ([]byte, apierror.ErrorResponse) {
	file, err := fileHeader.Open()
	if err != nil {
		log.Errorf("failed to open file: %v", err)
		return nil, apierror.InternalServerError
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Errorf("failed to read file: %v", err)
		return nil, apierror.InternalServerError
	}
	return bytes, nil
}

func toNoteResponse(note *entity.Note, forceContent bool) *NoteResponse {
	content := ""
	if note.NoteType == "REFERENCE" || forceContent {
		content = note.Content
	}

	return &NoteResponse{
		ID:          note.ID,
		Name:        note.Name,
		Content:     content,
		Tags:        toTagsArray(note.Tags),
		Visibility:  note.Visibility,
		NoteType:    note.NoteType,
		ContentSize: note.ContentSize,
		CreatedByID: note.CreatedByID,
		CreatedAt:   utils.FormatEpoch(note.CreatedAt),
		UpdatedAt:   utils.FormatEpoch(note.UpdatedAt),
	}
}

// deleteBucketObject deletes the file with the given name from the attachments directory in S3.
//
// It is idempotent: it returns nil if the object does not exist.
// This prevents errors when the database and S3 bucket are out of sync.
func deleteBucketObject(bucket storage.S3Client, fileName string) error {
	if fileName == "" {
		return fmt.Errorf("deleteBucketObject: filename cannot be empty")
	}

	key := storage.PathAttachments + fileName
	err := bucket.DeleteFile(key)

	var noKey *types.NoSuchKey
	if errors.As(err, &noKey) {
		return nil
	}

	if err != nil {
		return err
	}
	return nil
}

func toTagsArray(tags string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	return strings.Split(tags, " ")
}

func toNoteType(ext string) string {
	if isText(ext) {
		return "TEXT"
	}
	return "REFERENCE"
}

func isText(ext string) bool {
	return ext == ".txt" || ext == ".md"
}

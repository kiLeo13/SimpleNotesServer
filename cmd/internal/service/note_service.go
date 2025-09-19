package service

import (
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
	CreatedByID int      `json:"created_by_id"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type NoteRequest struct {
	Name       string   `form:"name" validate:"required,min=2,max=80"`
	Visibility string   `form:"visibility" validate:"required,oneof=PUBLIC CONFIDENTIAL"`
	Tags       []string `form:"tags" validate:"required,max=50,nodupes,dive,required,min=2,max=30,nospaces"`
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
		resp[i] = toNoteResponse(note)
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
	return toNoteResponse(note), nil
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

	// Here, if the extension represents a `.txt` file, then
	// "filename" will be the raw text inside the file.
	// Any other file extension will be uploaded to S3 and return the filename.
	filename, apierr := handleNoteUpload(n.S3, fileHeader)
	if apierr != nil {
		return nil, apierr
	}

	now := utils.NowUTC()
	note := &entity.Note{
		Name:        req.Name,
		Content:     filename,
		CreatedByID: issuer.ID,
		Tags:        strings.Join(req.Tags, " "),
		Visibility:  req.Visibility,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = n.NoteRepo.Save(note)
	if err != nil {
		log.Errorf("failed to create note: %v", err)
		return nil, apierror.InternalServerError
	}
	return toNoteResponse(note), nil
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

func toNoteResponse(note *entity.Note) *NoteResponse {
	ext := filepath.Ext(note.Content)
	content := ""
	if !isText(ext) {
		content = note.Content
	}

	return &NoteResponse{
		ID:          note.ID,
		Name:        note.Name,
		Content:     content,
		Tags:        strings.Split(note.Tags, " "),
		Visibility:  note.Visibility,
		CreatedByID: note.CreatedByID,
		CreatedAt:   utils.FormatEpoch(note.CreatedAt),
		UpdatedAt:   utils.FormatEpoch(note.UpdatedAt),
	}
}

func isText(ext string) bool {
	return ext == ".txt" || ext == ".md"
}

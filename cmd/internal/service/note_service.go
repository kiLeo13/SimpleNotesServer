package service

import (
	"context"
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
	"simplenotes/cmd/internal/domain/events"
	"simplenotes/cmd/internal/infrastructure/aws/storage"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
	"strings"
)

const MaxNoteFileSizeBytes = 30 * 1024 * 1024

var ValidNoteFileTypes = []string{"pdf", "png", "jpg", "jpeg", "jfif", "webp", "gif", "mp4", "mp3"}

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
	Name       string   `json:"name" validate:"required,min=2,max=80"`
	Visibility string   `json:"visibility" validate:"required,oneof=PUBLIC CONFIDENTIAL"`
	Tags       []string `json:"tags" validate:"required,max=50,nodupes,dive,required,min=2,max=30,nospaces"`
}

type TextNoteRequest struct {
	Name       string   `json:"name" validate:"required,min=2,max=80"`
	Content    string   `json:"content" validate:"required,max=1000000"`
	NoteType   string   `json:"note_type" validate:"required,oneof=MARKDOWN FLOWCHART"`
	Visibility string   `json:"visibility" validate:"required,oneof=PUBLIC CONFIDENTIAL"`
	Tags       []string `json:"tags" validate:"required,max=50,nodupes,dive,required,min=2,max=30,nospaces"`
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
	NoteRepo  NoteRepository
	UserRepo  UserRepository
	WSService *WebSocketService
	S3        storage.S3Client
	Validate  *validator.Validate
}

func NewNoteService(
	noteRepo NoteRepository,
	userRepo UserRepository,
	wsService *WebSocketService,
	s3 storage.S3Client,
	validate *validator.Validate,
) *DefaultNoteService {
	return &DefaultNoteService{
		NoteRepo:  noteRepo,
		UserRepo:  userRepo,
		WSService: wsService,
		S3:        s3,
		Validate:  validate,
	}
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

func (n *DefaultNoteService) GetNoteByID(actor *entity.User, noteId int) (*NoteResponse, apierror.ErrorResponse) {
	note, err := n.NoteRepo.FindByID(noteId)
	if err != nil {
		log.Errorf("failed to fetch note: %v", err)
		return nil, apierror.InternalServerError
	}

	if note == nil {
		return nil, apierror.NotFoundError
	}
	return toNoteResponse(note, true), nil
}

func (n *DefaultNoteService) CreateTextNote(actor *entity.User, req *TextNoteRequest) (*NoteResponse, apierror.ErrorResponse) {
	if !actor.Permissions.HasEffective(entity.PermissionCreateNotes) {
		return nil, apierror.UserMissingPermsError
	}

	utils.Sanitize(req)
	if valerr := n.Validate.Struct(req); valerr != nil {
		return nil, apierror.FromValidationError(valerr)
	}

	tags := strings.Join(req.Tags, " ")
	now := utils.NowUTC()

	note := &entity.Note{
		Name:        req.Name,
		Content:     req.Content,
		CreatedByID: actor.ID,
		Tags:        strings.ToLower(tags),
		NoteType:    entity.NoteType(req.NoteType),
		ContentSize: len(req.Content),
		Visibility:  req.Visibility,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := n.NoteRepo.Save(note); err != nil {
		log.Errorf("failed to save note: %v", err)
		return nil, apierror.InternalServerError
	}

	// I cannot reuse the same response since gateway events should not include the
	// `content` if it's not a REFERENCE type (the payload gets too big ^^).
	go n.dispatchNoteCreateEvent(toNoteResponse(note, false))
	return toNoteResponse(note, true), nil
}

func (n *DefaultNoteService) CreateFileNote(actor *entity.User, req *NoteRequest, fileHeader *multipart.FileHeader) (*NoteResponse, apierror.ErrorResponse) {
	if !actor.Permissions.HasEffective(entity.PermissionCreateNotes) {
		return nil, apierror.UserMissingPermsError
	}

	utils.Sanitize(req)
	if valerr := n.Validate.Struct(req); valerr != nil {
		return nil, apierror.FromValidationError(valerr)
	}

	if apierr := checkNoteFile(fileHeader); apierr != nil {
		return nil, apierr
	}

	filename, fileLength, apierr := handleNoteUpload(n.S3, fileHeader)
	if apierr != nil {
		return nil, apierr
	}

	tags := strings.Join(req.Tags, " ")
	now := utils.NowUTC()
	note := &entity.Note{
		Name:        req.Name,
		Content:     filename,
		CreatedByID: actor.ID,
		Tags:        strings.ToLower(tags),
		NoteType:    entity.NoteTypeReference,
		ContentSize: fileLength,
		Visibility:  req.Visibility,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := n.NoteRepo.Save(note); err != nil {
		log.Errorf("failed to create note: %v", err)
		return nil, apierror.InternalServerError
	}

	resp := toNoteResponse(note, true)
	go n.dispatchNoteCreateEvent(resp)
	return resp, nil
}

func (n *DefaultNoteService) UpdateNote(actor *entity.User, noteId int, req *UpdateNoteRequest) (*NoteResponse, apierror.ErrorResponse) {
	utils.Sanitize(req)
	if valerr := n.Validate.Struct(req); valerr != nil {
		return nil, apierror.FromValidationError(valerr)
	}

	if !actor.Permissions.HasEffective(entity.PermissionEditNotes) {
		return nil, apierror.UserMissingPermsError
	}

	note, err := n.NoteRepo.FindByID(noteId)
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

	resp := toNoteResponse(note, false)
	go n.dispatchNoteUpdateEvent(resp)
	return resp, nil
}

func (n *DefaultNoteService) DeleteNote(actor *entity.User, noteId int) apierror.ErrorResponse {
	if !actor.Permissions.HasEffective(entity.PermissionDeleteNotes) {
		return apierror.UserMissingPermsError
	}

	note, err := n.NoteRepo.FindByID(noteId)
	if err != nil {
		log.Errorf("failed to fetch note: %v", err)
		return apierror.InternalServerError
	}

	if note == nil {
		return apierror.NotFoundError
	}

	err = deleteBucketObject(n.S3, note)
	if err != nil {
		log.Errorf("failed to delete note: %v", err)
		return apierror.InternalServerError
	}

	err = n.NoteRepo.Delete(note)
	if err != nil {
		log.Errorf("failed to delete note: %v", err)
		return apierror.InternalServerError
	}

	go n.dispatchNoteDeleteEvent(note.ID)
	return nil
}

func (n *DefaultNoteService) dispatchNoteCreateEvent(note *NoteResponse) {
	n.WSService.Broadcast(context.Background(), &events.CreateNoteEvent{
		NoteResponse: note,
	})
}

func (n *DefaultNoteService) dispatchNoteUpdateEvent(note *NoteResponse) {
	n.WSService.Broadcast(context.Background(), &events.UpdateNoteEvent{
		NoteResponse: note,
	})
}

func (n *DefaultNoteService) dispatchNoteDeleteEvent(noteID int) {
	n.WSService.Broadcast(context.Background(), &events.DeleteNoteEvent{
		NoteID: noteID,
	})
}

// handleNoteUpload tries to upload the note to S3 and already generates the
// new UUID name of the file object that will persist.
func handleNoteUpload(s3 storage.S3Client, fileheader *multipart.FileHeader) (string, int, apierror.ErrorResponse) {
	ext := filepath.Ext(fileheader.Filename)
	bytes, apierr := readNoteFile(fileheader)
	if apierr != nil {
		return "", 0, apierr
	}

	filename := uuid.NewString() + ext
	err := s3.UploadFile(bytes, storage.PathAttachments+filename)
	if err != nil {
		log.Errorf("failed to upload file: %v", err)
		return "", 0, apierror.InternalServerError
	}
	return filename, len(bytes), nil
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
	var content string
	if note.NoteType == entity.NoteTypeReference || forceContent {
		content = note.Content
	}

	return &NoteResponse{
		ID:          note.ID,
		Name:        note.Name,
		Content:     content,
		Tags:        toTagsArray(note.Tags),
		Visibility:  note.Visibility,
		NoteType:    string(note.NoteType),
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
func deleteBucketObject(bucket storage.S3Client, note *entity.Note) error {
	fileName := note.Content

	// If the note is a text/chart file, then there is nothing to delete from
	// AWS, as we only store files on S3.
	if note.NoteType != entity.NoteTypeReference {
		return nil
	}

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

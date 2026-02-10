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
	"simplenotes/cmd/internal/contract"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/domain/events"
	"simplenotes/cmd/internal/domain/policy"
	"simplenotes/cmd/internal/infrastructure/aws/storage"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
	"strings"
)

type NoteRepository interface {
	FindAll(withPrivate bool) ([]*entity.Note, error)
	FindByID(id int) (*entity.Note, error)
	Save(note *entity.Note) error
	Delete(note *entity.Note) error
}

type NoteService struct {
	NoteRepo   NoteRepository
	UserRepo   UserRepository
	WSService  *WebSocketService
	S3         storage.S3Client
	Validate   *validator.Validate
	NotePolicy *policy.NotePolicy
}

func NewNoteService(
	noteRepo NoteRepository,
	userRepo UserRepository,
	wsService *WebSocketService,
	s3 storage.S3Client,
	validate *validator.Validate,
	notePolicy *policy.NotePolicy,
) *NoteService {
	return &NoteService{
		NoteRepo:   noteRepo,
		UserRepo:   userRepo,
		WSService:  wsService,
		S3:         s3,
		Validate:   validate,
		NotePolicy: notePolicy,
	}
}

func (n *NoteService) GetAllNotes(actor *entity.User) ([]*contract.NoteResponse, apierror.ErrorResponse) {
	canSeeHidden := actor.Permissions.HasEffective(entity.PermissionSeeHiddenNotes)
	notes, err := n.NoteRepo.FindAll(canSeeHidden)
	if err != nil {
		log.Errorf("failed to fetch notes: %v", err)
		return nil, apierror.InternalServerError
	}

	resp := make([]*contract.NoteResponse, len(notes))
	for i, note := range notes {
		resp[i] = toNoteResponse(note, false)
	}
	return resp, nil
}

func (n *NoteService) GetNoteByID(actor *entity.User, noteId int) (*contract.NoteResponse, apierror.ErrorResponse) {
	note, err := n.NoteRepo.FindByID(noteId)
	if err != nil {
		log.Errorf("failed to fetch note: %v", err)
		return nil, apierror.InternalServerError
	}

	apierr := n.NotePolicy.CanSee(note, actor)
	if apierr != nil {
		return nil, apierr
	}

	if note == nil {
		return nil, apierror.NotFoundError
	}
	return toNoteResponse(note, true), nil
}

func (n *NoteService) CreateTextNote(actor *entity.User, req *contract.TextNoteRequest) (*contract.NoteResponse, apierror.ErrorResponse) {
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
		Visibility:  entity.NoteVisibility(req.Visibility),
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

func (n *NoteService) CreateFileNote(actor *entity.User, req *contract.NoteRequest, fileHeader *multipart.FileHeader) (*contract.NoteResponse, apierror.ErrorResponse) {
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
		Visibility:  entity.NoteVisibility(req.Visibility),
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

func (n *NoteService) UpdateNote(actor *entity.User, noteId int, req *contract.UpdateNoteRequest) (*contract.NoteResponse, apierror.ErrorResponse) {
	if !actor.Permissions.HasEffective(entity.PermissionEditNotes) {
		return nil, apierror.UserMissingPermsError
	}

	utils.Sanitize(req)
	if valerr := n.Validate.Struct(req); valerr != nil {
		return nil, apierror.FromValidationError(valerr)
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
		note.Visibility = entity.NoteVisibility(*req.Visibility)
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

func (n *NoteService) DeleteNote(actor *entity.User, noteId int) apierror.ErrorResponse {
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

func (n *NoteService) dispatchNoteCreateEvent(note *contract.NoteResponse) {
	n.WSService.Broadcast(context.Background(), &events.NoteCreated{
		NoteResponse: note,
	})
}

func (n *NoteService) dispatchNoteUpdateEvent(note *contract.NoteResponse) {
	n.WSService.Broadcast(context.Background(), &events.NoteUpdated{
		NoteResponse: note,
	})
}

func (n *NoteService) dispatchNoteDeleteEvent(noteID int) {
	n.WSService.Broadcast(context.Background(), &events.NoteDeleted{
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
	if fileHeader.Size > contract.MaxNoteFileSizeBytes {
		return apierror.NewNoteContentTooLargeError(contract.MaxNoteFileSizeBytes)
	}

	if strings.TrimSpace(fileHeader.Filename) == "" {
		return apierror.MissingFileNameError
	}

	if ext, ok := utils.CheckFileExt(fileHeader.Filename, contract.ValidNoteFileTypes); !ok {
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

func toNoteResponse(note *entity.Note, forceContent bool) *contract.NoteResponse {
	var content string
	if note.NoteType == entity.NoteTypeReference || forceContent {
		content = note.Content
	}

	return &contract.NoteResponse{
		ID:          note.ID,
		Name:        note.Name,
		Content:     content,
		Tags:        toTagsArray(note.Tags),
		Visibility:  string(note.Visibility),
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

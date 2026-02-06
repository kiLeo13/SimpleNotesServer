package handler

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"simplenotes/cmd/internal/contract"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

// NoteService interface updated to accept *entity.User instead of strings.
// This allows the service to check permissions without hitting the DB again.
type NoteService interface {
	GetAllNotes() ([]*contract.NoteResponse, apierror.ErrorResponse)
	GetNoteByID(actor *entity.User, noteId int) (*contract.NoteResponse, apierror.ErrorResponse)
	CreateTextNote(actor *entity.User, req *contract.TextNoteRequest) (*contract.NoteResponse, apierror.ErrorResponse)
	CreateFileNote(actor *entity.User, req *contract.NoteRequest, fileHeader *multipart.FileHeader) (*contract.NoteResponse, apierror.ErrorResponse)
	UpdateNote(actor *entity.User, noteId int, req *contract.UpdateNoteRequest) (*contract.NoteResponse, apierror.ErrorResponse)
	DeleteNote(actor *entity.User, noteId int) apierror.ErrorResponse
}

type DefaultNoteRoute struct {
	NoteService NoteService
}

func NewNoteDefault(noteService NoteService) *DefaultNoteRoute {
	return &DefaultNoteRoute{NoteService: noteService}
}

func (n *DefaultNoteRoute) GetNotes(c echo.Context) error {
	notes, err := n.NoteService.GetAllNotes()
	if err != nil {
		return c.JSON(err.Code(), err)
	}

	resp := echo.Map{"notes": notes}
	return c.JSON(http.StatusOK, &resp)
}

func (n *DefaultNoteRoute) GetNote(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, apierror.NewInvalidParamTypeError("id", "int"))
	}

	note, apierr := n.NoteService.GetNoteByID(user, id)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusOK, note)
}

func (n *DefaultNoteRoute) CreateNote(c echo.Context) error {
	contentType := c.Request().Header.Get(echo.HeaderContentType)

	if strings.HasPrefix(contentType, echo.MIMEApplicationJSON) {
		return n.createFromText(c)
	}

	if strings.HasPrefix(contentType, echo.MIMEMultipartForm) {
		return n.createFromFile(c)
	}

	mediaTypeError := apierror.InvalidMediaTypeError
	return c.JSON(http.StatusUnsupportedMediaType, &mediaTypeError)
}

func (n *DefaultNoteRoute) UpdateNote(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, apierror.NewInvalidParamTypeError("id", "int"))
	}

	var req contract.UpdateNoteRequest
	if err = c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apierror.MalformedBodyError)
	}

	newNote, apierr := n.NoteService.UpdateNote(user, id, &req)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusOK, &newNote)
}

func (n *DefaultNoteRoute) DeleteNote(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, apierror.NewInvalidParamTypeError("id", "int"))
	}

	serr := n.NoteService.DeleteNote(user, id)
	if serr != nil {
		return c.JSON(serr.Code(), serr)
	}
	return c.NoContent(http.StatusOK)
}

func (n *DefaultNoteRoute) createFromText(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	var req contract.TextNoteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apierror.MalformedBodyError)
	}

	note, apierr := n.NoteService.CreateTextNote(user, &req)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusCreated, &note)
}

func (n *DefaultNoteRoute) createFromFile(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	jsonPayload := strings.TrimSpace(c.FormValue("json_payload"))
	if jsonPayload == "" {
		return c.JSON(http.StatusBadRequest, apierror.FormJSONRequiredError)
	}

	var req contract.NoteRequest
	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return c.JSON(http.StatusBadRequest, apierror.MalformedBodyError)
	}

	fileHeader, err := c.FormFile("content")
	if err != nil {
		return c.JSON(http.StatusBadRequest, apierror.MissingNoteFileError)
	}

	note, apierr := n.NoteService.CreateFileNote(user, &req, fileHeader)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusCreated, &note)
}

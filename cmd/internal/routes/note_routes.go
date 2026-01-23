package routes

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"mime/multipart"
	"net/http"
	"simplenotes/cmd/internal/service"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
	"strconv"
	"strings"
)

type NoteService interface {
	GetAllNotes() ([]*service.NoteResponse, apierror.ErrorResponse)
	GetNoteByID(id int) (*service.NoteResponse, apierror.ErrorResponse)
	CreateFileNote(req *service.NoteRequest, fileHeader *multipart.FileHeader, issuerId string) (*service.NoteResponse, apierror.ErrorResponse)
	CreateTextNote(req *service.TextNoteRequest, issuerId string) (*service.NoteResponse, apierror.ErrorResponse)
	UpdateNote(id int, userSub string, req *service.UpdateNoteRequest) (*service.NoteResponse, apierror.ErrorResponse)
	DeleteNote(noteId int, issuerId string) apierror.ErrorResponse
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

	resp := echo.Map{
		"notes": notes,
	}
	return c.JSON(http.StatusOK, &resp)
}

func (n *DefaultNoteRoute) GetNote(c echo.Context) error {
	rawId := c.Param("id")
	id, err := strconv.Atoi(rawId)
	if err != nil {
		errResp := apierror.NewSimple(400, "ID is not a number")
		return c.JSON(errResp.Status, errResp)
	}

	note, apierr := n.NoteService.GetNoteByID(id)
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
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		errResp := apierror.NewSimple(400, "ID is not a number")
		return c.JSON(errResp.Status, errResp)
	}
	var req service.UpdateNoteRequest
	if err = c.Bind(&req); err != nil {
		return c.JSON(400, apierror.MalformedBodyError)
	}

	token, err := utils.ParseTokenDataCtx(c)
	if err != nil {
		return c.JSON(401, apierror.InvalidAuthTokenError)
	}

	newNote, apierr := n.NoteService.UpdateNote(id, token.Sub, &req)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusOK, &newNote)
}

func (n *DefaultNoteRoute) DeleteNote(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		errResp := apierror.NewSimple(400, "ID is not a number")
		return c.JSON(errResp.Status, errResp)
	}

	data, err := utils.ParseTokenDataCtx(c)
	if err != nil {
		return c.JSON(401, apierror.InvalidAuthTokenError)
	}

	serr := n.NoteService.DeleteNote(id, data.Sub)
	if serr != nil {
		return c.JSON(serr.Code(), serr)
	}
	return c.NoContent(http.StatusOK)
}

func (n *DefaultNoteRoute) createFromText(c echo.Context) error {
	var req service.TextNoteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, apierror.MalformedBodyError)
	}

	data, err := utils.ParseTokenDataCtx(c)
	if err != nil {
		return c.JSON(401, apierror.InvalidAuthTokenError)
	}

	note, apierr := n.NoteService.CreateTextNote(&req, data.Sub)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusCreated, &note)
}

func (n *DefaultNoteRoute) createFromFile(c echo.Context) error {
	jsonPayload := strings.TrimSpace(c.FormValue("json_payload"))
	if jsonPayload == "" {
		return c.JSON(400, apierror.FormJSONRequiredError)
	}

	var req service.NoteRequest
	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return c.JSON(400, apierror.MalformedBodyError)
	}

	data, err := utils.ParseTokenDataCtx(c)
	if err != nil {
		return c.JSON(401, apierror.InvalidAuthTokenError)
	}

	fileHeader, err := c.FormFile("content")
	if err != nil {
		return c.JSON(400, apierror.MissingNoteFileError)
	}

	note, apierr := n.NoteService.CreateFileNote(&req, fileHeader, data.Sub)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusCreated, &note)
}

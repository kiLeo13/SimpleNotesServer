package routes

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"mime/multipart"
	"net/http"
	"simplenotes/cmd/internal/service"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
	"strconv"
)

type NoteService interface {
	GetAllNotes() ([]*service.NoteResponse, apierror.ErrorResponse)
	CreateNote(req *service.NoteRequest, fileHeader *multipart.FileHeader, issuerId string) (*service.NoteResponse, apierror.ErrorResponse)
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

func (n *DefaultNoteRoute) CreateNote(c echo.Context) error {
	fmt.Println("Received")
	var req service.NoteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, apierror.MalformedBodyError)
	}

	fmt.Println("Parsing token...")
	data, err := utils.ParseTokenData(c)
	if err != nil {
		return c.JSON(401, apierror.InvalidAuthTokenError)
	}
	fmt.Println("Parsed")

	fmt.Println("Getting content")
	fileHeader, err := c.FormFile("content")
	if err != nil {
		return c.JSON(400, apierror.MissingNoteFileError)
	}
	fmt.Println("Got content")

	fmt.Println("Calling creator")
	note, apierr := n.NoteService.CreateNote(&req, fileHeader, data.Sub)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	fmt.Println("Got response")
	return c.JSON(http.StatusCreated, &note)
}

func (n *DefaultNoteRoute) DeleteNote(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		errResp := apierror.NewSimple(400, "ID is not a number")
		return c.JSON(errResp.Status, errResp)
	}

	data, err := utils.ParseTokenData(c)
	if err != nil {
		return c.JSON(401, apierror.InvalidAuthTokenError)
	}

	serr := n.NoteService.DeleteNote(id, data.Sub)
	if serr != nil {
		return c.JSON(serr.Code(), serr)
	}
	return c.NoContent(http.StatusOK)
}

package routes

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"simplenotes/internal/service"
	"strconv"
)

type NoteService interface {
	GetAllNotes() ([]*service.NoteResponse, *service.APIError)
	CreateNote(req *service.NoteRequest) (*service.NoteResponse, *service.APIError)
	DeleteNote(noteId int) *service.APIError
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
		return c.JSON(err.Status, err)
	}

	resp := echo.Map{
		"notes": notes,
	}
	return c.JSON(http.StatusOK, &resp)
}

func (n *DefaultNoteRoute) CreateNote(c echo.Context) error {
	var req service.NoteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, service.MalformedJSONError)
	}

	note, err := n.NoteService.CreateNote(&req)
	if err != nil {
		return c.JSON(err.Status, err)
	}
	return c.JSON(http.StatusCreated, &note)
}

func (n *DefaultNoteRoute) DeleteNote(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		errResp := service.NewError(400, "ID is not a number")
		return c.JSON(errResp.Status, errResp)
	}

	serr := n.NoteService.DeleteNote(id)
	if serr != nil {
		return c.JSON(serr.Status, serr)
	}
	return c.NoContent(http.StatusOK)
}

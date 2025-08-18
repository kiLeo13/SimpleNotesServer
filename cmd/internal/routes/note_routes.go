package routes

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"simplenotes/internal/service"
	"strconv"
)

func GetNotes(c echo.Context) error {
	notes, err := service.GetAllNotes()
	if err != nil {
		return c.JSON(err.Status, err)
	}

	resp := echo.Map{
		"notes": notes,
	}
	return c.JSON(http.StatusOK, &resp)
}

func CreateNote(c echo.Context) error {
	var req service.NoteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, service.ErrorMalformedJSON)
	}

	note, err := service.CreateNote(&req)
	if err != nil {
		return c.JSON(err.Status, err)
	}
	return c.JSON(http.StatusCreated, &note)
}

func DeleteNote(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		errResp := service.NewError(400, "ID is not a number")
		return c.JSON(errResp.Status, errResp)
	}

	serr := service.DeleteNote(id)
	if serr != nil {
		return c.JSON(serr.Status, serr)
	}
	return c.NoContent(http.StatusOK)
}

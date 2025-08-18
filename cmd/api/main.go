package main

import (
	"github.com/labstack/echo/v4/middleware"
	"simplenotes/internal/domain/sqlite"
	"simplenotes/internal/routes"

	"github.com/labstack/echo/v4"
)

func main() {
	err := sqlite.Init()
	if err != nil {
		panic(err)
	}

	e := echo.New()
	e.Use(middleware.CORS())

	e.GET("/api/notes", routes.GetNotes)
	e.POST("/api/notes", routes.CreateNote)
	e.DELETE("/api/notes/:id", routes.DeleteNote)

	if err := e.Start(":7070"); err != nil {
		panic(err)
	}
}

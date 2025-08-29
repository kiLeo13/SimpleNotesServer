package main

import (
	"github.com/labstack/echo/v4/middleware"
	"os"
	"simplenotes/internal/domain/sqlite"
	cognitoclient "simplenotes/internal/infrastructure/aws/cognito"
	"simplenotes/internal/routes"

	"github.com/labstack/echo/v4"
)

func main() {
	err := sqlite.Init()
	if err != nil {
		panic(err)
	}

	// Init cognito client
	appClientId := os.Getenv("COGNITO_CLIENT_ID")
	err = cognitoclient.InitCognitoClient(appClientId)
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

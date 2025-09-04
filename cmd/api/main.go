package main

import (
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"os"
	"simplenotes/internal/domain/sqlite"
	"simplenotes/internal/domain/sqlite/repository"
	cognitoclient "simplenotes/internal/infrastructure/aws/cognito"
	"simplenotes/internal/routes"
	"simplenotes/internal/service"
	"simplenotes/internal/validators"
)

func main() {
	validate := validator.New()
	registerValidators(validate)

	// Init env vars
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	// Init SQLite
	db, err := sqlite.Init()
	if err != nil {
		panic(err)
	}

	// Init cognito client
	cognitoClientId := os.Getenv("AWS_COGNITO_CLIENT_ID")
	cogClient, err := cognitoclient.InitCognitoClient(cognitoClientId)
	if err != nil {
		panic(err)
	}

	// Gettings repos
	noteRepo := repository.NewNoteRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Getting services
	userService := service.NewUserService(userRepo, validate, cogClient)
	noteService := service.NewNoteService(noteRepo, validate)

	// Gettings routes
	noteRoutes := routes.NewNoteDefault(noteService)
	userRoutes := routes.NewUserDefault(userService)

	e := echo.New()
	e.Use(middleware.CORS())

	// Notes
	e.GET("/api/notes", noteRoutes.GetNotes)
	e.POST("/api/notes", noteRoutes.CreateNote)
	e.DELETE("/api/notes/:id", noteRoutes.DeleteNote)

	// Users
	e.GET("/api/users/:id", userRoutes.GetUser)
	e.POST("/api/users/query", userRoutes.QueryUsers)
	e.POST("/api/users", userRoutes.CreateUser)
	e.POST("/api/users/login", userRoutes.CreateLogin)
	e.POST("/api/users/confirms", userRoutes.ConfirmSignup)
	e.POST("/api/users/confirms/resend", userRoutes.ResendConfirmation)

	if err := e.Start(":7070"); err != nil {
		panic(err)
	}
}

func registerValidators(validate *validator.Validate) {
	_ = validate.RegisterValidation("x-password", validators.PasswordValidator, false)
}

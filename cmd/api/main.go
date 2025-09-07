package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"os"
	"simplenotes/cmd/internal/domain/sqlite"
	"simplenotes/cmd/internal/domain/sqlite/repository"
	"simplenotes/cmd/internal/infrastructure/aws/cognito"
	"simplenotes/cmd/internal/routes"
	"simplenotes/cmd/internal/service"
	"simplenotes/cmd/internal/utils/validators"
)

const envVarsPrefix = "/simplenotes/prod/"

func main() {
	validate := validator.New()
	registerValidators(validate)

	// Loads env vars depending on environment
	if os.Getenv("GO_ENV") == "production" {
		loadProdEnv() // AWS SSM Parameter Store
	} else {
		// Loads from .env
		err := godotenv.Load()
		if err != nil {
			panic(err)
		}
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

	// Docker Compose healthcheck
	e.GET("/health", healthCheckRoute)

	if err := e.Start(":7070"); err != nil {
		panic(err)
	}
}

func registerValidators(validate *validator.Validate) {
	_ = validate.RegisterValidation("hasupper", validators.HasUpper, false)
	_ = validate.RegisterValidation("haslower", validators.HasLower, false)
	_ = validate.RegisterValidation("hasdigit", validators.HasDigit, false)
	_ = validate.RegisterValidation("hasspecial", validators.HasSpecial, false)
}

func loadProdEnv() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	client := ssm.NewFromConfig(cfg)
	out, err := client.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
		Path:           aws.String(envVarsPrefix),
		WithDecryption: aws.Bool(true),
		Recursive:      aws.Bool(true),
	})
	if err != nil {
		log.Fatalf("unable to load prod environment, %v", err)
	}

	prefixLength := len(envVarsPrefix)
	// Export vars
	for _, param := range out.Parameters {
		key := (*param.Name)[prefixLength:]
		value := *param.Value
		enverr := os.Setenv(key, value)
		if enverr != nil {
			log.Fatalf("unable to set environment variable, %v", enverr)
		}
	}
	log.Debugf("loaded %d prod environment variables", len(out.Parameters))
}

func healthCheckRoute(c echo.Context) error {
	return c.String(200, "OK")
}

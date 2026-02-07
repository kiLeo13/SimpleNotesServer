package main

import (
	"context"
	"os"
	"simplenotes/cmd/internal/domain/policy"
	"simplenotes/cmd/internal/domain/sqlite"
	"simplenotes/cmd/internal/domain/sqlite/repository"
	"simplenotes/cmd/internal/http/handler"
	mdlware "simplenotes/cmd/internal/http/middleware"
	cognitoclient "simplenotes/cmd/internal/infrastructure/aws/cognito"
	"simplenotes/cmd/internal/infrastructure/aws/storage"
	"simplenotes/cmd/internal/infrastructure/aws/websocket"
	"simplenotes/cmd/internal/service"
	"simplenotes/cmd/internal/service/jobs"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/validators"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

const envVarsPrefix = "/simplenotes/prod/"

func main() {
	validate := validator.New()
	registerValidators(validate)

	if os.Getenv("GO_ENV") == "production" {
		loadProdEnv()
	} else {
		if err := godotenv.Load(); err != nil {
			panic(err)
		}
	}

	// Infra Init
	db, err := sqlite.Init()
	if err != nil {
		panic(err)
	}

	// --- Cognito/Auth Init ---
	appClientID := os.Getenv("AWS_COGNITO_CLIENT_ID")
	region := os.Getenv("AWS_COGNITO_REGION")
	poolID := os.Getenv("AWS_COGNITO_USER_POOL_ID")
	cogClient, err := cognitoclient.InitCognitoClient(appClientID, region, poolID)
	if err != nil {
		panic(err)
	}

	if err = utils.InitJWKS(region, poolID); err != nil {
		panic(err)
	}

	// --- Storage Init ---
	s3Client, err := storage.NewStorageClient()
	if err != nil {
		panic(err)
	}

	wsEndpoint := os.Getenv("AWS_WS_GATEWAY_ENDPOINT")
	wsRegion := os.Getenv("AWS_WS_GATEWAY_REGION")
	wsClient, err := websocket.NewAWSGatewayClient(context.Background(), wsEndpoint, wsRegion)
	if err != nil {
		panic(err)
	}

	// Domain & Service Wiring
	userPolicy := policy.NewUserPolicy()

	connRepo := repository.NewConnectionRepository(db)
	noteRepo := repository.NewNoteRepository(db)
	userRepo := repository.NewUserRepository(db)

	connService := service.NewWebSocketService(connRepo, wsClient)
	userService := service.NewUserService(userRepo, validate, connService, cogClient, userPolicy)
	noteService := service.NewNoteService(noteRepo, userRepo, connService, s3Client, validate)

	connRoutes := handler.NewWSDefault(connService)
	noteRoutes := handler.NewNoteDefault(noteService)
	userRoutes := handler.NewUserDefault(userService)

	// --- Background Jobs ---
	cleaner := jobs.NewConnectionCleaner(connService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go cleaner.Start(ctx)

	// --- Middleware Setup ---
	authMiddleware := mdlware.NewAuthMiddleware(&mdlware.AuthMiddlewareConfig{
		UserRepo: userRepo,
	})

	// --- Server Setup ---
	e := echo.New()

	e.Use(middleware.CORS())
	e.Use(middleware.BodyLimit("30M"))
	e.Use(middleware.Recover())

	// --- Register Routes ---
	registerRoutes(e, noteRoutes, userRoutes, connRoutes, authMiddleware)

	if err = e.Start(":7070"); err != nil {
		panic(err)
	}
}

// registerRoutes separates the routing logic from the wiring logic.
func registerRoutes(
	e *echo.Echo,
	noteH *handler.DefaultNoteRoute,
	userH *handler.DefaultUserRoute,
	wsH *handler.DefaultWSRoute,
	authMiddleware echo.MiddlewareFunc,
) {
	// --- Public Routes (Unauthenticated) ---
	public := e.Group("/api")

	e.GET("/health", healthCheckRoute)

	// User Auth & Registration
	public.POST("/users/login", userH.CreateLogin)
	public.POST("/users", userH.CreateUser) // Registration is public
	public.POST("/users/check-email", userH.CheckEmail)
	public.POST("/users/confirms", userH.ConfirmSignup)
	public.POST("/users/confirms/resend", userH.ResendConfirmation)

	// --- Protected Routes ---
	protected := e.Group("/api")
	protected.Use(authMiddleware)

	// Notes
	protected.GET("/notes", noteH.GetNotes)
	protected.GET("/notes/:id", noteH.GetNote)
	protected.POST("/notes", noteH.CreateNote)
	protected.PATCH("/notes/:id", noteH.UpdateNote)
	protected.DELETE("/notes/:id", noteH.DeleteNote)

	// Users
	protected.GET("/users", userH.GetUsers)
	protected.GET("/users/:id", userH.GetUser)
	protected.PATCH("/users/:id", userH.UpdateUser)
	protected.DELETE("/users/:id", userH.DeleteUser)

	// --- WebSocket ---
	ws := e.Group("/ws")

	ws.POST("/connect", wsH.HandleConnect, authMiddleware)
	ws.POST("/default", wsH.HandleMessage)
	ws.POST("/disconnect", wsH.HandleDisconnect)
}

func registerValidators(validate *validator.Validate) {
	_ = validate.RegisterValidation("hasupper", validators.HasUpper)
	_ = validate.RegisterValidation("haslower", validators.HasLower)
	_ = validate.RegisterValidation("hasdigit", validators.HasDigit)
	_ = validate.RegisterValidation("hasspecial", validators.HasSpecial)
	_ = validate.RegisterValidation("nodupes", validators.NoDupes)
	_ = validate.RegisterValidation("nospaces", validators.NoWhiteSpaces)
}

func loadProdEnv() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-2"))
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
	for _, param := range out.Parameters {
		key := (*param.Name)[prefixLength:]
		value := *param.Value
		if err := os.Setenv(key, value); err != nil {
			log.Fatalf("unable to set environment variable, %v", err)
		}
	}
	log.Debugf("loaded %d prod environment variables", len(out.Parameters))
}

func healthCheckRoute(c echo.Context) error {
	return c.String(200, "OK")
}

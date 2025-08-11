package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"facebook-bot/config"
	"facebook-bot/handlers"
	"facebook-bot/services"
	"facebook-bot/webhooks"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found")
	}

	// Initialize structured logger
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(logHandler))

	// Check for Claude debug mode
	if os.Getenv("DEBUG_CLAUDE") == "true" {
		slog.Info("🔍 CLAUDE DEBUG MODE ENABLED")
		slog.Info("All messages sent to Claude AI will be printed to console")
		slog.Info("To disable, unset DEBUG_CLAUDE environment variable")
	}

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := services.InitMongoDB(ctx, cfg.MongoURI)
	if err != nil {
		slog.Error("Failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer db.Disconnect(ctx)

	// Initialize services
	services.InitServices(db, cfg.DatabaseName)

	// Initialize vector database
	if err := services.InitVectorDB(ctx); err != nil {
		slog.Error("Failed to initialize vector DB", "error", err)
		// Continue anyway - vector DB is optional
	}

	// Initialize CRM data before starting server
	slog.Info("Initializing CRM data...")
	if err := services.InitializeCRMData(ctx); err != nil {
		slog.Error("Failed to initialize CRM data", "error", err)
		// Continue anyway - we can still serve requests
	}

	// Start CRM update scheduler
	schedulerCtx, cancelScheduler := context.WithCancel(context.Background())
	defer cancelScheduler()
	services.StartCRMUpdateScheduler(schedulerCtx)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			slog.Error("Request error", "error", err, "status", code)
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path}\n",
	}))

	// Register webhook routes
	webhooks.RegisterRoutes(app, cfg)

	// Register admin routes (for testing)
	admin := app.Group("/admin")

	// Company management endpoints
	admin.Post("/companies", handlers.CreateCompany)
	admin.Get("/companies/:companyID", handlers.GetCompany)
	admin.Post("/companies/:companyID/pages", handlers.AddPageToCompany)

	// Voyage embedding configuration endpoints
	admin.Put("/companies/:companyID/voyage", handlers.UpdateVoyageConfig)
	admin.Post("/companies/:companyID/voyage/test", handlers.TestVoyageEmbedding)
	admin.Post("/companies/:companyID/reindex", handlers.ReindexDocuments)

	// Keep old endpoints for backward compatibility
	admin.Put("/companies/:companyID/embedding", handlers.UpdateEmbeddingConfig)
	admin.Post("/companies/:companyID/embedding/test", handlers.TestEmbedding)

	// RAG debugging endpoints
	admin.Post("/companies/:companyID/rag/test", handlers.TestRAGRetrieval)
	admin.Get("/companies/:companyID/documents", handlers.GetStoredDocuments)

	// User management endpoints
	admin.Post("/users", handlers.CreateUser)
	admin.Get("/users/:userID", handlers.GetUser)
	admin.Get("/companies/:companyID/users", handlers.GetCompanyUsers)
	admin.Put("/users/:userID/role", handlers.UpdateUserRole)
	admin.Post("/users/:userID/regenerate-api-key", handlers.RegenerateUserAPIKey)

	// Utility endpoints
	admin.Post("/test-webhook", handlers.TestWebhookConnection)
	admin.Get("/roles", handlers.GetRolePermissions)

	// Testing endpoints (simplified)
	test := app.Group("/test")
	test.Post("/add-user", handlers.TestCreateUser)
	test.Post("/add-company", handlers.TestCreateCompany)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "facebook-bot",
		})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Server starting", "port", port)
	if err := app.Listen(":" + port); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

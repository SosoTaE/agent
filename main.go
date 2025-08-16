package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"facebook-bot/config"
	"facebook-bot/handlers"
	"facebook-bot/middleware"
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

	// Initialize session store for authentication
	services.InitSession()

	// Create indexes for customers collection
	if err := services.CreateIndexesForCustomers(ctx); err != nil {
		slog.Error("Failed to create customer indexes", "error", err)
		// Continue anyway - the app can still work without indexes
	}

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

	// CORS configuration - Allow frontend development server
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173, http://localhost:3000, http://localhost:5174",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length, Content-Type, X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset",
		MaxAge:           86400, // 24 hours
	}))

	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path}\n",
	}))

	// Register webhook routes
	webhooks.RegisterRoutes(app, cfg)

	auth := app.Group("/auth")
	auth.Post("/login", handlers.Login)
	auth.Post("/logout", handlers.Logout)
	auth.Get("/me", handlers.GetCurrentUser)
	auth.Get("/check", handlers.CheckSession)

	// Register admin routes (protected)
	admin := app.Group("/admin", middleware.RequireAuth)

	// Company management endpoints (all users can view company info)
	admin.Get("/company", handlers.GetCompany)

	// Super admin endpoint to create new companies (requires special privileges)
	admin.Post("/company", handlers.CreateCompany) // TODO: Add super admin middleware when needed

	// Company admin only endpoints
	admin.Post("/company/pages", middleware.RequireCompanyAdmin, handlers.AddPageToCompany)
	admin.Post("/users", middleware.RequireCompanyAdmin, handlers.CreateUser)
	admin.Put("/users/:userID/role", middleware.RequireCompanyAdmin, handlers.UpdateUserRole)

	// User viewing endpoints (all authenticated users)
	admin.Get("/users", handlers.GetCompanyUsers)
	admin.Get("/users/:userID", handlers.GetUser)

	// Dashboard API endpoints (protected)
	dashboard := app.Group("/api/dashboard", middleware.RequireAuth, middleware.ExtractCompanyPages)

	// get pages which are registered in company
	dashboard.Get("/pages", handlers.GetCompanyPages)
	dashboard.Get("/comments/post/:postID", middleware.ValidatePostAccess, handlers.GetPostComments)
	dashboard.Get("/comments/post/:postID/hierarchical", middleware.ValidatePostAccess, handlers.GetPostCommentsHandler) // New hierarchical endpoint
	dashboard.Get("/comments/:commentID/replies", handlers.GetCommentWithRepliesHandler)                                 // Get specific comment with replies
	dashboard.Get("/comments/post/:postID/threaded", middleware.ValidatePostAccess, handlers.GetThreadedCommentsHandler) // Get threaded view
	dashboard.Get("/comments", handlers.GetCompanyComments)
	dashboard.Get("/stats", handlers.GetCommentStats)
	dashboard.Get("/name-changes", handlers.GetNameChanges)
	dashboard.Get("/sender/:senderID/history", handlers.GetSenderHistory)
	dashboard.Get("/conversations", handlers.GetUserConversations)
	dashboard.Get("/messages/:senderID", handlers.GetUserMessages)
	dashboard.Get("/messages", handlers.GetChatMessages)                                  // Get all messages with pagination and filtering
	dashboard.Get("/messages/page/:pageID", handlers.GetAllMessagesByPage)                // Get all messages for a page
	dashboard.Get("/messages/page/:pageID/conversations", handlers.GetPageConversations)  // Get all conversations for a page
	dashboard.Get("/messages/page/:pageID/chats", handlers.GetChatIDs)                    // Get all chat IDs for a page
	dashboard.Get("/messages/chat/:chatID", handlers.GetMessagesByChatID)                 // Get messages by chat ID
	dashboard.Get("/messages/conversation/:customerID", handlers.GetCustomerConversation) // Get customer conversation with bot

	// Customer endpoints
	dashboard.Get("/customers", handlers.GetCustomers)                                      // Get customers list
	dashboard.Get("/customers/:customerID", handlers.GetCustomerDetails)                    // Get specific customer
	dashboard.Get("/customers/search", handlers.SearchCustomers)                            // Search customers
	dashboard.Get("/customers/stats", handlers.GetCustomerStats)                            // Get customer statistics
	dashboard.Put("/customers/:customerID/stop", handlers.UpdateCustomerStopStatus)         // Update customer stop status
	dashboard.Post("/customers/:customerID/toggle-stop", handlers.ToggleCustomerStopStatus) // Toggle customer stop status
	dashboard.Post("/customers/:customerID/message", handlers.SendMessageToCustomer)        // Send message to customer

	dashboard.Get("/posts", handlers.GetPostsList)
	dashboard.Get("/posts/company", handlers.GetPostIDsByCompanyHandler) // Get posts for company

	// WebSocket endpoint (requires authentication)
	dashboard.Get("/ws", handlers.WebSocketUpgrade, websocket.New(handlers.HandleWebSocket))

	// Public API endpoints (no auth required)
	api := app.Group("/api")

	// Comments endpoints
	api.Get("/comments/post/:postID", handlers.GetPostCommentsHandler)            // Public endpoint for hierarchical comments
	api.Get("/comments/:commentID", handlers.GetCommentWithRepliesHandler)        // Public endpoint for single comment with replies
	api.Get("/comments/post/:postID/debug", handlers.GetPostCommentsDebugHandler) // Debug endpoint

	// Posts endpoints
	api.Get("/posts", handlers.GetAllPostIDsHandler)                 // Get all post IDs
	api.Get("/posts/paginated", handlers.GetPostIDsPaginatedHandler) // Get paginated posts
	api.Get("/posts/page/:pageID", handlers.GetPostIDsByPageHandler) // Get posts by page
	api.Get("/posts/stats", handlers.GetPostsStatsHandler)           // Get posts statistics

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

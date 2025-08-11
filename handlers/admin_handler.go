package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"facebook-bot/models"
	"facebook-bot/services"
)

// CreateCompanyRequest represents the request body for creating a company
type CreateCompanyRequest struct {
	CompanyID     string           `json:"company_id" validate:"required"`
	CompanyName   string           `json:"company_name" validate:"required"`
	Pages         []PageRequest    `json:"pages" validate:"required,min=1"`
	AppSecret     string           `json:"app_secret" validate:"required"`
	ClaudeAPIKey  string           `json:"claude_api_key" validate:"required"`
	ClaudeModel   string           `json:"claude_model,omitempty"`
	SystemPrompt  string           `json:"system_prompt,omitempty"`
	MaxTokens     int              `json:"max_tokens,omitempty"`
	ResponseDelay int              `json:"response_delay,omitempty"`
	CRMLinks      []models.CRMLink `json:"crm_links,omitempty"`
}

// QuickCreateCompanyRequest represents a simplified request for quick company setup
type QuickCreateCompanyRequest struct {
	CompanyName     string           `json:"company_name" validate:"required"`
	PageID          string           `json:"page_id" validate:"required"`
	PageAccessToken string           `json:"page_access_token" validate:"required"`
	AppSecret       string           `json:"app_secret" validate:"required"`
	AIAPIKey        string           `json:"ai_api_key" validate:"required"`
	SystemPrompt    string           `json:"system_prompt,omitempty"`
	CRMLinks        []models.CRMLink `json:"crm_links,omitempty"`
}

// PageRequest represents a Facebook page in the request
type PageRequest struct {
	PageID          string `json:"page_id" validate:"required"`
	PageName        string `json:"page_name" validate:"required"`
	PageAccessToken string `json:"page_access_token" validate:"required"`
	IsActive        bool   `json:"is_active"`
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	UserID        string   `json:"user_id" validate:"required"`
	Username      string   `json:"username" validate:"required"`
	Email         string   `json:"email" validate:"required,email"`
	FullName      string   `json:"full_name" validate:"required"`
	CompanyID     string   `json:"company_id" validate:"required"`
	CompanyName   string   `json:"company_name" validate:"required"`
	Role          string   `json:"role" validate:"required"`
	Password      string   `json:"password,omitempty"`
	AssignedPages []string `json:"assigned_pages,omitempty"`
}

// CreateCompany handles the creation of a new company
func CreateCompany(c *fiber.Ctx) error {
	var req CreateCompanyRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Convert request to model
	company := &models.Company{
		ID:            primitive.NewObjectID(),
		CompanyID:     req.CompanyID,
		CompanyName:   req.CompanyName,
		AppSecret:     req.AppSecret,
		ClaudeAPIKey:  req.ClaudeAPIKey,
		ClaudeModel:   req.ClaudeModel,
		SystemPrompt:  req.SystemPrompt,
		IsActive:      true,
		MaxTokens:     req.MaxTokens,
		ResponseDelay: req.ResponseDelay,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Set defaults
	if company.ClaudeModel == "" {
		company.ClaudeModel = "claude-3-haiku-20240307"
	}
	if company.MaxTokens == 0 {
		company.MaxTokens = 1024
	}

	// Convert pages
	company.Pages = make([]models.FacebookPage, len(req.Pages))
	for i, page := range req.Pages {
		company.Pages[i] = models.FacebookPage{
			PageID:          page.PageID,
			PageName:        page.PageName,
			PageAccessToken: page.PageAccessToken,
			IsActive:        page.IsActive,
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Save to database
	if err := services.CreateCompany(ctx, company); err != nil {
		slog.Error("Failed to create company", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to create company",
			"details": err.Error(),
		})
	}

	slog.Info("Company created successfully",
		"companyID", company.CompanyID,
		"companyName", company.CompanyName,
		"pageCount", len(company.Pages))

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Company created successfully",
		"company": company,
	})
}

// CreateUser handles the creation of a new user
func CreateUser(c *fiber.Ctx) error {
	var req CreateUserRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Validate role
	if !models.IsValidRole(req.Role) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid role",
			"details": "Role must be one of: company_admin, bot_manager, human_agent, analyst, viewer",
			"valid_roles": []string{
				string(models.RoleCompanyAdmin),
				string(models.RoleBotManager),
				string(models.RoleHumanAgent),
				string(models.RoleAnalyst),
				string(models.RoleViewer),
			},
		})
	}

	// Convert request to model
	user := &models.User{
		UserID:        req.UserID,
		Username:      req.Username,
		Email:         req.Email,
		FullName:      req.FullName,
		CompanyID:     req.CompanyID,
		CompanyName:   req.CompanyName,
		Role:          models.UserRole(req.Role),
		AssignedPages: req.AssignedPages,
	}

	// Hash password if provided
	if req.Password != "" {
		hashedPassword, err := services.HashPassword(req.Password)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to hash password",
				"details": err.Error(),
			})
		}
		user.PasswordHash = hashedPassword
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Save to database
	if err := services.CreateUser(ctx, user); err != nil {
		slog.Error("Failed to create user", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to create user",
			"details": err.Error(),
		})
	}

	slog.Info("User created successfully",
		"userID", user.UserID,
		"username", user.Username,
		"companyID", user.CompanyID,
		"role", user.Role)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User created successfully",
		"user": fiber.Map{
			"user_id":        user.UserID,
			"username":       user.Username,
			"email":          user.Email,
			"full_name":      user.FullName,
			"company_id":     user.CompanyID,
			"company_name":   user.CompanyName,
			"role":           user.Role,
			"api_key":        user.APIKey,
			"assigned_pages": user.AssignedPages,
		},
	})
}

// GetUser retrieves a user by ID
func GetUser(c *fiber.Ctx) error {
	userID := c.Params("userID")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := services.GetUserByID(ctx, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "User not found",
			"details": err.Error(),
		})
	}

	return c.JSON(user)
}

// GetCompanyUsers retrieves all users for a company
func GetCompanyUsers(c *fiber.Ctx) error {
	companyID := c.Params("companyID")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	users, err := services.GetUsersByCompany(ctx, companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to retrieve users",
			"details": err.Error(),
		})
	}

	// Group users by role for better visualization
	usersByRole := make(map[string][]fiber.Map)
	for _, user := range users {
		userInfo := fiber.Map{
			"user_id":        user.UserID,
			"username":       user.Username,
			"email":          user.Email,
			"full_name":      user.FullName,
			"is_active":      user.IsActive,
			"last_login":     user.LastLogin,
			"assigned_pages": user.AssignedPages,
		}
		usersByRole[string(user.Role)] = append(usersByRole[string(user.Role)], userInfo)
	}

	return c.JSON(fiber.Map{
		"company_id":    companyID,
		"total_users":   len(users),
		"users_by_role": usersByRole,
	})
}

// UpdateUserRole updates a user's role
func UpdateUserRole(c *fiber.Ctx) error {
	userID := c.Params("userID")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	var req struct {
		Role          string   `json:"role" validate:"required"`
		AssignedPages []string `json:"assigned_pages,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Validate role
	if !models.IsValidRole(req.Role) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role",
			"valid_roles": []string{
				string(models.RoleCompanyAdmin),
				string(models.RoleBotManager),
				string(models.RoleHumanAgent),
				string(models.RoleAnalyst),
				string(models.RoleViewer),
			},
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{"role": req.Role}
	if req.Role == string(models.RoleBotManager) && len(req.AssignedPages) > 0 {
		update["assigned_pages"] = req.AssignedPages
	}

	if err := services.UpdateUser(ctx, userID, update); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to update user role",
			"details": err.Error(),
		})
	}

	slog.Info("User role updated", "userID", userID, "newRole", req.Role)

	return c.JSON(fiber.Map{
		"message":  "User role updated successfully",
		"user_id":  userID,
		"new_role": req.Role,
	})
}

// RegenerateUserAPIKey regenerates a user's API key
func RegenerateUserAPIKey(c *fiber.Ctx) error {
	userID := c.Params("userID")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	newAPIKey, err := services.RegenerateAPIKey(ctx, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to regenerate API key",
			"details": err.Error(),
		})
	}

	slog.Info("API key regenerated", "userID", userID)

	return c.JSON(fiber.Map{
		"message":     "API key regenerated successfully",
		"user_id":     userID,
		"new_api_key": newAPIKey,
	})
}

// AddPageToCompany handles adding a new page to an existing company
func AddPageToCompany(c *fiber.Ctx) error {
	companyID := c.Params("companyID")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID is required",
		})
	}

	var page PageRequest
	if err := c.BodyParser(&page); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get existing company
	company, err := services.GetCompanyByID(ctx, companyID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Company not found",
			"details": err.Error(),
		})
	}

	// Check if page already exists
	for _, existingPage := range company.Pages {
		if existingPage.PageID == page.PageID {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Page already exists for this company",
			})
		}
	}

	// Add new page
	newPage := models.FacebookPage{
		PageID:          page.PageID,
		PageName:        page.PageName,
		PageAccessToken: page.PageAccessToken,
		IsActive:        page.IsActive,
	}

	company.Pages = append(company.Pages, newPage)

	// Update company in database
	update := bson.M{
		"pages":      company.Pages,
		"updated_at": time.Now(),
	}

	if err := services.UpdateCompany(ctx, companyID, update); err != nil {
		slog.Error("Failed to add page to company", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to add page to company",
			"details": err.Error(),
		})
	}

	slog.Info("Page added to company successfully",
		"companyID", companyID,
		"pageID", page.PageID,
		"pageName", page.PageName)

	return c.JSON(fiber.Map{
		"message": "Page added successfully",
		"page":    newPage,
	})
}

// GetCompany retrieves a company by ID
func GetCompany(c *fiber.Ctx) error {
	companyID := c.Params("companyID")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	company, err := services.GetCompanyByID(ctx, companyID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Company not found",
			"details": err.Error(),
		})
	}

	// Hide sensitive information
	company.ClaudeAPIKey = "***HIDDEN***"
	company.AppSecret = "***HIDDEN***"
	for i := range company.Pages {
		if len(company.Pages[i].PageAccessToken) > 10 {
			company.Pages[i].PageAccessToken = company.Pages[i].PageAccessToken[:10] + "...***HIDDEN***"
		}
	}

	return c.JSON(company)
}

// TestWebhookConnection tests the webhook connection for a specific page
func TestWebhookConnection(c *fiber.Ctx) error {
	var req struct {
		PageID string `json:"page_id" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company by page ID
	company, err := services.GetCompanyByPageID(ctx, req.PageID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Company not found for this page",
			"details": err.Error(),
		})
	}

	// Get page config
	pageConfig, err := services.GetPageConfig(company, req.PageID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Page configuration not found",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":           "Page configuration found successfully",
		"company_id":        company.CompanyID,
		"company_name":      company.CompanyName,
		"page_id":           pageConfig.PageID,
		"page_name":         pageConfig.PageName,
		"is_active":         pageConfig.IsActive,
		"claude_model":      company.ClaudeModel,
		"has_system_prompt": company.SystemPrompt != "",
	})
}

// GetRolePermissions returns all available roles and their permissions
func GetRolePermissions(c *fiber.Ctx) error {
	permissions := models.GetRolePermissions()

	// Convert to a more readable format
	roles := make([]fiber.Map, 0)
	for _, perm := range permissions {
		roles = append(roles, fiber.Map{
			"role":        perm.Role,
			"description": perm.Description,
			"permissions": perm.Permissions,
		})
	}

	return c.JSON(fiber.Map{
		"roles": roles,
	})
}

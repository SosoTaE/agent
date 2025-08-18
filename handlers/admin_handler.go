package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"facebook-bot/models"
	"facebook-bot/services"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

// PageRequest represents a Facebook page in the request
type PageRequest struct {
	PageID          string `json:"page_id" validate:"required"`
	PageName        string `json:"page_name" validate:"required"`
	PageAccessToken string `json:"page_access_token" validate:"required"`
	IsActive        bool   `json:"is_active"`
}

// PageFullRequest represents a Facebook page with all configuration details
type PageFullRequest struct {
	PageID          string `json:"page_id" validate:"required"`
	PageName        string `json:"page_name" validate:"required"`
	PageAccessToken string `json:"page_access_token" validate:"required"`
	AppSecret       string `json:"app_secret" validate:"required"`
	ClaudeAPIKey    string `json:"claude_api_key" validate:"required"`
	ClaudeModel     string `json:"claude_model"`
	VoyageAPIKey    string `json:"voyage_api_key,omitempty"`
	VoyageModel     string `json:"voyage_model,omitempty"`
	SystemPrompt    string `json:"system_prompt,omitempty"`
	IsActive        bool   `json:"is_active"`
	MaxTokens       int    `json:"max_tokens"`
}

// PageUpdateRequest represents updates to an existing page configuration
type PageUpdateRequest struct {
	PageName        string `json:"page_name,omitempty"`
	PageAccessToken string `json:"page_access_token,omitempty"`
	AppSecret       string `json:"app_secret,omitempty"`
	ClaudeAPIKey    string `json:"claude_api_key,omitempty"`
	ClaudeModel     string `json:"claude_model,omitempty"`
	VoyageAPIKey    string `json:"voyage_api_key,omitempty"`
	VoyageModel     string `json:"voyage_model,omitempty"`
	SystemPrompt    string `json:"system_prompt,omitempty"`
	IsActive        *bool  `json:"is_active,omitempty"`
	MaxTokens       *int   `json:"max_tokens,omitempty"`
}

// AdminCreateUser handles the creation of a new user with pre-hashed password for admin
func AdminCreateUser(c *fiber.Ctx) error {
	// Only super admin or company admin can create users directly
	userRole := c.Locals("role")
	if userRole != string(models.RoleCompanyAdmin) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only company admins can create users",
		})
	}

	var req struct {
		Username  string    `json:"username" validate:"required"`
		Email     string    `json:"email" validate:"required,email"`
		Password  string    `json:"password" validate:"required"` // Pre-hashed password
		FirstName string    `json:"first_name" validate:"required"`
		LastName  string    `json:"last_name" validate:"required"`
		CompanyID string    `json:"company_id" validate:"required"`
		Role      string    `json:"role" validate:"required"`
		IsActive  bool      `json:"is_active"`
		CreatedAt time.Time `json:"created_at,omitempty"`
		UpdatedAt time.Time `json:"updated_at,omitempty"`
		LastLogin time.Time `json:"last_login,omitempty"`
	}

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

	// Set timestamps if not provided
	now := time.Now()
	if req.CreatedAt.IsZero() {
		req.CreatedAt = now
	}
	if req.UpdatedAt.IsZero() {
		req.UpdatedAt = now
	}

	// Create user with provided data
	user := &models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password, // Already hashed password
		FirstName: req.FirstName,
		LastName:  req.LastName,
		CompanyID: req.CompanyID,
		Role:      models.UserRole(req.Role),
		IsActive:  req.IsActive,
		CreatedAt: req.CreatedAt,
		UpdatedAt: req.UpdatedAt,
		LastLogin: req.LastLogin,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Save to database without hashing password (it's already hashed)
	if err := services.CreateUserWithHashedPassword(ctx, user); err != nil {
		slog.Error("Failed to create user", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to create user",
			"details": err.Error(),
		})
	}

	slog.Info("User created successfully by admin",
		"userID", user.ID.Hex(),
		"username", user.Username,
		"companyID", user.CompanyID,
		"role", user.Role)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User created successfully",
		"user": fiber.Map{
			"id":         user.ID.Hex(),
			"username":   user.Username,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"company_id": user.CompanyID,
			"role":       user.Role,
			"is_active":  user.IsActive,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
			"last_login": user.LastLogin,
		},
	})
}

// CreateUser handles the creation of a new user
func CreateUser(c *fiber.Ctx) error {
	// Get company_id and role from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Only company admin can create users
	userRole := c.Locals("role")
	if userRole != string(models.RoleCompanyAdmin) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only company admins can create users",
		})
	}

	var req struct {
		Username  string `json:"username" validate:"required"`
		Email     string `json:"email" validate:"required,email"`
		Password  string `json:"password" validate:"required"`
		FirstName string `json:"first_name" validate:"required"`
		LastName  string `json:"last_name" validate:"required"`
		Role      string `json:"role" validate:"required"`
	}

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

	// Convert request to model - use company_id from session
	user := &models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password, // Will be hashed in CreateUser
		FirstName: req.FirstName,
		LastName:  req.LastName,
		CompanyID: companyID.(string), // Use company_id from session
		Role:      models.UserRole(req.Role),
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
		"userID", user.ID.Hex(),
		"username", user.Username,
		"companyID", user.CompanyID,
		"role", user.Role)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User created successfully",
		"user": fiber.Map{
			"id":         user.ID.Hex(),
			"username":   user.Username,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"company_id": user.CompanyID,
			"role":       user.Role,
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

	// Get company_id from session
	companyID := c.Locals("company_id")

	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := services.GetUserByIDAndCompanyID(ctx, userID, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "User not found",
			"details": err.Error(),
		})
	}

	return c.JSON(user)
}

// GetCompanyUsers retrieves all users for the authenticated user's company
func GetCompanyUsers(c *fiber.Ctx) error {
	// Get company_id from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	users, err := services.GetUsersByCompany(ctx, companyID.(string))
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
			"id":         user.ID.Hex(),
			"username":   user.Username,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"is_active":  user.IsActive,
			"last_login": user.LastLogin,
		}
		usersByRole[string(user.Role)] = append(usersByRole[string(user.Role)], userInfo)
	}

	return c.JSON(fiber.Map{
		"company_id":    companyID.(string),
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

	// Get company_id and role from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Only company admin can update roles
	userRole := c.Locals("role")
	if userRole != string(models.RoleCompanyAdmin) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only company admins can update user roles",
		})
	}

	var req struct {
		Role string `json:"role" validate:"required"`
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

	// Verify the user belongs to the same company before updating
	_, err := services.GetUserByIDAndCompanyID(ctx, userID, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "User not found in your company",
		})
	}

	update := bson.M{"role": req.Role}

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

// RegenerateUserAPIKey is deprecated - API keys are no longer used
func RegenerateUserAPIKey(c *fiber.Ctx) error {
	return c.Status(fiber.StatusGone).JSON(fiber.Map{
		"error": "API key functionality has been removed. Please use session-based authentication.",
	})
}

// CreateCompanyRequest represents the request body for creating a new company
type CreateCompanyRequest struct {
	CompanyName     string `json:"company_name" validate:"required"`
	PageID          string `json:"page_id" validate:"required"`
	PageName        string `json:"page_name" validate:"required"`
	PageAccessToken string `json:"page_access_token" validate:"required"`
	AppSecret       string `json:"app_secret" validate:"required"`
	ClaudeAPIKey    string `json:"claude_api_key" validate:"required"`
	ClaudeModel     string `json:"claude_model,omitempty"`
	VoyageAPIKey    string `json:"voyage_api_key,omitempty"`
	VoyageModel     string `json:"voyage_model,omitempty"`
	SystemPrompt    string `json:"system_prompt,omitempty"`
	MaxTokens       int    `json:"max_tokens,omitempty"`
	ResponseDelay   int    `json:"response_delay,omitempty"`
	DefaultLanguage string `json:"default_language,omitempty"`
}

// CreateCompany handles creating a new company with its first page
func CreateCompany(c *fiber.Ctx) error {
	var req CreateCompanyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create first page
	firstPage := models.FacebookPage{
		PageID:          req.PageID,
		PageName:        req.PageName,
		PageAccessToken: req.PageAccessToken,
		AppSecret:       req.AppSecret,
		ClaudeAPIKey:    req.ClaudeAPIKey,
		ClaudeModel:     req.ClaudeModel,
		VoyageAPIKey:    req.VoyageAPIKey,
		VoyageModel:     req.VoyageModel,
		SystemPrompt:    req.SystemPrompt,
		IsActive:        true,
		MaxTokens:       req.MaxTokens,
	}

	// Create new company document with auto-generated ID
	newCompany := &models.Company{
		CompanyName:     req.CompanyName,
		Pages:           []models.FacebookPage{firstPage},
		IsActive:        true,
		ResponseDelay:   req.ResponseDelay,
		DefaultLanguage: req.DefaultLanguage,
	}

	// The company ID will be auto-generated in CreateCompany service
	if err := services.CreateCompany(ctx, newCompany); err != nil {
		slog.Error("Failed to create company", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to create company",
			"details": err.Error(),
		})
	}

	slog.Info("Company created successfully",
		"companyID", newCompany.CompanyID,
		"companyName", req.CompanyName,
		"pageID", req.PageID)

	// Return the generated company ID
	return c.JSON(fiber.Map{
		"message":    "Company created successfully",
		"company_id": newCompany.CompanyID,
		"company": fiber.Map{
			"company_id":   newCompany.CompanyID,
			"company_name": req.CompanyName,
			"page_id":      req.PageID,
			"page_name":    req.PageName,
		},
	})
}

// AddPageToCompany handles adding a new page to the authenticated user's company
func AddPageToCompany(c *fiber.Ctx) error {
	// Get company_id and role from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Only company admin can add pages
	userRole := c.Locals("role")
	if userRole != string(models.RoleCompanyAdmin) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only company admins can add pages",
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
	existingCompany, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Company not found",
			"details": err.Error(),
		})
	}

	// Check if page already exists
	for _, existingPage := range existingCompany.Pages {
		if existingPage.PageID == page.PageID {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Page already exists for this company",
			})
		}
	}

	// Get default values from first page if available
	var defaultPage *models.FacebookPage
	if len(existingCompany.Pages) > 0 {
		defaultPage = &existingCompany.Pages[0]
	}

	// Create new page with defaults from existing page
	newPage := models.FacebookPage{
		PageID:          page.PageID,
		PageName:        page.PageName,
		PageAccessToken: page.PageAccessToken,
		IsActive:        page.IsActive,
	}

	// Copy settings from default page if available
	if defaultPage != nil {
		newPage.AppSecret = defaultPage.AppSecret
		newPage.ClaudeAPIKey = defaultPage.ClaudeAPIKey
		newPage.ClaudeModel = defaultPage.ClaudeModel
		newPage.VoyageAPIKey = defaultPage.VoyageAPIKey
		newPage.VoyageModel = defaultPage.VoyageModel
		newPage.SystemPrompt = defaultPage.SystemPrompt
		newPage.MaxTokens = defaultPage.MaxTokens
	}

	// Add page to company's pages array
	updateData := bson.M{
		"$push": bson.M{
			"pages": newPage,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	if err := services.UpdateCompany(ctx, companyID.(string), updateData); err != nil {
		slog.Error("Failed to add page to company", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to add page to company",
			"details": err.Error(),
		})
	}

	slog.Info("Page added to company successfully",
		"companyID", companyID.(string),
		"pageID", page.PageID,
		"pageName", page.PageName)

	return c.JSON(fiber.Map{
		"message": "Page added successfully",
		"page": fiber.Map{
			"page_id":           page.PageID,
			"page_name":         page.PageName,
			"page_access_token": page.PageAccessToken[:10] + "...***HIDDEN***",
			"is_active":         page.IsActive,
		},
	})
}

// AddPageWithFullDetails handles adding a new page with complete configuration to the company
func AddPageWithFullDetails(c *fiber.Ctx) error {
	// Get company_id and role from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Only company admin can add pages
	userRole := c.Locals("role")
	if userRole != string(models.RoleCompanyAdmin) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only company admins can add pages",
		})
	}

	var req PageFullRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get existing company
	existingCompany, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Company not found",
			"details": err.Error(),
		})
	}

	// Check if page already exists
	for _, existingPage := range existingCompany.Pages {
		if existingPage.PageID == req.PageID {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Page already exists for this company",
			})
		}
	}

	// Create new page with full configuration
	newPage := models.FacebookPage{
		PageID:          req.PageID,
		PageName:        req.PageName,
		PageAccessToken: req.PageAccessToken,
		AppSecret:       req.AppSecret,
		ClaudeAPIKey:    req.ClaudeAPIKey,
		ClaudeModel:     req.ClaudeModel,
		VoyageAPIKey:    req.VoyageAPIKey,
		VoyageModel:     req.VoyageModel,
		SystemPrompt:    req.SystemPrompt,
		IsActive:        req.IsActive,
		MaxTokens:       req.MaxTokens,
	}

	// Set defaults if not provided
	if newPage.ClaudeModel == "" {
		newPage.ClaudeModel = "claude-3-haiku-20240307"
	}
	if newPage.MaxTokens == 0 {
		newPage.MaxTokens = 1024
	}

	// Add page to company's pages array
	updateData := bson.M{
		"$push": bson.M{
			"pages": newPage,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	if err := services.UpdateCompany(ctx, companyID.(string), updateData); err != nil {
		slog.Error("Failed to add page with full details to company", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to add page to company",
			"details": err.Error(),
		})
	}

	slog.Info("Page with full details added to company successfully",
		"companyID", companyID.(string),
		"pageID", req.PageID,
		"pageName", req.PageName)

	// Prepare response with hidden sensitive data
	pageAccessTokenDisplay := req.PageAccessToken
	if len(pageAccessTokenDisplay) > 10 {
		pageAccessTokenDisplay = pageAccessTokenDisplay[:10] + "...***HIDDEN***"
	}

	return c.JSON(fiber.Map{
		"message": "Page added successfully with full configuration",
		"page": fiber.Map{
			"page_id":           req.PageID,
			"page_name":         req.PageName,
			"page_access_token": pageAccessTokenDisplay,
			"app_secret":        "***HIDDEN***",
			"claude_api_key":    "***HIDDEN***",
			"claude_model":      newPage.ClaudeModel,
			"voyage_api_key":    "***HIDDEN***",
			"voyage_model":      req.VoyageModel,
			"system_prompt":     req.SystemPrompt,
			"is_active":         req.IsActive,
			"max_tokens":        newPage.MaxTokens,
		},
	})
}

// UpdatePageConfiguration updates an existing page's configuration
func UpdatePageConfiguration(c *fiber.Ctx) error {
	// Get company_id and role from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Only company admin can update pages
	userRole := c.Locals("role")
	if userRole != string(models.RoleCompanyAdmin) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only company admins can update pages",
		})
	}

	pageID := c.Params("pageID")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID is required",
		})
	}

	var req PageUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get existing company
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Company not found",
			"details": err.Error(),
		})
	}

	// Find and update the page
	pageFound := false
	updatedPages := make([]models.FacebookPage, len(company.Pages))

	for i, page := range company.Pages {
		if page.PageID == pageID {
			pageFound = true
			// Update only provided fields
			if req.PageName != "" {
				page.PageName = req.PageName
			}
			if req.PageAccessToken != "" {
				page.PageAccessToken = req.PageAccessToken
			}
			if req.AppSecret != "" {
				page.AppSecret = req.AppSecret
			}
			if req.ClaudeAPIKey != "" {
				page.ClaudeAPIKey = req.ClaudeAPIKey
			}
			if req.ClaudeModel != "" {
				page.ClaudeModel = req.ClaudeModel
			}
			if req.VoyageAPIKey != "" {
				page.VoyageAPIKey = req.VoyageAPIKey
			}
			if req.VoyageModel != "" {
				page.VoyageModel = req.VoyageModel
			}
			if req.SystemPrompt != "" {
				page.SystemPrompt = req.SystemPrompt
			}
			if req.IsActive != nil {
				page.IsActive = *req.IsActive
			}
			if req.MaxTokens != nil {
				page.MaxTokens = *req.MaxTokens
			}
		}
		updatedPages[i] = page
	}

	if !pageFound {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Page not found in company",
		})
	}

	// Update the company document with modified pages array
	updateData := bson.M{
		"$set": bson.M{
			"pages":      updatedPages,
			"updated_at": time.Now(),
		},
	}

	if err := services.UpdateCompany(ctx, companyID.(string), updateData); err != nil {
		slog.Error("Failed to update page configuration", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to update page configuration",
			"details": err.Error(),
		})
	}

	slog.Info("Page configuration updated successfully",
		"companyID", companyID.(string),
		"pageID", pageID)

	return c.JSON(fiber.Map{
		"message": "Page configuration updated successfully",
		"page_id": pageID,
	})
}

// GetCompany retrieves the authenticated user's company info with all pages
func GetCompany(c *fiber.Ctx) error {
	// Get company_id from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company document
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Company not found",
			"details": err.Error(),
		})
	}

	// Prepare pages with hidden tokens
	pages := make([]fiber.Map, 0, len(company.Pages))
	for _, page := range company.Pages {
		pageToken := page.PageAccessToken
		if len(pageToken) > 10 {
			pageToken = pageToken[:10] + "...***HIDDEN***"
		}
		pages = append(pages, fiber.Map{
			"page_id":           page.PageID,
			"page_name":         page.PageName,
			"page_access_token": pageToken,
			"app_secret":        "***HIDDEN***",
			"claude_api_key":    "***HIDDEN***",
			"claude_model":      page.ClaudeModel,
			"voyage_api_key":    "***HIDDEN***",
			"voyage_model":      page.VoyageModel,
			"system_prompt":     page.SystemPrompt,
			"is_active":         page.IsActive,
			"max_tokens":        page.MaxTokens,
			"crm_links":         page.CRMLinks,
		})
	}

	// Build response with company info and pages array
	response := fiber.Map{
		"id":               company.ID,
		"company_id":       company.CompanyID,
		"company_name":     company.CompanyName,
		"pages":            pages,
		"is_active":        company.IsActive,
		"response_delay":   company.ResponseDelay,
		"default_language": company.DefaultLanguage,
		"created_at":       company.CreatedAt,
		"updated_at":       company.UpdatedAt,
	}

	return c.JSON(response)
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
		"claude_model":      pageConfig.ClaudeModel,
		"has_system_prompt": pageConfig.SystemPrompt != "",
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

// ToggleCRMLink handles toggling the active status of a CRM link
func ToggleCRMLink(c *fiber.Ctx) error {
	// Get company ID from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get page ID from URL params
	pageID := c.Params("pageID")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID is required",
		})
	}

	// Parse request body
	var reqBody struct {
		URL      string `json:"url" validate:"required"`
		IsActive bool   `json:"is_active"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if reqBody.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "CRM URL is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify page belongs to company
	_, err := services.ValidatePageOwnership(ctx, pageID, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Toggle the CRM link status
	err = services.ToggleCRMLink(ctx, companyID.(string), pageID, reqBody.URL, reqBody.IsActive)
	if err != nil {
		slog.Error("Failed to toggle CRM link status",
			"companyID", companyID,
			"pageID", pageID,
			"url", reqBody.URL,
			"error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	slog.Info("CRM link status toggled",
		"companyID", companyID,
		"pageID", pageID,
		"url", reqBody.URL,
		"isActive", reqBody.IsActive)

	return c.JSON(fiber.Map{
		"message":   "CRM link status updated successfully",
		"url":       reqBody.URL,
		"is_active": reqBody.IsActive,
	})
}

// GetCRMLinks retrieves all CRM links for a specific page
func GetCRMLinks(c *fiber.Ctx) error {
	// Get company ID from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get page ID from URL params
	pageID := c.Params("pageID")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify page belongs to company
	_, err := services.ValidatePageOwnership(ctx, pageID, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Get CRM links
	crmLinks, err := services.GetCRMLinks(ctx, companyID.(string), pageID)
	if err != nil {
		slog.Error("Failed to get CRM links",
			"companyID", companyID,
			"pageID", pageID,
			"error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve CRM links",
		})
	}

	return c.JSON(fiber.Map{
		"crm_links": crmLinks,
		"page_id":   pageID,
		"count":     len(crmLinks),
	})
}

// UpdateCRMLink updates a CRM link configuration
func UpdateCRMLink(c *fiber.Ctx) error {
	// Get company ID from session
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get page ID from URL params
	pageID := c.Params("pageID")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID is required",
		})
	}

	// Parse request body
	var reqBody models.CRMLink
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if reqBody.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "CRM URL is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify page belongs to company
	_, err := services.ValidatePageOwnership(ctx, pageID, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Get the database
	db := services.GetDatabase()
	collection := db.Collection("companies")

	// Update the CRM link
	filter := bson.M{
		"company_id":    companyID,
		"pages.page_id": pageID,
	}

	// Find the company first to get indices
	var company models.Company
	err = collection.FindOne(ctx, filter).Decode(&company)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company or page not found",
		})
	}

	// Find indices
	pageIndex := -1
	crmIndex := -1
	for i, page := range company.Pages {
		if page.PageID == pageID {
			pageIndex = i
			for j, crm := range page.CRMLinks {
				if crm.URL == reqBody.URL {
					crmIndex = j
					break
				}
			}
			break
		}
	}

	if pageIndex == -1 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Page not found",
		})
	}

	var update bson.M
	if crmIndex == -1 {
		// Add new CRM link
		update = bson.M{
			"$push": bson.M{
				fmt.Sprintf("pages.%d.crm_links", pageIndex): reqBody,
			},
			"$set": bson.M{
				"updated_at": time.Now(),
			},
		}
	} else {
		// Update existing CRM link
		update = bson.M{
			"$set": bson.M{
				fmt.Sprintf("pages.%d.crm_links.%d", pageIndex, crmIndex): reqBody,
				"updated_at": time.Now(),
			},
		}
	}

	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {
		slog.Error("Failed to update CRM link",
			"companyID", companyID,
			"pageID", pageID,
			"error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update CRM link",
		})
	}

	// Clear cache
	services.GetWebSocketManager() // This will trigger any necessary cache clearing

	action := "updated"
	if crmIndex == -1 {
		action = "added"
	}

	return c.JSON(fiber.Map{
		"message":  fmt.Sprintf("CRM link %s successfully", action),
		"crm_link": reqBody,
	})
}

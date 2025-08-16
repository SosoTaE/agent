package handlers

import (
	"context"
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

	// Create new company document with auto-generated ID
	newCompany := &models.Company{
		CompanyName:     req.CompanyName,
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
// With the new structure, this creates a new company document for the page
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

	// Get one existing company document to copy settings from
	existingCompany, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Company not found",
			"details": err.Error(),
		})
	}

	// Check if page already exists for this company
	existingPages, err := services.GetPagesByCompanyID(ctx, companyID.(string))
	if err == nil {
		for _, existingPage := range existingPages {
			if existingPage.PageID == page.PageID {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": "Page already exists for this company",
				})
			}
		}
	}

	// Create new company document for this page
	newCompanyDoc := &models.Company{
		CompanyID:       existingCompany.CompanyID,
		CompanyName:     existingCompany.CompanyName,
		PageID:          page.PageID,
		PageName:        page.PageName,
		PageAccessToken: page.PageAccessToken,
		AppSecret:       existingCompany.AppSecret,
		ClaudeAPIKey:    existingCompany.ClaudeAPIKey,
		ClaudeModel:     existingCompany.ClaudeModel,
		VoyageAPIKey:    existingCompany.VoyageAPIKey,
		VoyageModel:     existingCompany.VoyageModel,
		SystemPrompt:    existingCompany.SystemPrompt,
		CRMLinks:        existingCompany.CRMLinks,
		IsActive:        page.IsActive,
		MaxTokens:       existingCompany.MaxTokens,
		ResponseDelay:   existingCompany.ResponseDelay,
		DefaultLanguage: existingCompany.DefaultLanguage,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Create the new company document
	if err := services.CreateCompany(ctx, newCompanyDoc); err != nil {
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

	// Get all company documents (one per page) for this company ID
	companies, err := services.GetCompaniesByCompanyID(ctx, companyID.(string))
	if err != nil || len(companies) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Company not found",
			"details": err.Error(),
		})
	}

	// Use the first company doc for general company info
	firstCompany := companies[0]

	// Collect all pages
	pages := make([]fiber.Map, 0, len(companies))
	for _, comp := range companies {
		pageToken := comp.PageAccessToken
		if len(pageToken) > 10 {
			pageToken = pageToken[:10] + "...***HIDDEN***"
		}
		pages = append(pages, fiber.Map{
			"page_id":           comp.PageID,
			"page_name":         comp.PageName,
			"page_access_token": pageToken,
			"is_active":         comp.IsActive,
		})
	}

	// Build response with company info and pages array
	response := fiber.Map{
		"id":               firstCompany.ID,
		"company_id":       firstCompany.CompanyID,
		"company_name":     firstCompany.CompanyName,
		"pages":            pages,
		"app_secret":       "***HIDDEN***",
		"claude_api_key":   "***HIDDEN***",
		"claude_model":     firstCompany.ClaudeModel,
		"voyage_api_key":   "***HIDDEN***",
		"voyage_model":     firstCompany.VoyageModel,
		"system_prompt":    firstCompany.SystemPrompt,
		"crm_links":        firstCompany.CRMLinks,
		"is_active":        firstCompany.IsActive,
		"max_tokens":       firstCompany.MaxTokens,
		"response_delay":   firstCompany.ResponseDelay,
		"default_language": firstCompany.DefaultLanguage,
		"created_at":       firstCompany.CreatedAt,
		"updated_at":       firstCompany.UpdatedAt,
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

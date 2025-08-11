package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"facebook-bot/models"
	"facebook-bot/services"
)

// SimpleUserRequest for testing - only requires email, password, company_name, and role
type SimpleUserRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required"`
	CompanyName string `json:"company_name" validate:"required"`
	Role        string `json:"role" validate:"required"`
}

// TestCreateUser creates a user with minimal required fields
func TestCreateUser(c *fiber.Ctx) error {
	var req SimpleUserRequest

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

	// Find company by name or generate company ID
	var companyID string
	companies, err := services.GetAllCompanies(ctx)
	if err == nil {
		// Look for company with matching name
		for _, comp := range companies {
			if strings.EqualFold(comp.CompanyName, req.CompanyName) {
				companyID = comp.CompanyID
				break
			}
		}
	}

	// If company not found, generate a new company ID
	if companyID == "" {
		companyID = fmt.Sprintf("company_%s_%d",
			strings.ReplaceAll(strings.ToLower(req.CompanyName), " ", "_"),
			time.Now().Unix())
		slog.Info("Company not found, generated new company ID",
			"companyName", req.CompanyName,
			"companyID", companyID)
	} else {
		slog.Info("Found existing company",
			"companyName", req.CompanyName,
			"companyID", companyID)
	}

	// Generate username from email (part before @)
	emailParts := strings.Split(req.Email, "@")
	username := emailParts[0]

	// Generate unique user ID
	userID := fmt.Sprintf("user_%s_%d", username, time.Now().Unix())

	// Hash password
	hashedPassword, err := services.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to hash password",
			"details": err.Error(),
		})
	}

	// Create user model
	user := &models.User{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		Username:     username,
		Email:        req.Email,
		FullName:     username, // Use username as full name for testing
		CompanyID:    companyID,
		CompanyName:  req.CompanyName,
		Role:         models.UserRole(req.Role),
		PasswordHash: hashedPassword,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

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
		"email", user.Email,
		"companyID", user.CompanyID,
		"role", user.Role)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User created successfully",
		"user": fiber.Map{
			"user_id":      user.UserID,
			"username":     user.Username,
			"email":        user.Email,
			"company_id":   user.CompanyID,
			"company_name": user.CompanyName,
			"role":         user.Role,
			"api_key":      user.APIKey,
		},
	})
}

// SimpleCompanyRequest for testing - minimal fields for quick setup
type SimpleCompanyRequest struct {
	CompanyName     string           `json:"company_name" validate:"required"`
	PageID          string           `json:"page_id" validate:"required"`
	PageAccessToken string           `json:"page_access_token" validate:"required"`
	AppSecret       string           `json:"app_secret" validate:"required"`
	AIAPIKey        string           `json:"ai_api_key" validate:"required"`
	SystemPrompt    string           `json:"system_prompt,omitempty"`
	CRMLinks        []models.CRMLink `json:"crm_links,omitempty"`
}

// TestCreateCompany creates a company with minimal required fields
func TestCreateCompany(c *fiber.Ctx) error {
	var req SimpleCompanyRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Generate company ID from name
	companyID := fmt.Sprintf("company_%s_%d",
		strings.ReplaceAll(strings.ToLower(req.CompanyName), " ", "_"),
		time.Now().Unix())

	// Create company model
	company := &models.Company{
		ID:            primitive.NewObjectID(),
		CompanyID:     companyID,
		CompanyName:   req.CompanyName,
		AppSecret:     req.AppSecret,
		ClaudeAPIKey:  req.AIAPIKey,
		ClaudeModel:   "claude-3-haiku-20240307", // Default model
		SystemPrompt:  req.SystemPrompt,
		CRMLinks:      req.CRMLinks,
		IsActive:      true,
		MaxTokens:     1024,
		ResponseDelay: 0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Add the Facebook page
	company.Pages = []models.FacebookPage{
		{
			PageID:          req.PageID,
			PageName:        req.CompanyName + " Page", // Default page name
			PageAccessToken: req.PageAccessToken,
			IsActive:        true,
		},
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
		"pageID", req.PageID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Company created successfully",
		"company": fiber.Map{
			"company_id":   company.CompanyID,
			"company_name": company.CompanyName,
			"page_id":      req.PageID,
			"is_active":    company.IsActive,
		},
	})
}

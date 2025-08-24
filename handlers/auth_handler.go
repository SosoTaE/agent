package handlers

import (
	"facebook-bot/models"
	"facebook-bot/services"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required"`
	CompanyID string `json:"company_id" validate:"required"`
}

type LoginResponse struct {
	Message string       `json:"message"`
	User    *models.User `json:"user"`
}

func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Email == "" || req.Password == "" || req.CompanyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email, password, and company_id are required",
		})
	}

	user, err := services.GetUserByEmailAndCompany(c.Context(), req.Email, req.CompanyID)
	if err != nil {
		slog.Error("Failed to get user", "error", err, "email", req.Email)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	if !user.IsActive {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Account is disabled",
		})
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		slog.Info("Invalid password attempt", "email", req.Email)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	// Create database session
	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")
	session, err := services.CreateSession(
		c.Context(),
		user.ID.Hex(),
		user.Username,
		user.Email,
		user.CompanyID,
		string(user.Role),
		ipAddress,
		userAgent,
	)
	if err != nil {
		slog.Error("Failed to create session", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Detect if we're using ngrok or cross-origin setup
	origin := c.Get("Origin", "")
	isNgrok := strings.Contains(origin, "ngrok") || strings.Contains(c.Hostname(), "ngrok")
	isCrossOrigin := origin != "" && !strings.HasPrefix(origin, "http://"+c.Hostname()) && !strings.HasPrefix(origin, "https://"+c.Hostname())

	// Set appropriate SameSite based on environment
	sameSite := "Lax"
	secure := false
	if isNgrok || isCrossOrigin {
		sameSite = "None"
		secure = true // Must be true when SameSite=None
	}

	// Set session cookie
	c.Cookie(&fiber.Cookie{
		Name:     services.SessionCookieName,
		Value:    session.SessionID,
		Expires:  session.ExpiresAt,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
	})

	err = services.UpdateUserLastLogin(c.Context(), user.ID.Hex())
	if err != nil {
		slog.Error("Failed to update last login", "error", err)
	}

	slog.Info("User logged in", "user_id", user.ID.Hex(), "email", user.Email)

	return c.Status(fiber.StatusOK).JSON(LoginResponse{
		Message: "Login successful",
		User:    user,
	})
}

func Logout(c *fiber.Ctx) error {
	// Get session ID from cookie
	sessionID := c.Cookies(services.SessionCookieName)
	if sessionID == "" {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Logged out successfully",
		})
	}

	// Get session from database to log user ID
	session, _ := services.GetSessionByID(c.Context(), sessionID)
	var userID string
	if session != nil {
		userID = session.UserID
	}

	// Destroy session in database
	if err := services.DestroySession(c.Context(), sessionID); err != nil {
		slog.Error("Failed to destroy session", "error", err)
	}

	// Detect if we're using ngrok or cross-origin setup (same as login)
	origin := c.Get("Origin", "")
	isNgrok := strings.Contains(origin, "ngrok") || strings.Contains(c.Hostname(), "ngrok")
	isCrossOrigin := origin != "" && !strings.HasPrefix(origin, "http://"+c.Hostname()) && !strings.HasPrefix(origin, "https://"+c.Hostname())

	// Set appropriate SameSite based on environment
	sameSite := "Lax"
	secure := false
	if isNgrok || isCrossOrigin {
		sameSite = "None"
		secure = true
	}

	// Clear session cookie
	c.Cookie(&fiber.Cookie{
		Name:     services.SessionCookieName,
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
	})

	slog.Info("User logged out", "user_id", userID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

func GetCurrentUser(c *fiber.Ctx) error {
	// Get session ID from cookie
	sessionID := c.Cookies(services.SessionCookieName)
	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	// Get session from database
	session, err := services.GetSessionByID(c.Context(), sessionID)
	if err != nil || session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	user, err := services.GetUserByID(c.Context(), session.UserID)
	if err != nil {
		slog.Error("Failed to get user", "error", err, "user_id", session.UserID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get user information",
		})
	}

	return c.Status(fiber.StatusOK).JSON(user)
}

func CheckSession(c *fiber.Ctx) error {
	// Get session ID from cookie
	sessionID := c.Cookies(services.SessionCookieName)
	if sessionID == "" {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"authenticated": false,
		})
	}

	// Get session from database
	session, err := services.GetSessionByID(c.Context(), sessionID)
	if err != nil || session == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"authenticated": false,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"authenticated": true,
		"user_id":       session.UserID,
		"username":      session.Username,
		"email":         session.Email,
		"company_id":    session.CompanyID,
		"role":          session.Role,
	})
}

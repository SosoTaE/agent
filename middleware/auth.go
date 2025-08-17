package middleware

import (
	"facebook-bot/models"
	"facebook-bot/services"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

func RequireAuth(c *fiber.Ctx) error {
	// Get session ID from cookie
	sessionID := c.Cookies(services.SessionCookieName)
	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Get session from database
	session, err := services.GetSessionByID(c.Context(), sessionID)
	if err != nil {
		slog.Error("Failed to get session", "error", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	if session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired session",
		})
	}

	// Set user information in locals for downstream handlers
	c.Locals("user_id", session.UserID)
	c.Locals("email", session.Email)
	c.Locals("company_id", session.CompanyID)
	c.Locals("role", session.Role)
	c.Locals("username", session.Username)

	// Extend session expiration on activity
	services.ExtendSession(c.Context(), sessionID)

	return c.Next()
}

func RequireRole(roles ...models.UserRole) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := RequireAuth(c); err != nil {
			return err
		}

		userRole := c.Locals("role")
		if userRole == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied",
			})
		}

		roleStr, ok := userRole.(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Invalid role",
			})
		}

		currentRole := models.UserRole(roleStr)

		for _, allowedRole := range roles {
			if currentRole == allowedRole {
				return c.Next()
			}
		}

		slog.Info("Access denied", "user_role", currentRole, "required_roles", roles)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}
}

func RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := RequireAuth(c); err != nil {
			return err
		}

		userRole := c.Locals("role")
		if userRole == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied",
			})
		}

		roleStr, ok := userRole.(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Invalid role",
			})
		}

		permissions := models.GetRolePermissions()
		rolePerms, exists := permissions[models.UserRole(roleStr)]
		if !exists {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Invalid role",
			})
		}

		for _, perm := range rolePerms.Permissions {
			if perm == permission {
				return c.Next()
			}
		}

		slog.Info("Permission denied", "user_role", roleStr, "required_permission", permission)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}
}

func RequireCompany(c *fiber.Ctx) error {
	if err := RequireAuth(c); err != nil {
		return err
	}

	companyID := c.Locals("company_id")
	if companyID == nil || companyID == "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "No company associated with this user",
		})
	}

	requestedCompanyID := c.Params("companyID")
	if requestedCompanyID != "" && requestedCompanyID != companyID {
		userRole := c.Locals("role")
		if userRole != string(models.RoleCompanyAdmin) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied to this company",
			})
		}
	}

	return c.Next()
}

func RequireCompanyAdmin(c *fiber.Ctx) error {
	if err := RequireAuth(c); err != nil {
		return err
	}

	userRole := c.Locals("role")
	if userRole == nil || userRole.(string) != string(models.RoleCompanyAdmin) {
		slog.Info("Access denied - company admin required", "user_role", userRole)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only company admins can perform this action",
		})
	}

	return c.Next()
}

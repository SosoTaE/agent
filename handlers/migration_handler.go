package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"

	"facebook-bot/services"
)

// MigrateChannelsHandler handles the migration of channels from array to map structure
func MigrateChannelsHandler(c *fiber.Ctx) error {
	// Check authentication - this should only be accessible to admins
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Optional: Add additional admin check here
	// For now, we'll allow any authenticated company to trigger migration

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	slog.Info("Starting channels migration",
		"triggered_by", companyID,
		"timestamp", time.Now().Format(time.RFC3339))

	// Run the migration
	err := services.MigrateChannelsToMap(ctx)
	if err != nil {
		slog.Error("Migration failed",
			"error", err,
			"triggered_by", companyID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Migration failed",
			"details": err.Error(),
		})
	}

	slog.Info("Migration completed successfully",
		"triggered_by", companyID,
		"timestamp", time.Now().Format(time.RFC3339))

	return c.JSON(fiber.Map{
		"message":   "Migration completed successfully",
		"details":   "Channels have been migrated from array to map structure",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

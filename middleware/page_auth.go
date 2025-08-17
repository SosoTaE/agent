package middleware

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"log/slog"

	"facebook-bot/services"
)

// ValidatePageAccess middleware ensures the user has access to the requested page
func ValidatePageAccess(c *fiber.Ctx) error {
	// Get page_id from params or query
	pageID := c.Params("pageID")
	if pageID == "" {
		pageID = c.Query("page_id")
	}

	// If no page specified, continue
	if pageID == "" {
		return c.Next()
	}

	// Get company_id from session
	companyID, ok := c.Locals("company_id").(string)
	if !ok || companyID == "" {
		// No company context, allow access (public route)
		return c.Next()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate page access
	err := services.ValidatePageAccess(ctx, pageID, companyID)
	if err != nil {
		slog.Warn("Page access denied",
			"pageID", pageID,
			"companyID", companyID,
			"error", err,
		)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied: page does not belong to your company",
		})
	}

	// Store validated page_id in locals for use in handlers
	c.Locals("validated_page_id", pageID)

	return c.Next()
}

// ValidatePostAccess middleware ensures the user has access to the requested post
func ValidatePostAccess(c *fiber.Ctx) error {
	// Get post_id from params
	postID := c.Params("postID")
	if postID == "" {
		postID = c.Query("post_id")
	}

	// If no post specified, continue
	if postID == "" {
		return c.Next()
	}

	// Get company_id from session
	companyID, ok := c.Locals("company_id").(string)
	if !ok || companyID == "" {
		// No company context, allow access (public route)
		return c.Next()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate post ownership
	pageID, err := services.ValidatePostOwnership(ctx, postID, companyID)
	if err != nil {
		slog.Warn("Post access denied",
			"postID", postID,
			"companyID", companyID,
			"error", err,
		)

		// Check if it's a not found error
		if err.Error() == "post not found: mongo: no documents in result" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Post not found",
			})
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied: post does not belong to your company",
		})
	}

	// Store validated page_id in locals for use in handlers
	c.Locals("validated_page_id", pageID)
	c.Locals("validated_post_id", postID)

	return c.Next()
}

// ExtractCompanyPages middleware extracts and stores company page IDs
func ExtractCompanyPages(c *fiber.Ctx) error {
	// Get company_id from session
	companyID, ok := c.Locals("company_id").(string)
	if !ok || companyID == "" {
		// No company context, continue
		return c.Next()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get all page IDs for the company
	pageIDs, err := services.GetCompanyPageIDs(ctx, companyID)
	if err != nil {
		slog.Error("Failed to get company pages",
			"companyID", companyID,
			"error", err,
		)
		// Continue anyway, individual handlers will handle the error
		return c.Next()
	}

	// Store page IDs in locals
	c.Locals("company_page_ids", pageIDs)

	return c.Next()
}

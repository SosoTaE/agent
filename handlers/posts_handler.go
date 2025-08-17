package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"log/slog"

	"facebook-bot/services"
)

// GetAllPostIDsHandler returns all unique post IDs
func GetAllPostIDsHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if we want simple IDs or detailed info
	detailed := c.Query("detailed", "false") == "true"

	if detailed {
		posts, err := services.GetAllPostIDsWithInfo(ctx)
		if err != nil {
			slog.Error("Failed to get post IDs with info", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve posts",
			})
		}

		return c.JSON(fiber.Map{
			"total": len(posts),
			"posts": posts,
		})
	} else {
		postIDs, err := services.GetAllPostIDs(ctx)
		if err != nil {
			slog.Error("Failed to get post IDs", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve post IDs",
			})
		}

		return c.JSON(fiber.Map{
			"total":    len(postIDs),
			"post_ids": postIDs,
		})
	}
}

// GetPostIDsPaginatedHandler returns paginated post IDs with info
func GetPostIDsPaginatedHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	posts, total, err := services.GetPostIDsPaginated(ctx, page, limit)
	if err != nil {
		slog.Error("Failed to get paginated post IDs",
			"error", err,
			"page", page,
			"limit", limit,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve posts",
		})
	}

	// Calculate pagination metadata
	totalPages := (total + int64(limit) - 1) / int64(limit)
	hasNext := page < int(totalPages)
	hasPrev := page > 1

	return c.JSON(fiber.Map{
		"posts": posts,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    hasNext,
			"has_prev":    hasPrev,
		},
	})
}

// GetPostIDsByPageHandler returns all post IDs for a specific page
func GetPostIDsByPageHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pageID := c.Params("pageID")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "page_id is required",
		})
	}

	posts, err := services.GetPostIDsByPage(ctx, pageID)
	if err != nil {
		slog.Error("Failed to get post IDs by page",
			"error", err,
			"pageID", pageID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve posts",
		})
	}

	return c.JSON(fiber.Map{
		"page_id": pageID,
		"total":   len(posts),
		"posts":   posts,
	})
}

// GetPostIDsByCompanyHandler returns all post IDs for a company's pages
func GetPostIDsByCompanyHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get company_id from session (for authenticated endpoints)
	companyID, ok := c.Locals("company_id").(string)
	if !ok || companyID == "" {
		// Try to get from query parameter (for public endpoints)
		companyID = c.Query("company_id")
		if companyID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "company_id is required",
			})
		}
	}

	// Get optional page_id filter
	pageID := c.Query("page_id")

	// If a specific page is requested, validate access
	if pageID != "" {
		err := services.ValidatePageAccess(ctx, pageID, companyID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied: " + err.Error(),
			})
		}

		// Get posts for the specific page
		posts, err := services.GetPostIDsByPage(ctx, pageID)
		if err != nil {
			slog.Error("Failed to get post IDs by page",
				"error", err,
				"pageID", pageID,
				"companyID", companyID,
			)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"company_id": companyID,
			"page_id":    pageID,
			"total":      len(posts),
			"posts":      posts,
		})
	}

	// Get posts for all company pages
	posts, err := services.GetPostIDsByCompany(ctx, companyID)
	if err != nil {
		slog.Error("Failed to get post IDs by company",
			"error", err,
			"companyID", companyID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"company_id": companyID,
		"total":      len(posts),
		"posts":      posts,
	})
}

// GetPostsStatsHandler returns statistics about posts
func GetPostsStatsHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get all posts with info
	posts, err := services.GetAllPostIDsWithInfo(ctx)
	if err != nil {
		slog.Error("Failed to get posts for stats", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve statistics",
		})
	}

	// Calculate statistics
	totalPosts := len(posts)
	totalComments := 0
	pageStats := make(map[string]int)

	var mostActivePost services.PostInfo
	maxComments := 0

	// Process statistics
	for _, post := range posts {
		totalComments += post.CommentCount
		pageStats[post.PageName]++

		if post.CommentCount > maxComments {
			maxComments = post.CommentCount
			mostActivePost = post
		}
	}

	// Calculate average comments per post
	avgComments := 0.0
	if totalPosts > 0 {
		avgComments = float64(totalComments) / float64(totalPosts)
	}

	return c.JSON(fiber.Map{
		"statistics": fiber.Map{
			"total_posts":               totalPosts,
			"total_comments":            totalComments,
			"average_comments_per_post": avgComments,
			"pages_count":               len(pageStats),
		},
		"most_active_post":  mostActivePost,
		"page_distribution": pageStats,
		"recent_posts":      posts[:min(10, len(posts))], // Top 10 most recent
	})
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

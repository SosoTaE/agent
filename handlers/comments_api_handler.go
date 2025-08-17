package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"log/slog"

	"facebook-bot/services"
)

// GetPostCommentsHandler handles requests to get comments for a post
func GetPostCommentsHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	postID := c.Params("postID")
	if postID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "post_id is required",
		})
	}

	// Get hierarchical view query parameter
	hierarchical := c.Query("hierarchical", "true") == "true"

	// Check for company context from session (for dashboard routes)
	companyID, _ := c.Locals("company_id").(string)

	// If no session company, check query parameter (for public routes with optional filtering)
	if companyID == "" {
		companyID = c.Query("company_id")
	}

	// Get optional page_id filter
	pageID := c.Query("page_id")

	if hierarchical {
		if companyID != "" {
			// Get comments with company authorization
			comments, err := services.GetPostCommentsHierarchicalForCompany(ctx, postID, companyID)
			if err != nil {
				slog.Error("Failed to get hierarchical comments for company",
					"error", err,
					"postID", postID,
					"companyID", companyID,
				)
				// Check if it's an authorization error
				if err.Error() == "post does not belong to company" {
					return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
						"error": "Access denied: post does not belong to your company",
					})
				}
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": err.Error(),
				})
			}

			return c.JSON(fiber.Map{
				"post_id":      postID,
				"company_id":   companyID,
				"comments":     comments,
				"count":        len(comments),
				"hierarchical": true,
			})
		} else if pageID != "" {
			// Filter by specific page ID
			comments, err := services.GetPostCommentsHierarchicalWithFilter(ctx, postID, []string{pageID})
			if err != nil {
				slog.Error("Failed to get hierarchical comments for page",
					"error", err,
					"postID", postID,
					"pageID", pageID,
				)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to get comments",
				})
			}

			return c.JSON(fiber.Map{
				"post_id":      postID,
				"page_id":      pageID,
				"comments":     comments,
				"count":        len(comments),
				"hierarchical": true,
			})
		} else {
			// Get all comments without authorization check
			comments, err := services.GetPostCommentsHierarchical(ctx, postID)
			if err != nil {
				slog.Error("Failed to get hierarchical comments",
					"error", err,
					"postID", postID,
				)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to get comments",
				})
			}

			return c.JSON(fiber.Map{
				"post_id":      postID,
				"comments":     comments,
				"count":        len(comments),
				"hierarchical": true,
			})
		}
	} else {
		// Get flat list of comments
		if companyID != "" {
			comments, err := services.GetPostCommentsForCompany(ctx, postID, companyID)
			if err != nil {
				slog.Error("Failed to get comments for company",
					"error", err,
					"postID", postID,
					"companyID", companyID,
				)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": err.Error(),
				})
			}

			return c.JSON(fiber.Map{
				"post_id":      postID,
				"comments":     comments,
				"count":        len(comments),
				"hierarchical": false,
			})
		} else {
			comments, err := services.GetPostComments(ctx, postID)
			if err != nil {
				slog.Error("Failed to get comments",
					"error", err,
					"postID", postID,
				)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to get comments",
				})
			}

			return c.JSON(fiber.Map{
				"post_id":      postID,
				"comments":     comments,
				"count":        len(comments),
				"hierarchical": false,
			})
		}
	}
}

// GetCommentWithRepliesHandler handles requests to get a specific comment with its replies
func GetCommentWithRepliesHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	commentID := c.Params("commentID")
	if commentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "comment_id is required",
		})
	}

	comment, replies, err := services.GetCommentWithReplies(ctx, commentID)
	if err != nil {
		slog.Error("Failed to get comment with replies",
			"error", err,
			"commentID", commentID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get comment",
		})
	}

	if comment == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Comment not found",
		})
	}

	return c.JSON(fiber.Map{
		"comment":     comment,
		"replies":     replies,
		"reply_count": len(replies),
	})
}

// GetThreadedCommentsHandler returns comments organized by threads
func GetThreadedCommentsHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	postID := c.Params("postID")
	if postID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "post_id is required",
		})
	}

	threads, err := services.GetPostCommentsThreaded(ctx, postID)
	if err != nil {
		slog.Error("Failed to get threaded comments",
			"error", err,
			"postID", postID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get comments",
		})
	}

	return c.JSON(fiber.Map{
		"post_id": postID,
		"threads": threads,
	})
}

// GetPostCommentsDebugHandler returns detailed debug info about comments structure
func GetPostCommentsDebugHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	postID := c.Params("postID")
	if postID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "post_id is required",
		})
	}

	// Get all comments flat
	comments, err := services.GetPostComments(ctx, postID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get comments",
		})
	}

	// Build debug info
	debugInfo := make([]fiber.Map, 0, len(comments))
	for _, comment := range comments {
		debugInfo = append(debugInfo, fiber.Map{
			"comment_id":     comment.CommentID,
			"parent_id":      comment.ParentID,
			"post_id":        comment.PostID,
			"is_reply":       comment.IsReply,
			"is_bot":         comment.IsBot,
			"sender_name":    comment.SenderName,
			"message":        comment.Message,
			"has_parent":     comment.ParentID != "",
			"parent_is_post": comment.ParentID == postID,
		})
	}

	// Also get hierarchical for comparison
	hierarchical, _ := services.GetPostCommentsHierarchical(ctx, postID)

	return c.JSON(fiber.Map{
		"post_id":                 postID,
		"total_comments":          len(comments),
		"comments_debug":          debugInfo,
		"hierarchical_root_count": len(hierarchical),
	})
}

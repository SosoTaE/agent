package webhooks

import (
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"facebook-bot/config"
	"facebook-bot/handlers"
)

func RegisterRoutes(app *fiber.App, cfg *config.Config) {
	webhook := app.Group("/webhook")

	// Webhook verification endpoint
	webhook.Get("/", verifyWebhook(cfg))

	// Webhook event handler
	webhook.Post("/", handleWebhookEvent(cfg))
}

// verifyWebhook handles Facebook webhook verification
func verifyWebhook(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		mode := c.Query("hub.mode")
		token := c.Query("hub.verify_token")
		challenge := c.Query("hub.challenge")

		if mode == "subscribe" && token == cfg.VerifyToken {
			slog.Info("Webhook verified successfully")
			return c.SendString(challenge)
		}

		slog.Warn("Webhook verification failed", "mode", mode, "token", token)
		return c.SendStatus(fiber.StatusForbidden)
	}
}

// handleWebhookEvent processes incoming webhook events
func handleWebhookEvent(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body WebhookEvent
		if err := c.BodyParser(&body); err != nil {
			slog.Error("Failed to parse webhook body", "error", err)
			return c.SendStatus(fiber.StatusBadRequest)
		}

		// Only process page events
		if body.Object != "page" {
			return c.SendStatus(fiber.StatusNotFound)
		}

		// Process webhook asynchronously
		go processWebhookEvent(body)

		// Return immediately to Facebook
		return c.SendString("EVENT_RECEIVED")
	}
}

// processWebhookEvent handles the webhook processing in a separate goroutine
func processWebhookEvent(body WebhookEvent) {
	// Process each entry
	for _, entry := range body.Entry {
		pageID := entry.ID

		slog.Info("Processing webhook for page", "pageID", pageID)
		fmt.Printf("%+v\n", body)

		// Handle messaging events
		for _, messaging := range entry.Messaging {
			if messaging.Message != nil {
				// Convert webhooks.Messaging to handlers.Messaging
				handlerMessaging := handlers.Messaging{
					Sender: handlers.User{
						ID: messaging.Sender.ID,
					},
					Recipient: handlers.User{
						ID: messaging.Recipient.ID,
					},
					Timestamp: messaging.Timestamp,
				}

				// Convert message if present
				if messaging.Message != nil {
					handlerMessage := &handlers.Message{
						MID:  messaging.Message.MID,
						Text: messaging.Message.Text,
					}

					// Convert quick reply if present
					if messaging.Message.QuickReply != nil {
						handlerMessage.QuickReply = &handlers.QuickReply{
							Payload: messaging.Message.QuickReply.Payload,
						}
					}

					// Convert attachments if present
					if len(messaging.Message.Attachments) > 0 {
						handlerMessage.Attachments = make([]handlers.Attachment, len(messaging.Message.Attachments))
						for i, att := range messaging.Message.Attachments {
							handlerMessage.Attachments[i] = handlers.Attachment{
								Type: att.Type,
								Payload: handlers.Payload{
									URL: att.Payload.URL,
								},
							}
						}
					}

					handlerMessaging.Message = handlerMessage
				}

				// Process message synchronously within this goroutine
				handlers.HandleMessage(handlerMessaging, pageID)
			}
		}

		// Handle comment events
		for _, change := range entry.Changes {
			if change.Field == "feed" && change.Value.Item == "comment" {
				// Extract sender information from From field (primary) or fallback fields
				senderID := change.Value.SenderID
				senderName := change.Value.SenderName

				// Prefer From field if available (Facebook's primary structure)
				if change.Value.From != nil {
					if change.Value.From.ID != "" {
						senderID = change.Value.From.ID
					}
					if change.Value.From.Name != "" {
						senderName = change.Value.From.Name
					}
				}

				// Log the extracted sender information
				slog.Info("Extracted sender from webhook",
					"commentID", change.Value.CommentID,
					"senderID", senderID,
					"senderName", senderName,
					"hasFromField", change.Value.From != nil,
				)

				// Convert webhooks.ChangeValue to handlers.ChangeValue
				handlerChange := handlers.ChangeValue{
					Item:        change.Value.Item,
					CommentID:   change.Value.CommentID,
					PostID:      change.Value.PostID,
					ParentID:    change.Value.ParentID,
					SenderID:    senderID,
					SenderName:  senderName,
					Message:     change.Value.Message,
					CreatedTime: change.Value.CreatedTime, // Already int64, no conversion needed
				}

				// Pass From field if available
				if change.Value.From != nil {
					handlerChange.From = &handlers.FacebookUser{
						ID:   change.Value.From.ID,
						Name: change.Value.From.Name,
					}
				}

				// Process comment synchronously within this goroutine
				handlers.HandleComment(handlerChange, pageID)
			}
		}
	}
}

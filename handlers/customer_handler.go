package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"log/slog"

	"facebook-bot/models"
	"facebook-bot/services"
)

// GetCustomers retrieves customers for a company or page
func GetCustomers(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get query parameters
	pageID := c.Query("page_id")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	if limit > 200 {
		limit = 200
	}
	skip := (page - 1) * limit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// If page_id is provided, verify it belongs to company
	if pageID != "" {
		// Validate page belongs to company
		_, err := services.ValidatePageOwnership(ctx, pageID, companyID.(string))
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Page not found or access denied",
			})
		}

		// Get customers for specific page
		customers, totalCount, err := services.GetCustomersByPage(ctx, pageID, limit, skip)
		if err != nil {
			slog.Error("Failed to get customers by page", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve customers",
			})
		}

		// Calculate pagination info
		totalPages := (int(totalCount) + limit - 1) / limit
		hasMore := page < totalPages

		return c.JSON(fiber.Map{
			"page_id":   pageID,
			"customers": customers,
			"pagination": fiber.Map{
				"page":        page,
				"limit":       limit,
				"total":       totalCount,
				"total_pages": totalPages,
				"has_more":    hasMore,
			},
		})
	}

	// Get all customers for company
	customers, totalCount, err := services.GetCustomersByCompany(ctx, companyID.(string), limit, skip)
	if err != nil {
		slog.Error("Failed to get customers by company", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve customers",
		})
	}

	// Calculate pagination info
	totalPages := (int(totalCount) + limit - 1) / limit
	hasMore := page < totalPages

	return c.JSON(fiber.Map{
		"company_id": companyID,
		"customers":  customers,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_more":    hasMore,
		},
	})
}

// GetCustomerDetails retrieves details for a specific customer
func GetCustomerDetails(c *fiber.Ctx) error {
	// Get customer_id from URL params
	customerID := c.Params("customerID")
	if customerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Customer ID is required",
		})
	}

	// Get page_id from query params
	pageID := c.Query("page_id")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID is required",
		})
	}

	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify page belongs to company
	// Validate page belongs to company
	_, err := services.ValidatePageOwnership(ctx, pageID, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Get customer details
	customer, err := services.GetCustomer(ctx, customerID, pageID)
	if err != nil {
		slog.Error("Failed to get customer", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve customer",
		})
	}

	if customer == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Customer not found",
		})
	}

	return c.JSON(fiber.Map{
		"customer": customer,
	})
}

// SearchCustomers searches for customers by name or ID
func SearchCustomers(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get search query
	searchTerm := c.Query("q")
	if searchTerm == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Search query is required",
		})
	}

	// Get limit
	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Search customers
	customers, err := services.SearchCustomers(ctx, companyID.(string), searchTerm, limit)
	if err != nil {
		slog.Error("Failed to search customers", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search customers",
		})
	}

	return c.JSON(fiber.Map{
		"query":     searchTerm,
		"customers": customers,
		"count":     len(customers),
	})
}

// GetCustomerStats retrieves customer statistics
func GetCustomerStats(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Page ID is optional - if provided, get stats for specific page
	pageID := c.Query("page_id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// If page ID provided, verify it belongs to company
	if pageID != "" {
		// Validate page belongs to company
		_, err := services.ValidatePageOwnership(ctx, pageID, companyID.(string))
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Page not found or access denied",
			})
		}
	}

	// Get customer statistics
	stats, err := services.GetCustomerStats(ctx, companyID.(string))
	if err != nil {
		slog.Error("Failed to get customer stats", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve customer statistics",
		})
	}

	// Add customers requesting human assistance count
	stoppedCustomers, err := services.GetStoppedCustomersCount(ctx, companyID.(string))
	if err == nil {
		stats["stopped_customers"] = stoppedCustomers
	}

	return c.JSON(stats)
}

// UpdateCustomerStopStatus updates the stop field for a customer who wants to talk to a real person
func UpdateCustomerStopStatus(c *fiber.Ctx) error {
	// Get customer_id from URL params
	customerID := c.Params("customerID")
	if customerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Customer ID is required",
		})
	}

	// Get page_id from body
	var reqBody struct {
		PageID string `json:"page_id"`
		Stop   bool   `json:"stop"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if reqBody.PageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID is required",
		})
	}

	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify page belongs to company
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	pageFound := false
	companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
	for _, p := range companyPages {
		if p.PageID == reqBody.PageID {
			pageFound = true
			break
		}
	}
	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Update customer stop status
	customer, err := services.UpdateCustomerStopStatus(ctx, customerID, reqBody.PageID, reqBody.Stop)
	if err != nil {
		slog.Error("Failed to update customer stop status", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update customer status",
		})
	}

	if customer == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Customer not found",
		})
	}

	return c.JSON(fiber.Map{
		"message":  "Customer status updated successfully",
		"customer": customer,
	})
}

// ToggleCustomerStopStatus toggles the stop field for a customer
func ToggleCustomerStopStatus(c *fiber.Ctx) error {
	// Get customer_id from URL params
	customerID := c.Params("customerID")
	if customerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Customer ID is required",
		})
	}

	// Get page_id from body
	var reqBody struct {
		PageID string `json:"page_id"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if reqBody.PageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID is required",
		})
	}

	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify page belongs to company
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	pageFound := false
	companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
	for _, p := range companyPages {
		if p.PageID == reqBody.PageID {
			pageFound = true
			break
		}
	}
	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Get current customer status
	customer, err := services.GetCustomer(ctx, customerID, reqBody.PageID)
	if err != nil || customer == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Customer not found",
		})
	}

	// Toggle the stop status
	newStopStatus := !customer.Stop

	// Update customer stop status
	updatedCustomer, err := services.UpdateCustomerStopStatus(ctx, customerID, reqBody.PageID, newStopStatus)
	if err != nil {
		slog.Error("Failed to toggle customer stop status", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to toggle customer status",
		})
	}

	return c.JSON(fiber.Map{
		"message":       "Customer status toggled successfully",
		"customer":      updatedCustomer,
		"previous_stop": customer.Stop,
		"current_stop":  newStopStatus,
	})
}

// SendMessageToCustomer sends a message from dashboard to a customer
func SendMessageToCustomer(c *fiber.Ctx) error {
	// Get customer_id from URL params
	customerID := c.Params("customerID")
	if customerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Customer ID is required",
		})
	}

	// Parse request body
	var reqBody struct {
		PageID  string `json:"page_id"`
		Message string `json:"message"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if reqBody.PageID == "" || reqBody.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Page ID and message are required",
		})
	}

	// Check authentication and get user info
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get agent information from session
	agentEmail, _ := c.Locals("user_email").(string)
	agentName, _ := c.Locals("user_name").(string)
	agentID, _ := c.Locals("user_id").(string)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify page belongs to company and get page config
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	var pageConfig *models.FacebookPage
	companyPages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)
	for _, p := range companyPages {
		if p.PageID == reqBody.PageID {
			pageConfig = &p
			break
		}
	}

	if pageConfig == nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Check if customer exists and has stop=true
	customer, err := services.GetCustomer(ctx, customerID, reqBody.PageID)
	if err != nil || customer == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Customer not found",
		})
	}

	if !customer.Stop {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Customer has not requested human assistance. Set stop=true first.",
		})
	}

	// Send message via Facebook Messenger
	slog.Info("Attempting to send message to customer",
		"customerID", customerID,
		"pageID", reqBody.PageID,
		"hasToken", pageConfig.PageAccessToken != "")

	if err := services.SendMessengerReply(ctx, customerID, reqBody.Message, pageConfig.PageAccessToken); err != nil {
		slog.Error("Failed to send message to customer",
			"customerID", customerID,
			"pageID", reqBody.PageID,
			"error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   fmt.Sprintf("Failed to send message to customer: %v", err),
			"details": err.Error(),
		})
	}

	// Save the message to database with agent information
	messageDoc := &models.Message{
		Type:        "chat",
		ChatID:      customerID,
		SenderID:    reqBody.PageID,
		RecipientID: customerID,
		PageID:      reqBody.PageID,
		PageName:    pageConfig.PageName,
		Message:     reqBody.Message,
		IsBot:       false,
		IsHuman:     true,
		AgentID:     agentID,
		AgentEmail:  agentEmail,
		AgentName:   agentName,
		Timestamp:   time.Now(),
	}

	if err := services.SaveMessage(ctx, messageDoc); err != nil {
		slog.Error("Failed to save dashboard message", "error", err)
		// Don't return error since message was sent successfully
	}

	// Broadcast message via WebSocket
	wsManager := services.GetWebSocketManager()
	wsManager.BroadcastToCompany(companyID.(string), services.BroadcastMessage{
		CompanyID: companyID.(string),
		PageID:    reqBody.PageID,
		Type:      "new_message",
		Data: map[string]interface{}{
			"chat_id":      customerID,
			"sender_id":    reqBody.PageID,
			"recipient_id": customerID,
			"message":      reqBody.Message,
			"is_bot":       false,
			"is_human":     true,
			"agent_id":     agentID,
			"agent_email":  agentEmail,
			"agent_name":   agentName,
			"timestamp":    time.Now().Unix(),
		},
	})

	slog.Info("Dashboard message sent successfully",
		"customerID", customerID,
		"pageID", reqBody.PageID,
		"agentEmail", agentEmail,
		"companyID", companyID)

	return c.JSON(fiber.Map{
		"message": "Message sent successfully",
		"data": fiber.Map{
			"customer_id": customerID,
			"page_id":     reqBody.PageID,
			"message":     reqBody.Message,
			"agent":       agentEmail,
			"timestamp":   time.Now().Unix(),
		},
	})
}

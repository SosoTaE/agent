package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"

	"facebook-bot/models"
	"facebook-bot/services"
)

// GetCRMLinksForChannel gets CRM links for a specific channel
func GetCRMLinksForChannel(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get page ID and channel from query params
	pageID := c.Query("page_id")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "page_id is required",
		})
	}

	channel := c.Query("channel") // "facebook" or "messenger"
	if channel != "" && channel != "facebook" && channel != "messenger" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "channel must be 'facebook' or 'messenger'",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company configuration
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Find the page
	var pageConfig *models.FacebookPage
	for _, p := range company.Pages {
		if p.PageID == pageID {
			pageConfig = &p
			break
		}
	}

	if pageConfig == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Page not found",
		})
	}

	// Get CRM links based on channel
	var crmLinks []models.CRMLink

	if channel == "facebook" && pageConfig.FacebookConfig != nil {
		crmLinks = pageConfig.FacebookConfig.CRMLinks
		// Ensure channels field is set for legacy data
		for i := range crmLinks {
			if crmLinks[i].Channels == nil {
				crmLinks[i].Channels = map[string]bool{"facebook": true, "messenger": false}
			}
		}
	} else if channel == "messenger" && pageConfig.MessengerConfig != nil {
		crmLinks = pageConfig.MessengerConfig.CRMLinks
		// Ensure channels field is set for legacy data
		for i := range crmLinks {
			if crmLinks[i].Channels == nil {
				crmLinks[i].Channels = map[string]bool{"facebook": false, "messenger": true}
			}
		}
	} else if channel == "" {
		// Return all CRM links from legacy field if no channel specified
		crmLinks = pageConfig.CRMLinks
		// Ensure channels field is set for legacy data
		for i := range crmLinks {
			if crmLinks[i].Channels == nil {
				crmLinks[i].Channels = map[string]bool{"facebook": true, "messenger": true}
			}
		}
	}

	// Count active and inactive links
	activeCount := 0
	inactiveCount := 0
	for _, link := range crmLinks {
		if link.IsActive {
			activeCount++
		} else {
			inactiveCount++
		}
	}

	return c.JSON(fiber.Map{
		"crm_links":      crmLinks,
		"page_id":        pageID,
		"channel":        channel,
		"total":          len(crmLinks),
		"active_count":   activeCount,
		"inactive_count": inactiveCount,
	})
}

// AddCRMLink adds a new CRM link to a page
func AddCRMLink(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Parse request body
	type AddCRMLinkRequest struct {
		PageID      string            `json:"page_id"`
		Channels    []string          `json:"channels"` // Which channels to add to
		Name        string            `json:"name"`
		URL         string            `json:"url"`
		Type        string            `json:"type"` // api, webhook, database, file
		APIKey      string            `json:"api_key,omitempty"`
		Headers     map[string]string `json:"headers,omitempty"`
		Description string            `json:"description,omitempty"`
		IsActive    bool              `json:"is_active"`
	}

	var req AddCRMLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.PageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "page_id is required",
		})
	}

	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "url is required",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	// Validate channels
	if len(req.Channels) == 0 {
		req.Channels = []string{"facebook", "messenger"} // Default to both
	}
	for _, ch := range req.Channels {
		if ch != "facebook" && ch != "messenger" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("invalid channel: %s", ch),
			})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Convert channels array to map
	channelsMap := make(map[string]bool)
	channelsMap["facebook"] = false
	channelsMap["messenger"] = false
	for _, ch := range req.Channels {
		if ch == "facebook" || ch == "messenger" {
			channelsMap[ch] = true
		}
	}

	// Create CRM link object
	crmLink := models.CRMLink{
		Name:        req.Name,
		URL:         req.URL,
		Type:        req.Type,
		Channels:    channelsMap,
		APIKey:      req.APIKey,
		Headers:     req.Headers,
		Description: req.Description,
		IsActive:    req.IsActive,
	}

	// Add CRM link to specified channels
	err := services.AddCRMLinkToChannels(ctx, companyID.(string), req.PageID, req.Channels, crmLink)
	if err != nil {
		slog.Error("Failed to add CRM link",
			"error", err,
			"companyID", companyID,
			"pageID", req.PageID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to add CRM link: %v", err),
		})
	}

	// If the CRM link is active and of type "api", trigger initial data fetch
	if crmLink.IsActive && crmLink.Type == "api" {
		go func() {
			fetchCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			// Fetch and store CRM data for each channel
			for _, channel := range req.Channels {
				if err := services.FetchAndStoreCRMDataForChannel(fetchCtx, companyID.(string), req.PageID, channel, crmLink); err != nil {
					slog.Error("Failed to fetch initial CRM data",
						"error", err,
						"url", crmLink.URL,
						"channel", channel)
				}
			}
		}()
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "CRM link added successfully",
		"crm_link": crmLink,
		"channels": req.Channels,
		"page_id":  req.PageID,
	})
}

// UpdateCRMLinkStatus toggles a CRM link's active status
func UpdateCRMLinkStatus(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Parse request body
	type UpdateStatusRequest struct {
		PageID   string   `json:"page_id"`
		URL      string   `json:"url"`
		Channels []string `json:"channels"` // Which channels to update
		IsActive bool     `json:"is_active"`
	}

	var req UpdateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.PageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "page_id is required",
		})
	}

	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "url is required",
		})
	}

	// Validate channels
	for _, ch := range req.Channels {
		if ch != "facebook" && ch != "messenger" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("invalid channel: %s", ch),
			})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Update CRM link status for specified channels
	updatedCount, err := services.UpdateCRMLinkStatusForChannels(
		ctx,
		companyID.(string),
		req.PageID,
		req.URL,
		req.Channels,
		req.IsActive,
	)

	if err != nil {
		slog.Error("Failed to update CRM link status",
			"error", err,
			"companyID", companyID,
			"pageID", req.PageID,
			"url", req.URL)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update CRM link status: %v", err),
		})
	}

	// Also update vector documents if needed
	for _, channel := range req.Channels {
		if err := services.SyncVectorDocumentsWithCRMLink(ctx, companyID.(string), req.PageID, channel, req.URL, req.IsActive); err != nil {
			slog.Warn("Failed to sync vector documents with CRM link",
				"error", err,
				"url", req.URL,
				"channel", channel)
		}
	}

	return c.JSON(fiber.Map{
		"message":       "CRM link status updated successfully",
		"updated_count": updatedCount,
		"url":           req.URL,
		"channels":      req.Channels,
		"is_active":     req.IsActive,
	})
}

// DeleteCRMLink removes a CRM link from specified channels
func DeleteCRMLink(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Parse request body
	type DeleteRequest struct {
		PageID   string   `json:"page_id"`
		URL      string   `json:"url"`
		Channels []string `json:"channels"` // Which channels to remove from
	}

	var req DeleteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.PageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "page_id is required",
		})
	}

	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "url is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Delete CRM link from specified channels
	deletedCount, err := services.DeleteCRMLinkFromChannels(
		ctx,
		companyID.(string),
		req.PageID,
		req.URL,
		req.Channels,
	)

	if err != nil {
		slog.Error("Failed to delete CRM link",
			"error", err,
			"companyID", companyID,
			"pageID", req.PageID,
			"url", req.URL)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete CRM link: %v", err),
		})
	}

	// Also delete associated vector documents
	for _, channel := range req.Channels {
		if err := services.DeleteVectorDocumentsByCRMURL(ctx, companyID.(string), req.PageID, channel, req.URL); err != nil {
			slog.Warn("Failed to delete vector documents for CRM link",
				"error", err,
				"url", req.URL,
				"channel", channel)
		}
	}

	return c.JSON(fiber.Map{
		"message":       "CRM link deleted successfully",
		"deleted_count": deletedCount,
		"url":           req.URL,
		"channels":      req.Channels,
	})
}

// GetRAGDocumentDetails gets detailed information about RAG documents
func GetRAGDocumentDetails(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	pageID := c.Query("page_id")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "page_id is required",
		})
	}

	// Optional filters
	channel := c.Query("channel")   // Filter by channel
	filename := c.Query("filename") // Filter by filename
	source := c.Query("source")     // Filter by source
	onlyActive := c.Query("active") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build filter
	filter := bson.M{
		"company_id": companyID.(string),
		"page_id":    pageID,
	}

	if channel != "" {
		filter["channels"] = channel
	}

	if filename != "" {
		filter["metadata.filename"] = filename
	}

	if source != "" {
		filter["source"] = source
	}

	if onlyActive {
		filter["is_active"] = true
	}

	// Get documents
	documents, err := services.GetVectorDocumentsWithFilter(ctx, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to retrieve documents: %v", err),
		})
	}

	// Group by filename and calculate statistics
	type DocumentGroup struct {
		Filename     string          `json:"filename"`
		TotalChunks  int             `json:"total_chunks"`
		ActiveChunks int             `json:"active_chunks"`
		Channels     map[string]bool `json:"channels"`
		Source       string          `json:"source"`
		UploadedBy   string          `json:"uploaded_by,omitempty"`
		UploadedAt   time.Time       `json:"uploaded_at"`
		Size         int             `json:"total_size"`
	}

	groups := make(map[string]*DocumentGroup)

	for _, doc := range documents {
		filename := doc.Metadata["filename"]
		if filename == "" {
			filename = "unnamed_" + doc.ID.Hex()[:8]
		}

		if groups[filename] == nil {
			groups[filename] = &DocumentGroup{
				Filename:   filename,
				Source:     doc.Source,
				UploadedAt: doc.CreatedAt,
				Channels:   make(map[string]bool),
			}

			if uploadedBy, ok := doc.Metadata["uploaded_by"]; ok {
				groups[filename].UploadedBy = uploadedBy
			}
		}

		group := groups[filename]
		group.TotalChunks++
		group.Size += len(doc.Content)

		if doc.IsActive {
			group.ActiveChunks++
		}

		// Merge channels - combine channel states from all chunks
		for ch, enabled := range doc.Channels {
			// If any chunk has the channel enabled, mark it as enabled for the group
			if enabled || !group.Channels[ch] {
				group.Channels[ch] = enabled
			}
		}
	}

	// Convert map to slice
	var groupList []DocumentGroup
	for _, group := range groups {
		groupList = append(groupList, *group)
	}

	// Calculate totals
	totalDocs := len(documents)
	totalGroups := len(groups)
	activeDocs := 0
	for _, doc := range documents {
		if doc.IsActive {
			activeDocs++
		}
	}

	return c.JSON(fiber.Map{
		"documents":        documents,
		"groups":           groupList,
		"total_documents":  totalDocs,
		"total_groups":     totalGroups,
		"active_documents": activeDocs,
		"page_id":          pageID,
		"filters": fiber.Map{
			"channel":  channel,
			"filename": filename,
			"source":   source,
			"active":   onlyActive,
		},
	})
}

// ToggleRAGDocumentByID toggles a specific RAG document by its ID
func ToggleRAGDocumentByID(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Parse request body
	type ToggleRequest struct {
		DocumentID string `json:"document_id"`
		IsActive   bool   `json:"is_active"`
	}

	var req ToggleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.DocumentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "document_id is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Update document status
	updatedCount, err := services.ToggleVectorDocumentByID(ctx, req.DocumentID, req.IsActive)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to toggle document: %v", err),
		})
	}

	if updatedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	return c.JSON(fiber.Map{
		"message":     "Document status updated successfully",
		"document_id": req.DocumentID,
		"is_active":   req.IsActive,
	})
}

// GetRAGDocumentsByFilename gets all chunks of a document by filename
func GetRAGDocumentsByFilename(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	pageID := c.Query("page_id")
	filename := c.Query("filename")

	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "page_id is required",
		})
	}

	if filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "filename is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all chunks for this filename
	filter := bson.M{
		"company_id":        companyID.(string),
		"page_id":           pageID,
		"metadata.filename": filename,
	}

	documents, err := services.GetVectorDocumentsWithFilter(ctx, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to retrieve documents: %v", err),
		})
	}

	// Calculate combined content
	var combinedContent string
	activeCount := 0
	totalSize := 0
	channelMap := make(map[string]bool)
	// Initialize with both channels as false
	channelMap["facebook"] = false
	channelMap["messenger"] = false

	for _, doc := range documents {
		combinedContent += doc.Content + "\n\n"
		totalSize += len(doc.Content)

		if doc.IsActive {
			activeCount++
		}

		// Merge channel states - if any chunk has channel enabled, mark it as enabled
		for ch, enabled := range doc.Channels {
			if enabled {
				channelMap[ch] = true
			}
		}
	}

	return c.JSON(fiber.Map{
		"filename":         filename,
		"chunks":           documents,
		"total_chunks":     len(documents),
		"active_chunks":    activeCount,
		"total_size":       totalSize,
		"channels":         channelMap,
		"combined_content": combinedContent,
		"page_id":          pageID,
	})
}

// GetAllCRMData gets all CRM links for all pages of a company
func GetAllCRMData(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get company with all pages
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Collect all CRM links from all pages and channels
	type PageCRMData struct {
		PageID            string           `json:"page_id"`
		PageName          string           `json:"page_name"`
		FacebookCRMLinks  []models.CRMLink `json:"facebook_crm_links"`
		MessengerCRMLinks []models.CRMLink `json:"messenger_crm_links"`
		LegacyCRMLinks    []models.CRMLink `json:"legacy_crm_links"`
		TotalActive       int              `json:"total_active"`
		TotalInactive     int              `json:"total_inactive"`
	}

	var allCRMData []PageCRMData
	totalCRMLinks := 0
	totalActive := 0
	totalInactive := 0

	for _, page := range company.Pages {
		pageData := PageCRMData{
			PageID:            page.PageID,
			PageName:          page.PageName,
			FacebookCRMLinks:  []models.CRMLink{},
			MessengerCRMLinks: []models.CRMLink{},
			LegacyCRMLinks:    page.CRMLinks,
		}

		// Get Facebook CRM links
		if page.FacebookConfig != nil {
			pageData.FacebookCRMLinks = page.FacebookConfig.CRMLinks
			for i, link := range page.FacebookConfig.CRMLinks {
				// Ensure channels field is set
				if link.Channels == nil || len(link.Channels) == 0 {
					pageData.FacebookCRMLinks[i].Channels = map[string]bool{"facebook": true}
				}
				totalCRMLinks++
				if link.IsActive {
					pageData.TotalActive++
					totalActive++
				} else {
					pageData.TotalInactive++
					totalInactive++
				}
			}
		}

		// Get Messenger CRM links
		if page.MessengerConfig != nil {
			pageData.MessengerCRMLinks = page.MessengerConfig.CRMLinks
			for i, link := range page.MessengerConfig.CRMLinks {
				// Ensure channels field is set
				if link.Channels == nil || len(link.Channels) == 0 {
					pageData.MessengerCRMLinks[i].Channels = map[string]bool{"messenger": true}
				}
				totalCRMLinks++
				if link.IsActive {
					pageData.TotalActive++
					totalActive++
				} else {
					pageData.TotalInactive++
					totalInactive++
				}
			}
		}

		// Count legacy links
		for i, link := range page.CRMLinks {
			// Ensure channels field is set for legacy links
			if link.Channels == nil || len(link.Channels) == 0 {
				pageData.LegacyCRMLinks[i].Channels = map[string]bool{"facebook": true, "messenger": true}
			}
			totalCRMLinks++
			if link.IsActive {
				pageData.TotalActive++
				totalActive++
			} else {
				pageData.TotalInactive++
				totalInactive++
			}
		}

		allCRMData = append(allCRMData, pageData)
	}

	return c.JSON(fiber.Map{
		"company_id":      companyID,
		"company_name":    company.CompanyName,
		"pages_data":      allCRMData,
		"total_pages":     len(company.Pages),
		"total_crm_links": totalCRMLinks,
		"total_active":    totalActive,
		"total_inactive":  totalInactive,
	})
}

// GetCRMLinksFromCollection retrieves CRM links directly from the crm_links collection
// It can either get links for a specific page_id (from query param) or for all pages the user has access to
func GetCRMLinksFromCollection(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if a specific page_id is provided in query params
	specificPageID := c.Query("page_id")

	var allCRMLinks []models.CRMLink
	var pageIDs []string

	if specificPageID != "" {
		// Get CRM links for the specific page
		crmLinks, err := services.GetCRMLinksByPageID(ctx, specificPageID)
		if err != nil {
			slog.Error("Failed to get CRM links from collection",
				"error", err,
				"pageID", specificPageID)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get CRM links: %v", err),
			})
		}
		allCRMLinks = crmLinks
		pageIDs = []string{specificPageID}
	} else {
		// Get all page IDs the user has access to
		companyPageIDs := c.Locals("company_page_ids")
		if companyPageIDs == nil {
			// If not set by middleware, get them directly
			var err error
			pageIDs, err = services.GetCompanyPageIDs(ctx, companyID.(string))
			if err != nil {
				slog.Error("Failed to get company pages",
					"error", err,
					"companyID", companyID)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to get company pages",
				})
			}
		} else {
			// Use the page IDs from middleware
			pageIDs, _ = companyPageIDs.([]string)
		}

		// Get CRM links for all pages
		for _, pageID := range pageIDs {
			crmLinks, err := services.GetCRMLinksByPageID(ctx, pageID)
			if err != nil {
				slog.Warn("Failed to get CRM links for page",
					"error", err,
					"pageID", pageID)
				continue
			}

			// Add page_id to each link if not already set
			for i := range crmLinks {
				if crmLinks[i].PageID == "" {
					crmLinks[i].PageID = pageID
				}
			}

			allCRMLinks = append(allCRMLinks, crmLinks...)
		}
	}

	// Generate CRM IDs for each link
	for i := range allCRMLinks {
		pageID := allCRMLinks[i].PageID
		if pageID == "" && len(pageIDs) == 1 {
			pageID = pageIDs[0]
		}

		// Generate a unique CRM ID for each link
		crmID := services.GenerateCRMID(pageID, allCRMLinks[i].URL)
		// Add CRM ID to metadata for reference
		if allCRMLinks[i].Description != "" {
			allCRMLinks[i].Description = fmt.Sprintf("%s (CRM_ID: %s)", allCRMLinks[i].Description, crmID)
		} else {
			allCRMLinks[i].Description = fmt.Sprintf("CRM_ID: %s", crmID)
		}
	}

	// Count active and inactive links
	activeCount := 0
	inactiveCount := 0
	for _, link := range allCRMLinks {
		if link.IsActive {
			activeCount++
		} else {
			inactiveCount++
		}
	}

	// Group links by page_id for better organization
	linksByPage := make(map[string][]models.CRMLink)
	for _, link := range allCRMLinks {
		pageID := link.PageID
		if pageID == "" && len(pageIDs) == 1 {
			pageID = pageIDs[0]
		}
		linksByPage[pageID] = append(linksByPage[pageID], link)
	}

	response := fiber.Map{
		"crm_links":      allCRMLinks,
		"total":          len(allCRMLinks),
		"active_count":   activeCount,
		"inactive_count": inactiveCount,
		"source":         "crm_links_collection",
		"message":        "CRM links retrieved from crm_links collection",
	}

	// Add additional info based on query
	if specificPageID != "" {
		response["page_id"] = specificPageID
	} else {
		response["page_ids"] = pageIDs
		response["links_by_page"] = linksByPage
	}

	return c.JSON(response)
}

// GetAllRAGDocuments gets all RAG documents for all pages of a company
func GetAllRAGDocuments(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get company with all pages
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Collect all RAG documents from all pages
	type PageRAGData struct {
		PageID            string                    `json:"page_id"`
		PageName          string                    `json:"page_name"`
		Documents         []services.VectorDocument `json:"documents"`
		DocumentGroups    map[string]interface{}    `json:"document_groups"`
		TotalDocuments    int                       `json:"total_documents"`
		ActiveDocuments   int                       `json:"active_documents"`
		InactiveDocuments int                       `json:"inactive_documents"`
		TotalSize         int                       `json:"total_size"`
		UniqueFiles       int                       `json:"unique_files"`
	}

	var allRAGData []PageRAGData
	totalDocuments := 0
	totalActive := 0
	totalInactive := 0
	totalSize := 0

	for _, page := range company.Pages {
		// Get all documents for this page
		filter := bson.M{
			"company_id": companyID.(string),
			"page_id":    page.PageID,
		}

		documents, err := services.GetVectorDocumentsWithFilter(ctx, filter)
		if err != nil {
			slog.Warn("Failed to get documents for page",
				"pageID", page.PageID,
				"error", err)
			continue
		}

		pageData := PageRAGData{
			PageID:         page.PageID,
			PageName:       page.PageName,
			Documents:      documents,
			DocumentGroups: make(map[string]interface{}),
		}

		// Group documents by filename
		fileGroups := make(map[string][]services.VectorDocument)
		for _, doc := range documents {
			totalDocuments++
			pageData.TotalDocuments++
			pageData.TotalSize += len(doc.Content)
			totalSize += len(doc.Content)

			if doc.IsActive {
				pageData.ActiveDocuments++
				totalActive++
			} else {
				pageData.InactiveDocuments++
				totalInactive++
			}

			// Group by filename
			filename := doc.Metadata["filename"]
			if filename == "" {
				filename = "unnamed_" + doc.ID.Hex()[:8]
			}
			fileGroups[filename] = append(fileGroups[filename], doc)
		}

		// Create document groups summary
		for filename, docs := range fileGroups {
			activeChunks := 0
			channelMap := make(map[string]bool)
			// Initialize with both channels as false
			channelMap["facebook"] = false
			channelMap["messenger"] = false
			totalChunkSize := 0

			for _, doc := range docs {
				if doc.IsActive {
					activeChunks++
				}
				totalChunkSize += len(doc.Content)

				// Merge channel states - if any chunk has channel enabled, mark it as enabled
				for ch, enabled := range doc.Channels {
					if enabled {
						channelMap[ch] = true
					}
				}
			}

			// Parse chunk metadata to get expected total chunks
			expectedTotalChunks := 0
			for _, doc := range docs {
				if chunkInfo, ok := doc.Metadata["chunk"]; ok {
					// Parse "3/324" format
					parts := strings.Split(chunkInfo, "/")
					if len(parts) == 2 {
						if total, err := strconv.Atoi(parts[1]); err == nil {
							if total > expectedTotalChunks {
								expectedTotalChunks = total
							}
						}
					}
				} else {
					// Single chunk document
					if expectedTotalChunks == 0 {
						expectedTotalChunks = 1
					}
				}
			}

			isComplete := len(docs) == expectedTotalChunks && expectedTotalChunks > 0

			pageData.DocumentGroups[filename] = map[string]interface{}{
				"stored_chunks":   len(docs),
				"expected_chunks": expectedTotalChunks,
				"is_complete":     isComplete,
				"active_chunks":   activeChunks,
				"channels":        channelMap,
				"total_size":      totalChunkSize,
				"source":          docs[0].Source,
				"created_at":      docs[0].CreatedAt,
			}
		}

		pageData.UniqueFiles = len(fileGroups)
		allRAGData = append(allRAGData, pageData)
	}

	return c.JSON(fiber.Map{
		"company_id":      companyID,
		"company_name":    company.CompanyName,
		"pages_data":      allRAGData,
		"total_pages":     len(company.Pages),
		"total_documents": totalDocuments,
		"total_active":    totalActive,
		"total_inactive":  totalInactive,
		"total_size":      totalSize,
		"size_mb":         float64(totalSize) / (1024 * 1024),
	})
}

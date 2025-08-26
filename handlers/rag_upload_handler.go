package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"facebook-bot/services"
)

// UploadRAGDocument handles file upload for RAG processing
func UploadRAGDocument(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Get page_id from form data
	pageID := c.FormValue("page_id")
	if pageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "page_id is required",
		})
	}

	// Get channels array (optional - defaults to both platforms)
	channelsStr := c.FormValue("channels")
	channels := []string{}
	channelsMap := make(map[string]bool)

	if channelsStr == "" {
		// Default to both platforms enabled
		channelsMap["facebook"] = true
		channelsMap["messenger"] = true
		channels = []string{"facebook", "messenger"}
	} else {
		// Parse channels from comma-separated string if provided
		channelsParts := strings.Split(channelsStr, ",")
		// Initialize both channels as false
		channelsMap["facebook"] = false
		channelsMap["messenger"] = false

		for _, ch := range channelsParts {
			ch = strings.ToLower(strings.TrimSpace(ch))
			// Normalize channel name
			if ch == "fb" {
				ch = "facebook"
			} else if ch == "messanger" || ch == "msg" {
				ch = "messenger"
			}

			if ch != "facebook" && ch != "messenger" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("invalid channel: %s (must be 'facebook' or 'messenger')", ch),
				})
			}

			// Set channel to true in map
			channelsMap[ch] = true

			// Add to channels array for backward compatibility with existing functions
			found := false
			for _, existing := range channels {
				if existing == ch {
					found = true
					break
				}
			}
			if !found {
				channels = append(channels, ch)
			}
		}

		// Check if at least one channel is enabled when explicitly provided
		hasEnabledChannel := false
		for _, enabled := range channelsMap {
			if enabled {
				hasEnabledChannel = true
				break
			}
		}

		if !hasEnabledChannel {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "at least one valid channel is required when channels parameter is provided",
			})
		}
	}

	// Get source type (optional, defaults to "manual_upload")
	source := c.FormValue("source")
	if source == "" {
		source = "manual_upload"
	}

	// Get metadata (optional)
	metadataStr := c.FormValue("metadata")
	metadata := make(map[string]string)
	if metadataStr != "" {
		// Parse simple key=value pairs separated by commas
		pairs := strings.Split(metadataStr, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				metadata[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Verify company and page ownership
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Check if page belongs to company
	pageFound := false
	for _, p := range company.Pages {
		if p.PageID == pageID {
			pageFound = true
			break
		}
	}
	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Validate file size (max 10MB)
	if file.Size > 10*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File size exceeds 10MB limit",
		})
	}

	// Process file based on type
	content, err := extractTextFromFile(file)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to extract text from file: %v", err),
		})
	}

	if content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No text content found in file",
		})
	}

	// Store metadata about the file
	metadata["filename"] = file.Filename
	metadata["upload_time"] = time.Now().Format(time.RFC3339)

	// Get user email if available
	userEmail := ""
	if email := c.Locals("user_email"); email != nil {
		userEmail = email.(string)
		metadata["uploaded_by"] = userEmail
	}

	// Log that we received the file
	slog.Info("RAG document received, processing in background",
		"companyID", company.CompanyID,
		"pageID", pageID,
		"filename", file.Filename,
		"contentLength", len(content),
	)

	// Process embeddings in background
	go processRAGDocumentInBackground(
		company.CompanyID,
		pageID,
		channels,
		content,
		source,
		metadata,
		file.Filename,
		userEmail,
	)

	// Return immediately
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":  "File received and queued for processing",
		"filename": file.Filename,
		"size":     file.Size,
		"source":   source,
		"page_id":  pageID,
		"channels": channels,
		"status":   "processing",
	})
}

// processRAGDocumentInBackground processes document embeddings in the background
func processRAGDocumentInBackground(companyID, pageID string, channels []string, content, source string, metadata map[string]string, filename, userEmail string) {
	// Create a new context for background processing
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	slog.Info("Starting background RAG document processing",
		"companyID", companyID,
		"pageID", pageID,
		"filename", filename,
		"contentLength", len(content),
	)

	// Split content into chunks if it's too large
	chunks := splitIntoChunks(content, 2000) // 2000 chars per chunk

	var storedCount int
	var errors []string

	for i, chunk := range chunks {
		// Add chunk info to metadata
		chunkMetadata := make(map[string]string)
		for k, v := range metadata {
			chunkMetadata[k] = v
		}
		if len(chunks) > 1 {
			chunkMetadata["chunk"] = fmt.Sprintf("%d/%d", i+1, len(chunks))
		}

		// Store embedding for this chunk with channels
		err := services.StoreEmbeddingsWithChannelsAndOptions(
			ctx,
			companyID,
			pageID,
			channels,
			chunk,
			source,
			chunkMetadata,
			"",   // No CRM URL for manual uploads
			true, // Set as active by default
		)

		if err != nil {
			errors = append(errors, fmt.Sprintf("Chunk %d: %v", i+1, err))
			slog.Error("Failed to store embedding for chunk",
				"chunk", i+1,
				"error", err,
				"companyID", companyID,
				"pageID", pageID,
				"filename", filename,
			)
		} else {
			storedCount++
			slog.Debug("Stored embedding for chunk",
				"chunk", i+1,
				"total", len(chunks),
				"filename", filename,
			)
		}
	}

	// Log completion
	if len(errors) > 0 {
		slog.Warn("RAG document processing completed with errors",
			"companyID", companyID,
			"pageID", pageID,
			"filename", filename,
			"chunks_total", len(chunks),
			"chunks_stored", storedCount,
			"errors", len(errors),
			"error_details", errors,
		)
	} else {
		slog.Info("RAG document processing completed successfully",
			"companyID", companyID,
			"pageID", pageID,
			"filename", filename,
			"chunks_total", len(chunks),
			"chunks_stored", storedCount,
		)
	}

	// TODO: You could send a webhook or notification here to inform about completion
}

// extractTextFromFile extracts text content from uploaded file
func extractTextFromFile(file *multipart.FileHeader) (string, error) {
	// Open the file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Check file extension
	filename := strings.ToLower(file.Filename)

	// For now, we'll support plain text files
	// You can add support for PDF, DOCX, etc. later
	if strings.HasSuffix(filename, ".txt") ||
		strings.HasSuffix(filename, ".md") ||
		strings.HasSuffix(filename, ".csv") ||
		strings.HasSuffix(filename, ".json") {
		// Read text content
		content, err := io.ReadAll(src)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return string(content), nil
	}

	return "", fmt.Errorf("unsupported file type. Supported types: .txt, .md, .csv, .json")
}

// splitIntoChunks splits text into chunks of specified size
func splitIntoChunks(text string, chunkSize int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	words := strings.Fields(text)

	var currentChunk strings.Builder
	for _, word := range words {
		// Check if adding this word would exceed chunk size
		if currentChunk.Len()+len(word)+1 > chunkSize {
			// Save current chunk if it has content
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
			}
		}

		// Add word to current chunk
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(word)
	}

	// Add the last chunk if it has content
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// DeleteRAGDocument deletes a document from vector database
func DeleteRAGDocument(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	var req struct {
		PageID   string `json:"page_id"`
		Content  string `json:"content,omitempty"`  // For content-based deletion
		Filename string `json:"filename,omitempty"` // For filename-based deletion
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify company and page ownership
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Check if page belongs to company
	pageFound := false
	for _, p := range company.Pages {
		if p.PageID == req.PageID {
			pageFound = true
			break
		}
	}
	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Delete document based on provided criteria
	var deletedCount int64
	if req.Filename != "" {
		// Delete by filename in metadata
		deletedCount, err = services.DeleteVectorDocumentsByMetadata(ctx, company.CompanyID, req.PageID, "filename", req.Filename)
	} else if req.Content != "" {
		// Delete by content
		deletedCount, err = services.DeleteVectorDocumentByContent(ctx, company.CompanyID, req.PageID, req.Content)
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Either content or filename must be provided",
		})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete documents: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Successfully deleted %d document(s)", deletedCount),
		"deleted": deletedCount,
	})
}

// UpdateRAGDocumentChannels updates which channels a document is active for
func UpdateRAGDocumentChannels(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Parse request body - now expects platform name and value
	type UpdateRequest struct {
		CompanyID string `json:"company_id"`
		PageID    string `json:"page_id"`
		Filename  string `json:"filename"`
		Platform  string `json:"platform"` // "facebook" or "messenger"
		Value     bool   `json:"value"`    // true or false
	}

	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.CompanyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "company_id is required",
		})
	}

	if req.PageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "page_id is required",
		})
	}

	if req.Filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "filename is required",
		})
	}

	// Validate platform
	if req.Platform != "facebook" && req.Platform != "messenger" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("invalid platform: %s (must be 'facebook' or 'messenger')", req.Platform),
		})
	}

	// Verify the company ID matches the authenticated one
	if req.CompanyID != companyID.(string) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied: company_id mismatch",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Update the specific channel value for the document
	result, err := services.UpdateVectorDocumentChannelValue(ctx, req.CompanyID, req.PageID, req.Filename, req.Platform, req.Value)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update document channel: %v", err),
		})
	}

	if result == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No documents found with the specified filename",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":       "Document channel updated successfully",
		"updated_count": result,
		"platform":      req.Platform,
		"value":         req.Value,
		"filename":      req.Filename,
	})
}

// ToggleRAGDocument toggles the active status of RAG documents
func ToggleRAGDocument(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	var req struct {
		PageID     string `json:"page_id"`
		Filename   string `json:"filename,omitempty"`    // Toggle by filename
		DocumentID string `json:"document_id,omitempty"` // Toggle by document ID
		IsActive   bool   `json:"is_active"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	fmt.Println(req)

	if req.PageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "page_id is required",
		})
	}

	if req.Filename == "" && req.DocumentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Either filename or document_id must be provided",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify company and page ownership
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Check if page belongs to company
	pageFound := false
	for _, p := range company.Pages {
		if p.PageID == req.PageID {
			pageFound = true
			break
		}
	}
	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Toggle document status
	var updatedCount int64
	if req.DocumentID != "" {
		// Toggle by document ID
		updatedCount, err = services.ToggleVectorDocumentByID(ctx, req.DocumentID, req.IsActive)
	} else if req.Filename != "" {
		// Toggle all documents with this filename (all chunks)
		updatedCount, err = services.ToggleVectorDocumentsByMetadata(ctx, company.CompanyID, req.PageID, "filename", req.Filename, req.IsActive)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to toggle document status: %v", err),
		})
	}

	if updatedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No documents found to update",
		})
	}

	slog.Info("RAG document status toggled",
		"companyID", company.CompanyID,
		"pageID", req.PageID,
		"filename", req.Filename,
		"documentID", req.DocumentID,
		"isActive", req.IsActive,
		"updatedCount", updatedCount,
	)

	return c.JSON(fiber.Map{
		"message":   fmt.Sprintf("Successfully updated %d document(s)", updatedCount),
		"updated":   updatedCount,
		"is_active": req.IsActive,
	})
}

// ListRAGDocuments lists all RAG documents for a page
func ListRAGDocuments(c *fiber.Ctx) error {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify company and page ownership
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Check if page belongs to company
	pageFound := false
	for _, p := range company.Pages {
		if p.PageID == pageID {
			pageFound = true
			break
		}
	}
	if !pageFound {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Page not found or access denied",
		})
	}

	// Get documents from vector database
	documents, err := services.GetVectorDocuments(ctx, company.CompanyID, pageID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to retrieve documents: %v", err),
		})
	}

	// Enhanced document groups structure
	type DocumentGroupInfo struct {
		Filename     string          `json:"filename"`
		StoredChunks int             `json:"stored_chunks"`
		TotalChunks  int             `json:"total_chunks"`
		IsComplete   bool            `json:"is_complete"`
		Channels     map[string]bool `json:"channels"`
		Source       string          `json:"source"`
		Documents    []fiber.Map     `json:"documents"`
	}

	// Group documents by filename if they were uploaded as chunks
	documentGroups := make(map[string]*DocumentGroupInfo)

	for _, doc := range documents {
		// Normalize channels to ensure correct values
		channels := doc.Channels
		if channels == nil {
			channels = make(map[string]bool)
		}
		// Fix any misspellings in channels (create a new normalized map)
		normalizedChannels := make(map[string]bool)
		for ch, enabled := range channels {
			if ch == "messanger" {
				normalizedChannels["messenger"] = enabled
			} else {
				normalizedChannels[ch] = enabled
			}
		}
		channels = normalizedChannels

		// Create document info
		docInfo := fiber.Map{
			"id":         doc.ID.Hex(),
			"content":    doc.Content,
			"source":     doc.Source,
			"is_active":  doc.IsActive,
			"created_at": doc.CreatedAt,
			"metadata":   doc.Metadata,
			"channels":   channels,
		}

		// Group by filename if available
		groupKey := ""
		if filename, ok := doc.Metadata["filename"]; ok {
			groupKey = filename
		} else {
			// Use content preview as key for non-file documents
			groupKey = doc.Content
			if len(groupKey) > 50 {
				groupKey = groupKey[:50] + "..."
			}
		}

		// Initialize group if not exists
		if documentGroups[groupKey] == nil {
			documentGroups[groupKey] = &DocumentGroupInfo{
				Filename:     groupKey,
				StoredChunks: 0,
				TotalChunks:  0,
				IsComplete:   false,
				Channels:     make(map[string]bool),
				Source:       doc.Source,
				Documents:    []fiber.Map{},
			}
		}

		// Add document to group
		documentGroups[groupKey].Documents = append(documentGroups[groupKey].Documents, docInfo)
		documentGroups[groupKey].StoredChunks++

		// Parse chunk metadata to get total chunks
		if chunkInfo, ok := doc.Metadata["chunk"]; ok {
			// Parse "3/324" format
			parts := strings.Split(chunkInfo, "/")
			if len(parts) == 2 {
				if total, err := strconv.Atoi(parts[1]); err == nil {
					// Update total chunks if we found a higher number
					if total > documentGroups[groupKey].TotalChunks {
						documentGroups[groupKey].TotalChunks = total
					}
				}
			}
		} else {
			// Single chunk document
			documentGroups[groupKey].TotalChunks = 1
		}

		// Merge channels - if any chunk has channel enabled, mark as enabled
		for ch, enabled := range channels {
			if enabled {
				documentGroups[groupKey].Channels[ch] = true
			}
		}
	}

	// Check completeness for each group
	for _, group := range documentGroups {
		group.IsComplete = group.StoredChunks == group.TotalChunks && group.TotalChunks > 0
	}

	return c.JSON(fiber.Map{
		"documents":       documents,
		"document_groups": documentGroups,
		"total":           len(documents),
	})
}

// ToggleDocumentChannel is a simplified endpoint to toggle a document's platform status
func ToggleDocumentChannel(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	// Parse request body - simplified structure
	type ToggleRequest struct {
		DocumentName string `json:"document_name"` // Filename of the document
		Platform     string `json:"platform_name"` // "facebook" or "messenger"
		Value        bool   `json:"value"`         // true or false
	}

	var req ToggleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.DocumentName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "document_name is required",
		})
	}

	if req.Platform == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "platform_name is required",
		})
	}

	// Normalize and validate platform
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	if platform != "facebook" && platform != "messenger" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("invalid platform_name: %s (must be 'facebook' or 'messenger')", req.Platform),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the company to find all page IDs
	company, err := services.GetCompanyByID(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Update the channel value for documents with this filename across all pages
	totalUpdated := int64(0)
	updatedPages := []string{}

	for _, page := range company.Pages {
		// Update the specific channel value for documents in this page
		result, err := services.UpdateVectorDocumentChannelValue(
			ctx,
			company.CompanyID,
			page.PageID,
			req.DocumentName,
			platform,
			req.Value,
		)

		if err != nil {
			slog.Warn("Failed to update document channel for page",
				"error", err,
				"pageID", page.PageID,
				"documentName", req.DocumentName,
			)
			continue
		}

		if result > 0 {
			totalUpdated += result
			updatedPages = append(updatedPages, page.PageID)
		}
	}

	if totalUpdated == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("No documents found with name '%s'", req.DocumentName),
		})
	}

	slog.Info("Document channel toggled",
		"companyID", company.CompanyID,
		"documentName", req.DocumentName,
		"platform", platform,
		"value", req.Value,
		"totalUpdated", totalUpdated,
		"pagesUpdated", len(updatedPages),
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":       "Document channel updated successfully",
		"document_name": req.DocumentName,
		"platform":      platform,
		"value":         req.Value,
		"updated_count": totalUpdated,
		"pages_updated": updatedPages,
	})
}

// GetAllRAGFiles returns all unique RAG filenames with channel data
func GetAllRAGFiles(c *fiber.Ctx) error {
	// Check authentication
	companyID := c.Locals("company_id")
	if companyID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Company ID not found in session",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get all documents for the company
	documents, err := services.GetAllVectorDocumentsByCompany(ctx, companyID.(string))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get documents: %v", err),
		})
	}

	// Enhanced structure for response
	type FileChannelData struct {
		Filename     string `json:"filename"`
		Facebook     bool   `json:"facebook"`
		Messenger    bool   `json:"messenger"`
		StoredChunks int    `json:"stored_chunks"`
		TotalChunks  int    `json:"total_chunks"`
		IsComplete   bool   `json:"is_complete"`
	}

	// Map to track unique files
	filesMap := make(map[string]*FileChannelData)

	// Process documents
	for _, doc := range documents {
		if filename, ok := doc.Metadata["filename"]; ok && filename != "" {
			if filesMap[filename] == nil {
				filesMap[filename] = &FileChannelData{
					Filename:     filename,
					Facebook:     false,
					Messenger:    false,
					StoredChunks: 0,
					TotalChunks:  0,
					IsComplete:   false,
				}
			}

			// Count stored chunks
			filesMap[filename].StoredChunks++

			// Parse chunk metadata to get total chunks expected
			if chunkInfo, ok := doc.Metadata["chunk"]; ok {
				// Parse "3/324" format
				parts := strings.Split(chunkInfo, "/")
				if len(parts) == 2 {
					if total, err := strconv.Atoi(parts[1]); err == nil {
						// Update total chunks if we found a higher number
						if total > filesMap[filename].TotalChunks {
							filesMap[filename].TotalChunks = total
						}
					}
				}
			} else {
				// Single chunk document
				filesMap[filename].TotalChunks = 1
			}

			// Update channel states - if any chunk has channel enabled, mark as enabled
			if doc.Channels["facebook"] {
				filesMap[filename].Facebook = true
			}
			if doc.Channels["messenger"] {
				filesMap[filename].Messenger = true
			}
		}
	}

	// Check completeness and convert to array
	var result []FileChannelData
	for _, fileData := range filesMap {
		// Mark as complete if all expected chunks are present
		fileData.IsComplete = fileData.StoredChunks == fileData.TotalChunks && fileData.TotalChunks > 0
		result = append(result, *fileData)
	}

	// Return the enhanced array
	return c.JSON(result)
}

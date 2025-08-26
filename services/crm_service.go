package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"facebook-bot/models"
)

// AddCRMLinkToChannels adds a CRM link to specified channels
func AddCRMLinkToChannels(ctx context.Context, companyID, pageID string, channels []string, crmLink models.CRMLink) error {
	db := GetDatabase()
	collection := db.Collection("companies")

	// Get the company
	filter := bson.M{
		"company_id":    companyID,
		"pages.page_id": pageID,
	}

	var company models.Company
	err := collection.FindOne(ctx, filter).Decode(&company)
	if err != nil {
		return fmt.Errorf("company or page not found: %w", err)
	}

	// Find the page index
	pageIndex := -1
	for i, page := range company.Pages {
		if page.PageID == pageID {
			pageIndex = i
			break
		}
	}

	if pageIndex == -1 {
		return fmt.Errorf("page not found")
	}

	// Prepare updates for each channel
	updates := bson.M{
		"updated_at": time.Now(),
	}

	for _, channel := range channels {
		if channel == "facebook" {
			// Initialize FacebookConfig if needed
			if company.Pages[pageIndex].FacebookConfig == nil {
				updates[fmt.Sprintf("pages.%d.facebook_config", pageIndex)] = &models.ChannelConfig{
					IsEnabled:  true,
					RAGEnabled: true,
					CRMLinks:   []models.CRMLink{crmLink},
				}
			} else {
				// Add to existing Facebook config
				updates[fmt.Sprintf("pages.%d.facebook_config.crm_links", pageIndex)] = append(
					company.Pages[pageIndex].FacebookConfig.CRMLinks,
					crmLink,
				)
			}
		}

		if channel == "messenger" {
			// Initialize MessengerConfig if needed
			if company.Pages[pageIndex].MessengerConfig == nil {
				updates[fmt.Sprintf("pages.%d.messenger_config", pageIndex)] = &models.ChannelConfig{
					IsEnabled:  true,
					RAGEnabled: true,
					CRMLinks:   []models.CRMLink{crmLink},
				}
			} else {
				// Add to existing Messenger config
				updates[fmt.Sprintf("pages.%d.messenger_config.crm_links", pageIndex)] = append(
					company.Pages[pageIndex].MessengerConfig.CRMLinks,
					crmLink,
				)
			}
		}
	}

	// Also add to legacy CRMLinks for backward compatibility
	updates[fmt.Sprintf("pages.%d.crm_links", pageIndex)] = append(
		company.Pages[pageIndex].CRMLinks,
		crmLink,
	)

	// Update the document
	update := bson.M{"$set": updates}
	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to add CRM link: %w", err)
	}

	slog.Info("CRM link added successfully",
		"companyID", companyID,
		"pageID", pageID,
		"channels", channels,
		"url", crmLink.URL)

	return nil
}

// UpdateCRMLinkStatusForChannels updates the status of a CRM link for specified channels
func UpdateCRMLinkStatusForChannels(ctx context.Context, companyID, pageID, url string, channels []string, isActive bool) (int64, error) {
	db := GetDatabase()
	collection := db.Collection("companies")

	// Get the company
	filter := bson.M{
		"company_id":    companyID,
		"pages.page_id": pageID,
	}

	var company models.Company
	err := collection.FindOne(ctx, filter).Decode(&company)
	if err != nil {
		return 0, fmt.Errorf("company or page not found: %w", err)
	}

	// Find the page
	var pageConfig *models.FacebookPage
	pageIndex := -1
	for i, page := range company.Pages {
		if page.PageID == pageID {
			pageConfig = &page
			pageIndex = i
			break
		}
	}

	if pageConfig == nil {
		return 0, fmt.Errorf("page not found")
	}

	updatedCount := int64(0)
	updates := bson.M{
		"updated_at": time.Now(),
	}

	// Update for each channel
	for _, channel := range channels {
		if channel == "facebook" && pageConfig.FacebookConfig != nil {
			for i, link := range pageConfig.FacebookConfig.CRMLinks {
				if link.URL == url {
					updates[fmt.Sprintf("pages.%d.facebook_config.crm_links.%d.is_active", pageIndex, i)] = isActive
					updatedCount++
					break
				}
			}
		}

		if channel == "messenger" && pageConfig.MessengerConfig != nil {
			for i, link := range pageConfig.MessengerConfig.CRMLinks {
				if link.URL == url {
					updates[fmt.Sprintf("pages.%d.messenger_config.crm_links.%d.is_active", pageIndex, i)] = isActive
					updatedCount++
					break
				}
			}
		}
	}

	// Also update in legacy CRMLinks
	for i, link := range pageConfig.CRMLinks {
		if link.URL == url {
			updates[fmt.Sprintf("pages.%d.crm_links.%d.is_active", pageIndex, i)] = isActive
			break
		}
	}

	if len(updates) > 1 { // More than just updated_at
		update := bson.M{"$set": updates}
		_, err = collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return 0, fmt.Errorf("failed to update CRM link status: %w", err)
		}
	}

	return updatedCount, nil
}

// DeleteCRMLinkFromChannels removes a CRM link from specified channels
func DeleteCRMLinkFromChannels(ctx context.Context, companyID, pageID, url string, channels []string) (int64, error) {
	db := GetDatabase()
	collection := db.Collection("companies")

	// Get the company
	filter := bson.M{
		"company_id":    companyID,
		"pages.page_id": pageID,
	}

	var company models.Company
	err := collection.FindOne(ctx, filter).Decode(&company)
	if err != nil {
		return 0, fmt.Errorf("company or page not found: %w", err)
	}

	// Find the page
	pageIndex := -1
	for i, page := range company.Pages {
		if page.PageID == pageID {
			pageIndex = i
			break
		}
	}

	if pageIndex == -1 {
		return 0, fmt.Errorf("page not found")
	}

	deletedCount := int64(0)
	updates := bson.M{
		"updated_at": time.Now(),
	}
	pulls := bson.M{}

	// Remove from each channel
	for _, channel := range channels {
		if channel == "facebook" {
			pulls[fmt.Sprintf("pages.%d.facebook_config.crm_links", pageIndex)] = bson.M{"url": url}
			deletedCount++
		}

		if channel == "messenger" {
			pulls[fmt.Sprintf("pages.%d.messenger_config.crm_links", pageIndex)] = bson.M{"url": url}
			deletedCount++
		}
	}

	// Also remove from legacy CRMLinks
	pulls[fmt.Sprintf("pages.%d.crm_links", pageIndex)] = bson.M{"url": url}

	// Execute update with both $set and $pull
	update := bson.M{
		"$set":  updates,
		"$pull": pulls,
	}

	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to delete CRM link: %w", err)
	}

	return deletedCount, nil
}

// FetchAndStoreCRMDataForChannel fetches CRM data and stores it with channel specification
func FetchAndStoreCRMDataForChannel(ctx context.Context, companyID, pageID, channel string, crmLink models.CRMLink) error {
	return FetchAndStoreCRMDataForChannelWithID(ctx, companyID, pageID, channel, crmLink, "")
}

// FetchAndStoreCRMDataForChannelWithID fetches CRM data and stores it with channel specification and CRM ID
func FetchAndStoreCRMDataForChannelWithID(ctx context.Context, companyID, pageID, channel string, crmLink models.CRMLink, crmID string) error {
	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", crmLink.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	if crmLink.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+crmLink.APIKey)
	}
	for k, v := range crmLink.Headers {
		req.Header.Set(k, v)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch CRM data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("CRM API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON to extract text content
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// If not JSON, use raw text
		content := string(body)

		// Store as embeddings with channel
		metadata := map[string]string{
			"crm_url":    crmLink.URL,
			"crm_name":   crmLink.Name,
			"fetch_time": time.Now().Format(time.RFC3339),
		}

		return StoreEmbeddingsWithCRMID(
			ctx,
			companyID,
			pageID,
			channel,
			content,
			"crm",
			metadata,
			crmLink.URL,
			crmID,
			crmLink.IsActive,
		)
	}

	// Convert JSON to readable text
	content := formatJSONAsText(data, "")

	// Store as embeddings with channel
	metadata := map[string]string{
		"crm_url":    crmLink.URL,
		"crm_name":   crmLink.Name,
		"fetch_time": time.Now().Format(time.RFC3339),
	}

	return StoreEmbeddingsWithCRMID(
		ctx,
		companyID,
		pageID,
		channel,
		content,
		"crm",
		metadata,
		crmLink.URL,
		crmID,
		crmLink.IsActive,
	)
}

// SyncVectorDocumentsWithCRMLink syncs vector documents when CRM link status changes
func SyncVectorDocumentsWithCRMLink(ctx context.Context, companyID, pageID, channel, crmURL string, isActive bool) error {
	collection := database.Collection("vector_documents")

	filter := bson.M{
		"company_id": companyID,
		"page_id":    pageID,
		"channels":   channel,
		"crm_url":    crmURL,
	}

	update := bson.M{
		"$set": bson.M{
			"is_active":  isActive,
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to sync vector documents: %w", err)
	}

	slog.Info("Synced vector documents with CRM link",
		"crmURL", crmURL,
		"channel", channel,
		"isActive", isActive,
		"modifiedCount", result.ModifiedCount)

	return nil
}

// DeleteVectorDocumentsByCRMURL deletes vector documents associated with a CRM URL
func DeleteVectorDocumentsByCRMURL(ctx context.Context, companyID, pageID, channel, crmURL string) error {
	collection := database.Collection("vector_documents")

	filter := bson.M{
		"company_id": companyID,
		"page_id":    pageID,
		"channels":   channel,
		"crm_url":    crmURL,
	}

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete vector documents: %w", err)
	}

	slog.Info("Deleted vector documents for CRM URL",
		"crmURL", crmURL,
		"channel", channel,
		"deletedCount", result.DeletedCount)

	return nil
}

// GetVectorDocumentsWithFilter retrieves vector documents with a custom filter
func GetVectorDocumentsWithFilter(ctx context.Context, filter bson.M) ([]VectorDocument, error) {
	collection := database.Collection("vector_documents")

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []VectorDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, fmt.Errorf("failed to decode documents: %w", err)
	}

	// Normalize channels for each document to ensure consistent values
	for i := range documents {
		documents[i].Channels = normalizeChannelsInCRM(documents[i].Channels)
	}

	return documents, nil
}

// normalizeChannelsInCRM ensures channel values are properly formatted (facebook, messenger)
func normalizeChannelsInCRM(channels map[string]bool) map[string]bool {
	if channels == nil {
		return make(map[string]bool)
	}

	normalized := make(map[string]bool)

	for ch, enabled := range channels {
		// Normalize channel names to lowercase
		ch = strings.ToLower(strings.TrimSpace(ch))

		// Fix common misspellings and ensure correct values
		validChannel := ""
		switch ch {
		case "facebook", "fb":
			validChannel = "facebook"
		case "messenger", "messanger", "msg":
			validChannel = "messenger"
		}

		// Add only valid channels with their enabled status
		if validChannel != "" {
			normalized[validChannel] = enabled
		}
	}

	return normalized
}

// GetCRMLinksByPageID retrieves all CRM links from the crm_links collection for a specific page
func GetCRMLinksByPageID(ctx context.Context, pageID string) ([]models.CRMLink, error) {
	db := GetDatabase()
	collection := db.Collection("crm_links")

	filter := bson.M{"page_id": pageID}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find CRM links: %w", err)
	}
	defer cursor.Close(ctx)

	var crmLinks []models.CRMLink
	if err := cursor.All(ctx, &crmLinks); err != nil {
		return nil, fmt.Errorf("failed to decode CRM links: %w", err)
	}

	slog.Info("Retrieved CRM links from collection",
		"pageID", pageID,
		"count", len(crmLinks))

	return crmLinks, nil
}

// GenerateCRMID generates a unique ID for a CRM link based on its URL and page ID
func GenerateCRMID(pageID, crmURL string) string {
	// Use a combination of page ID and URL to create a unique ID
	return fmt.Sprintf("%s_%s", pageID, crmURL)
}

// formatJSONAsText converts JSON data to readable text format
func formatJSONAsText(data interface{}, prefix string) string {
	result := ""

	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if prefix != "" {
				result += fmt.Sprintf("%s.%s: %s\n", prefix, key, formatJSONAsText(value, ""))
			} else {
				result += fmt.Sprintf("%s: %s\n", key, formatJSONAsText(value, ""))
			}
		}
	case []interface{}:
		for i, item := range v {
			itemPrefix := fmt.Sprintf("%s[%d]", prefix, i)
			if prefix == "" {
				itemPrefix = fmt.Sprintf("Item %d", i+1)
			}
			result += formatJSONAsText(item, itemPrefix)
		}
	default:
		result = fmt.Sprintf("%v", v)
	}

	return result
}

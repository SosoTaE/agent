package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"facebook-bot/models"
)

// ProcessorFunc is a function that processes CRM JSON and returns meaningful text
type ProcessorFunc func(jsonData []byte) (string, error)

// PageProcessor contains the processor function and update interval for a page
type PageProcessor struct {
	ProcessorFunc  ProcessorFunc
	UpdateInterval time.Duration
	CRMURLs        []string // URLs to fetch CRM data from
	LastUpdated    time.Time
}

// PageProcessors maps page IDs to their processor configurations
var PageProcessors = map[string]PageProcessor{
	// Archi.ge page
	"102222849213007": {
		ProcessorFunc:  ArchiProcessor,
		UpdateInterval: 6 * time.Hour,
		CRMURLs: []string{
			"https://crm.archi.ge/rest/brokers/product/getBuildings.php",
			"https://crm.archi.ge/rest/brokers/product/getFlats.php?buildingId=3472381",
		},
	},
	// Fake Store API page
	"461998383671026": {
		ProcessorFunc:  FakeStoreProcessor,
		UpdateInterval: 12 * time.Hour, // Update every 12 hours
		CRMURLs: []string{
			"https://fakestoreapi.com/products",
		},
	},
	// Add more pages here as needed
}

// Translation represents a single translated string
type Translation struct {
	LanguageCode string `json:"languageCode"`
	Title        string `json:"title"`
}

// Building represents a single building record from the "data" array
type Building struct {
	ID                 int           `json:"id"`
	Type               string        `json:"type"`
	ComplexID          int           `json:"complexId"`
	ProjectEndDate     string        `json:"projectEndDate"`
	Commission         string        `json:"commission"`
	FullPaymentPercent string        `json:"fullPaymentPercent"`
	Billing            string        `json:"billing"`
	ProjectID          string        `json:"projectId"`
	Responsible        string        `json:"responsible"`
	Phone              interface{}   `json:"phone"`
	Department         string        `json:"department"`
	Address            []Translation `json:"address"`
	City               []Translation `json:"city"`
	District           []Translation `json:"district"`
	Translations       []Translation `json:"translations"`
}

// BuildingsResponse is the top-level structure for buildings API
type BuildingsResponse struct {
	Data []Building `json:"data"`
}

// findTranslation searches for a title in a specific language, defaulting to English
func findTranslation(translations []Translation, langCode string) string {
	defaultTitle := ""
	for _, t := range translations {
		if t.LanguageCode == langCode {
			return t.Title
		}
		if t.LanguageCode == "en" {
			defaultTitle = t.Title
		}
	}
	// If the desired language isn't found, return the English version or empty if not found
	return defaultTitle
}

// ArchiProcessor processes Archi.ge CRM JSON data
func ArchiProcessor(jsonData []byte) (string, error) {
	var result strings.Builder

	// Try to parse as buildings data with proper structure
	var buildingsResp BuildingsResponse
	if err := json.Unmarshal(jsonData, &buildingsResp); err == nil && len(buildingsResp.Data) > 0 {
		result.WriteString("უძრავი ქონების ინფორმაცია:\n\n")

		for _, building := range buildingsResp.Data {
			// Get the Georgian translation for key fields, falling back to English
			buildingName := findTranslation(building.Translations, "ka")
			address := findTranslation(building.Address, "ka")
			city := findTranslation(building.City, "ka")
			district := findTranslation(building.District, "ka")

			// Start building the description for the current building
			result.WriteString(fmt.Sprintf("შენობის დასახელება: %s, ტიპი: %s.\n", buildingName, building.Type))
			result.WriteString(fmt.Sprintf("მდებარეობა: %s, %s, %s.\n", city, district, address))
			result.WriteString(fmt.Sprintf("შიდა ID: %d, კომპლექსის ID: %d, პროექტის ID: %s.\n",
				building.ID, building.ComplexID, building.ProjectID))

			// Add optional fields only if they contain data
			if building.Responsible != "" {
				result.WriteString(fmt.Sprintf("პასუხისმგებელი პირი: %s.\n", building.Responsible))
			}
			if building.Department != "" {
				result.WriteString(fmt.Sprintf("დეპარტამენტი: %s.\n", building.Department))
			}
			if building.Billing != "" {
				result.WriteString(fmt.Sprintf("ბილინგის კოდი: %s.\n", building.Billing))
			}
			if building.FullPaymentPercent != "" {
				result.WriteString(fmt.Sprintf("სრული გადახდის პროცენტი: %s%%.\n", building.FullPaymentPercent))
			}
			if building.ProjectEndDate != "" {
				result.WriteString(fmt.Sprintf("პროექტის დასრულების თარიღი: %s.\n", building.ProjectEndDate))
			}
			if building.Commission != "" {
				result.WriteString(fmt.Sprintf("საკომისიო: %s.\n", building.Commission))
			}

			// Add English translations as well for better search
			result.WriteString("\n")
			buildingNameEn := findTranslation(building.Translations, "en")
			addressEn := findTranslation(building.Address, "en")
			cityEn := findTranslation(building.City, "en")
			districtEn := findTranslation(building.District, "en")

			if buildingNameEn != "" {
				result.WriteString(fmt.Sprintf("Building Name: %s, Type: %s.\n", buildingNameEn, building.Type))
			}
			if cityEn != "" || districtEn != "" || addressEn != "" {
				result.WriteString(fmt.Sprintf("Location: %s, %s, %s.\n", cityEn, districtEn, addressEn))
			}

			// Add a separator for clarity between records
			result.WriteString("---\n\n")
		}

		return result.String(), nil
	}

	// Try to parse as flats data (for the getFlats API)
	var flatsData map[string]interface{}
	if err := json.Unmarshal(jsonData, &flatsData); err == nil {
		if flatsList, ok := flatsData["flats"].([]interface{}); ok {
			result.WriteString("ხელმისაწვდომი ბინები / Available Apartments:\n\n")

			for _, flatInterface := range flatsList {
				if flat, ok := flatInterface.(map[string]interface{}); ok {
					// Extract flat information
					if flatNumber, ok := flat["flat_number"]; ok {
						result.WriteString(fmt.Sprintf("ბინის ნომერი / Apartment Number: %v\n", flatNumber))
					}

					if floor, ok := flat["floor"]; ok {
						result.WriteString(fmt.Sprintf("სართული / Floor: %v\n", floor))
					}

					if rooms, ok := flat["rooms"]; ok {
						result.WriteString(fmt.Sprintf("ოთახები / Rooms: %v\n", rooms))
					}

					if area, ok := flat["area"]; ok {
						result.WriteString(fmt.Sprintf("ფართობი / Area: %v კვ.მ / sq.m\n", area))
					}

					if price, ok := flat["price"]; ok {
						result.WriteString(fmt.Sprintf("ფასი / Price: %v\n", price))
					}

					if pricePerSqm, ok := flat["price_per_sqm"]; ok {
						result.WriteString(fmt.Sprintf("ფასი კვ.მ-ზე / Price per sq.m: %v\n", pricePerSqm))
					}

					if status, ok := flat["status"].(string); ok {
						result.WriteString(fmt.Sprintf("სტატუსი / Status: %s\n", status))
					}

					if view, ok := flat["view"].(string); ok && view != "" {
						result.WriteString(fmt.Sprintf("ხედი / View: %s\n", view))
					}

					if balcony, ok := flat["balcony"].(bool); ok && balcony {
						result.WriteString("მახასიათებლები / Features: აივანი / Balcony\n")
					}

					result.WriteString("\n---\n\n")
				}
			}

			return result.String(), nil
		}
	}

	// If we can't parse the specific format, try generic parsing
	var genericData interface{}
	if err := json.Unmarshal(jsonData, &genericData); err == nil {
		return fmt.Sprintf("Property Data:\n%s", formatGenericJSON(genericData, 0)), nil
	}

	return "", fmt.Errorf("unable to parse CRM data")
}

// Product represents a product from Fake Store API
type Product struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Image       string  `json:"image"`
	Rating      struct {
		Rate  float64 `json:"rate"`
		Count int     `json:"count"`
	} `json:"rating"`
}

// FakeStoreProcessor processes Fake Store API JSON data
func FakeStoreProcessor(jsonData []byte) (string, error) {
	var result strings.Builder

	// Try to parse as array of products
	var products []Product
	if err := json.Unmarshal(jsonData, &products); err != nil {
		// If not array, try single product
		var product Product
		if err := json.Unmarshal(jsonData, &product); err != nil {
			// If still fails, try generic parsing
			var genericData interface{}
			if err := json.Unmarshal(jsonData, &genericData); err == nil {
				return fmt.Sprintf("Store Data:\n%s", formatGenericJSON(genericData, 0)), nil
			}
			return "", fmt.Errorf("unable to parse store data: %w", err)
		}
		// Single product
		products = []Product{product}
	}

	result.WriteString("Online Store Products Catalog:\n\n")
	result.WriteString("Available Products / ხელმისაწვდომი პროდუქტები:\n\n")

	// Group products by category
	categoryMap := make(map[string][]Product)
	for _, product := range products {
		categoryMap[product.Category] = append(categoryMap[product.Category], product)
	}

	// Process each category
	for category, categoryProducts := range categoryMap {
		result.WriteString(fmt.Sprintf("=== %s ===\n\n", strings.Title(category)))

		for _, product := range categoryProducts {
			// Product name and ID
			result.WriteString(fmt.Sprintf("Product: %s\n", product.Title))
			result.WriteString(fmt.Sprintf("Product ID: %d\n", product.ID))

			// Price
			result.WriteString(fmt.Sprintf("Price: $%.2f\n", product.Price))

			// Category
			result.WriteString(fmt.Sprintf("Category: %s\n", product.Category))

			// Description
			if product.Description != "" {
				// Truncate very long descriptions
				desc := product.Description
				if len(desc) > 300 {
					desc = desc[:297] + "..."
				}
				result.WriteString(fmt.Sprintf("Description: %s\n", desc))
			}

			// Rating
			if product.Rating.Rate > 0 {
				result.WriteString(fmt.Sprintf("Rating: %.1f/5.0 (%d reviews)\n",
					product.Rating.Rate, product.Rating.Count))
			}

			// Image URL
			if product.Image != "" {
				result.WriteString(fmt.Sprintf("Image: %s\n", product.Image))
			}

			// Add multilingual product info
			result.WriteString("\n")
			result.WriteString(fmt.Sprintf("პროდუქტი: %s\n", product.Title))
			result.WriteString(fmt.Sprintf("ფასი: $%.2f\n", product.Price))
			result.WriteString(fmt.Sprintf("კატეგორია: %s\n", product.Category))
			if product.Rating.Rate > 0 {
				result.WriteString(fmt.Sprintf("რეიტინგი: %.1f/5.0 (%d მიმოხილვა)\n",
					product.Rating.Rate, product.Rating.Count))
			}

			result.WriteString("\n---\n\n")
		}
	}

	// Add summary
	result.WriteString(fmt.Sprintf("\nTotal Products Available: %d\n", len(products)))
	result.WriteString(fmt.Sprintf("სულ ხელმისაწვდომია: %d პროდუქტი\n", len(products)))

	// Calculate average price
	if len(products) > 0 {
		totalPrice := 0.0
		for _, p := range products {
			totalPrice += p.Price
		}
		avgPrice := totalPrice / float64(len(products))
		result.WriteString(fmt.Sprintf("Average Price: $%.2f\n", avgPrice))
		result.WriteString(fmt.Sprintf("საშუალო ფასი: $%.2f\n", avgPrice))
	}

	return result.String(), nil
}

// formatGenericJSON formats generic JSON data into readable text
func formatGenericJSON(data interface{}, indent int) string {
	var result strings.Builder
	indentStr := strings.Repeat("  ", indent)

	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			// Skip technical fields
			if key == "_id" || key == "created_at" || key == "updated_at" || strings.HasPrefix(key, "__") {
				continue
			}

			// Make key more readable
			readableKey := strings.ReplaceAll(key, "_", " ")
			readableKey = strings.Title(readableKey)

			result.WriteString(fmt.Sprintf("%s%s: ", indentStr, readableKey))

			if nested, ok := value.(map[string]interface{}); ok {
				result.WriteString("\n")
				result.WriteString(formatGenericJSON(nested, indent+1))
			} else if arr, ok := value.([]interface{}); ok {
				result.WriteString("\n")
				for i, item := range arr {
					result.WriteString(fmt.Sprintf("%s  [%d] ", indentStr, i+1))
					result.WriteString(formatGenericJSON(item, indent+2))
					result.WriteString("\n")
				}
			} else {
				result.WriteString(fmt.Sprintf("%v\n", value))
			}
		}
	case []interface{}:
		for i, item := range v {
			result.WriteString(fmt.Sprintf("%s[%d] ", indentStr, i+1))
			result.WriteString(formatGenericJSON(item, indent+1))
			result.WriteString("\n")
		}
	default:
		result.WriteString(fmt.Sprintf("%v", v))
	}

	return result.String()
}

// FetchAndProcessCRMData fetches CRM data and processes it for a specific page
func FetchAndProcessCRMData(ctx context.Context, pageID string, companyID string) error {
	processor, exists := PageProcessors[pageID]
	if !exists {
		slog.Info("No processor found for page", "pageID", pageID)
		return nil
	}

	// Get company configuration to check CRM links
	company, err := GetCompanyByID(ctx, companyID)
	if err != nil {
		slog.Error("Failed to get company configuration", "error", err, "companyID", companyID)
		return fmt.Errorf("failed to get company configuration: %w", err)
	}

	// Find the page configuration
	var pageConfig *models.FacebookPage
	for _, page := range company.Pages {
		if page.PageID == pageID {
			pageConfig = &page
			break
		}
	}

	if pageConfig == nil {
		slog.Warn("Page not found in company configuration", "pageID", pageID, "companyID", companyID)
		return fmt.Errorf("page %s not found in company configuration", pageID)
	}

	// Filter only active CRM links
	var activeCRMURLs []string
	for _, crmLink := range pageConfig.CRMLinks {
		if crmLink.IsActive {
			// Check if this URL is in the processor's URLs
			for _, processorURL := range processor.CRMURLs {
				if crmLink.URL == processorURL {
					activeCRMURLs = append(activeCRMURLs, crmLink.URL)
					break
				}
			}
		}
	}

	if len(activeCRMURLs) == 0 {
		slog.Info("No active CRM links found for page",
			"pageID", pageID,
			"companyID", companyID,
			"totalCRMLinks", len(pageConfig.CRMLinks),
		)
		return nil
	}

	slog.Info("Processing CRM data for page",
		"pageID", pageID,
		"companyID", companyID,
		"activeCRMURLs", len(activeCRMURLs),
		"totalCRMURLs", len(processor.CRMURLs),
	)

	var allProcessedText strings.Builder
	allProcessedText.WriteString(fmt.Sprintf("Real Estate Information (Updated: %s)\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Fetch and process only active CRM URLs
	for _, url := range activeCRMURLs {
		slog.Info("Fetching CRM data from active link", "url", url)

		// Fetch the CRM data
		jsonData, err := FetchCRMData(ctx, url, "")
		if err != nil {
			slog.Error("Failed to fetch CRM data",
				"error", err,
				"url", url,
				"pageID", pageID,
			)
			continue
		}

		// Process the JSON data to meaningful text
		processedText, err := processor.ProcessorFunc(jsonData)
		if err != nil {
			slog.Error("Failed to process CRM data",
				"error", err,
				"url", url,
				"pageID", pageID,
			)
			continue
		}

		allProcessedText.WriteString(processedText)
		allProcessedText.WriteString("\n\n")

		slog.Info("Successfully processed CRM data",
			"url", url,
			"textLength", len(processedText),
		)
	}

	finalText := allProcessedText.String()
	if finalText == "" {
		slog.Warn("No CRM data was processed", "pageID", pageID)
		return nil
	}

	// Store one document per CRM URL
	successCount := 0
	for _, url := range activeCRMURLs {
		// Find if this CRM link is active
		isActive := false
		for _, crmLink := range pageConfig.CRMLinks {
			if crmLink.URL == url && crmLink.IsActive {
				isActive = true
				break
			}
		}

		// Store the processed text as embeddings in MongoDB
		metadata := map[string]string{
			"source_type": "crm",
			"page_id":     pageID,
			"crm_url":     url,
			"update_time": time.Now().Format(time.RFC3339),
		}

		// Store the entire content as one document per CRM URL
		err := StoreEmbeddingsWithOptions(ctx, companyID, pageID, finalText, "crm", metadata, url, isActive)
		if err != nil {
			slog.Error("Failed to store embeddings",
				"error", err,
				"pageID", pageID,
				"crmURL", url,
			)
			continue
		}
		successCount++
	}

	if successCount == 0 {
		return fmt.Errorf("failed to store any CRM embeddings")
	}

	// Update the last updated time
	processor.LastUpdated = time.Now()
	PageProcessors[pageID] = processor

	slog.Info("Successfully stored CRM embeddings",
		"pageID", pageID,
		"companyID", companyID,
		"documentsStored", successCount,
		"totalTextLength", len(finalText),
	)

	return nil
}

// splitTextIntoChunks splits text into chunks of specified max size
func splitTextIntoChunks(text string, maxSize int) []string {
	if len(text) <= maxSize {
		return []string{text}
	}

	var chunks []string
	lines := strings.Split(text, "\n")
	currentChunk := ""

	for _, line := range lines {
		// If adding this line would exceed max size, start new chunk
		if len(currentChunk)+len(line)+1 > maxSize && currentChunk != "" {
			chunks = append(chunks, currentChunk)
			currentChunk = line
		} else {
			if currentChunk != "" {
				currentChunk += "\n"
			}
			currentChunk += line
		}
	}

	// Add the last chunk
	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

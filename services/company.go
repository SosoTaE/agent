package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"facebook-bot/models"
)

// companyCache stores company configurations in memory for faster access
var companyCache = make(map[string]*models.Company)
var cacheExpiry = make(map[string]time.Time)

// GetCompanyByPageID retrieves company configuration by Facebook page ID
func GetCompanyByPageID(ctx context.Context, pageID string) (*models.Company, error) {
	// Check cache first
	if company, found := getFromCache(pageID); found {
		return company, nil
	}

	collection := database.Collection("companies")

	// Find company document that contains this page ID in the pages array
	filter := bson.M{
		"pages.page_id": pageID,
	}

	var company models.Company
	err := collection.FindOne(ctx, filter).Decode(&company)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			slog.Warn("No company found for page", "pageID", pageID)
			return nil, fmt.Errorf("no company configuration found for page %s", pageID)
		}
		return nil, err
	}

	// Store in cache
	storeInCache(pageID, &company)

	return &company, nil
}

// GetPageConfig retrieves page configuration from company pages array
func GetPageConfig(company *models.Company, pageID string) (*models.FacebookPage, error) {
	// Find the page in the pages array
	for _, page := range company.Pages {
		if page.PageID == pageID && page.IsActive {
			return &page, nil
		}
	}
	return nil, fmt.Errorf("page %s not found or inactive in company configuration", pageID)
}

// GetCompanyByID retrieves the company document with the given company ID
func GetCompanyByID(ctx context.Context, companyID string) (*models.Company, error) {
	collection := database.Collection("companies")

	filter := bson.M{
		"company_id": companyID,
	}

	var company models.Company
	err := collection.FindOne(ctx, filter).Decode(&company)
	if err != nil {
		return nil, err
	}

	return &company, nil
}

// GetAllCompanies retrieves all company documents
func GetAllCompanies(ctx context.Context) ([]models.Company, error) {
	collection := database.Collection("companies")

	filter := bson.M{}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var companies []models.Company
	if err = cursor.All(ctx, &companies); err != nil {
		return nil, err
	}

	return companies, nil
}

// GetAllActiveCompanies is an alias for GetAllCompanies for clarity
func GetAllActiveCompanies(ctx context.Context) ([]models.Company, error) {
	return GetAllCompanies(ctx)
}

// GetCompaniesByCompanyID retrieves company by ID (single document with pages array)
func GetCompaniesByCompanyID(ctx context.Context, companyID string) ([]models.Company, error) {
	company, err := GetCompanyByID(ctx, companyID)
	if err != nil {
		return nil, err
	}
	return []models.Company{*company}, nil
}

// GetPagesByCompanyID retrieves all pages for a specific company ID
func GetPagesByCompanyID(ctx context.Context, companyID string) ([]models.FacebookPage, error) {
	company, err := GetCompanyByID(ctx, companyID)
	if err != nil {
		return nil, err
	}

	return company.Pages, nil
}

// GenerateCompanyID generates a unique company ID in the format: company_<name>_<timestamp>
func GenerateCompanyID(companyName string) string {
	// Convert company name to lowercase and replace spaces with underscores
	sanitizedName := strings.ToLower(companyName)
	sanitizedName = strings.ReplaceAll(sanitizedName, " ", "_")
	// Remove any special characters, keeping only alphanumeric and underscores
	var cleanName strings.Builder
	for _, r := range sanitizedName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			cleanName.WriteRune(r)
		}
	}

	// Generate timestamp (Unix timestamp in seconds)
	timestamp := time.Now().Unix()

	// Create the company ID
	return fmt.Sprintf("company_%s_%d", cleanName.String(), timestamp)
}

// CreateCompany creates a new company configuration
func CreateCompany(ctx context.Context, company *models.Company) error {
	collection := database.Collection("companies")

	// Generate company ID if not provided
	if company.CompanyID == "" {
		company.CompanyID = GenerateCompanyID(company.CompanyName)
	}

	company.CreatedAt = time.Now()
	company.UpdatedAt = time.Now()

	// Set defaults for each page if not provided
	for i := range company.Pages {
		if company.Pages[i].ClaudeModel == "" {
			company.Pages[i].ClaudeModel = "claude-3-haiku-20240307"
		}
		if company.Pages[i].MaxTokens == 0 {
			company.Pages[i].MaxTokens = 1024
		}
	}

	_, err := collection.InsertOne(ctx, company)
	return err
}

// UpdateCompany updates an existing company configuration
func UpdateCompany(ctx context.Context, companyID string, update interface{}) error {
	collection := database.Collection("companies")

	filter := bson.M{"company_id": companyID}

	// Handle both $set and $push operations
	var updateDoc bson.M
	if m, ok := update.(bson.M); ok {
		updateDoc = m
	} else {
		updateDoc = bson.M{"$set": update}
	}

	_, err := collection.UpdateOne(ctx, filter, updateDoc)

	// Clear cache for all pages of this company
	clearCompanyCache(companyID)

	return err
}

// Cache helper functions
func getFromCache(pageID string) (*models.Company, bool) {
	if expiry, exists := cacheExpiry[pageID]; exists {
		if time.Now().Before(expiry) {
			if company, found := companyCache[pageID]; found {
				return company, true
			}
		}
		// Cache expired, remove it
		delete(companyCache, pageID)
		delete(cacheExpiry, pageID)
	}
	return nil, false
}

func storeInCache(pageID string, company *models.Company) {
	companyCache[pageID] = company
	cacheExpiry[pageID] = time.Now().Add(5 * time.Minute) // Cache for 5 minutes
}

func clearCompanyCache(companyID string) {
	// Clear cache entries for all pages of this company
	for pageID, company := range companyCache {
		if company.CompanyID == companyID {
			delete(companyCache, pageID)
			delete(cacheExpiry, pageID)
		}
	}
}

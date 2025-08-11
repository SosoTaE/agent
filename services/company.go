package services

import (
	"context"
	"fmt"
	"log/slog"
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

	// Find company that has this page ID
	filter := bson.M{
		"pages.page_id": pageID,
		"is_active":     true,
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

// GetPageConfig retrieves specific page configuration from company
func GetPageConfig(company *models.Company, pageID string) (*models.FacebookPage, error) {
	for _, page := range company.Pages {
		if page.PageID == pageID && page.IsActive {
			return &page, nil
		}
	}
	return nil, fmt.Errorf("page %s not found or inactive in company configuration", pageID)
}

// GetCompanyByID retrieves company configuration by company ID
func GetCompanyByID(ctx context.Context, companyID string) (*models.Company, error) {
	collection := database.Collection("companies")

	filter := bson.M{
		"company_id": companyID,
		"is_active":  true,
	}

	var company models.Company
	err := collection.FindOne(ctx, filter).Decode(&company)
	if err != nil {
		return nil, err
	}

	return &company, nil
}

// GetAllCompanies retrieves all active companies
func GetAllCompanies(ctx context.Context) ([]models.Company, error) {
	collection := database.Collection("companies")

	filter := bson.M{"is_active": true}

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

// CreateCompany creates a new company configuration
func CreateCompany(ctx context.Context, company *models.Company) error {
	collection := database.Collection("companies")

	company.CreatedAt = time.Now()
	company.UpdatedAt = time.Now()

	// Set defaults if not provided
	if company.ClaudeModel == "" {
		company.ClaudeModel = "claude-3-haiku-20240307"
	}
	if company.MaxTokens == 0 {
		company.MaxTokens = 1024
	}

	_, err := collection.InsertOne(ctx, company)
	return err
}

// UpdateCompany updates an existing company configuration
func UpdateCompany(ctx context.Context, companyID string, update bson.M) error {
	collection := database.Collection("companies")

	filter := bson.M{"company_id": companyID}
	update["updated_at"] = time.Now()

	_, err := collection.UpdateOne(ctx, filter, bson.M{"$set": update})

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

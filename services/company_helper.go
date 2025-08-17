package services

import (
	"context"
	"fmt"
)

// CompanyPageFilter provides filtering options for company pages
type CompanyPageFilter struct {
	CompanyID string
	PageID    string
	PageIDs   []string
}

// GetPageIDsForFilter returns the page IDs based on the filter criteria
func GetPageIDsForFilter(ctx context.Context, filter CompanyPageFilter) ([]string, error) {
	// If specific page IDs are provided, use them
	if len(filter.PageIDs) > 0 {
		return filter.PageIDs, nil
	}

	// If a specific page ID is provided, return it
	if filter.PageID != "" {
		return []string{filter.PageID}, nil
	}

	// If company ID is provided, get all pages for the company
	if filter.CompanyID != "" {
		return GetCompanyPageIDs(ctx, filter.CompanyID)
	}

	// No filter specified
	return nil, nil
}

// ValidatePageAccess checks if a user has access to a specific page
func ValidatePageAccess(ctx context.Context, pageID string, companyID string) error {
	if companyID == "" {
		return nil // No validation needed if no company context
	}

	pageIDs, err := GetCompanyPageIDs(ctx, companyID)
	if err != nil {
		return fmt.Errorf("failed to get company pages: %w", err)
	}

	for _, id := range pageIDs {
		if id == pageID {
			return nil
		}
	}

	return fmt.Errorf("page does not belong to company")
}

// ValidatePageOwnership checks if a page belongs to a company and returns the page name
func ValidatePageOwnership(ctx context.Context, pageID string, companyID string) (string, error) {
	pages, err := GetPagesByCompanyID(ctx, companyID)
	if err != nil {
		return "", fmt.Errorf("failed to get company pages: %w", err)
	}

	for _, page := range pages {
		if page.PageID == pageID {
			return page.PageName, nil
		}
	}

	return "", fmt.Errorf("page %s does not belong to company %s", pageID, companyID)
}

// ValidateMultiplePageAccess checks if a user has access to multiple pages
func ValidateMultiplePageAccess(ctx context.Context, requestedPageIDs []string, companyID string) ([]string, error) {
	if companyID == "" {
		return requestedPageIDs, nil // No validation needed
	}

	companyPageIDs, err := GetCompanyPageIDs(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company pages: %w", err)
	}

	// Create a map for faster lookup
	companyPageMap := make(map[string]bool)
	for _, id := range companyPageIDs {
		companyPageMap[id] = true
	}

	// Filter requested pages to only those the company has access to
	validPageIDs := make([]string, 0)
	for _, pageID := range requestedPageIDs {
		if companyPageMap[pageID] {
			validPageIDs = append(validPageIDs, pageID)
		}
	}

	if len(validPageIDs) == 0 {
		return nil, fmt.Errorf("none of the requested pages belong to the company")
	}

	return validPageIDs, nil
}

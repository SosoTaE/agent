package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// InitializeCRMData processes all CRM data for all configured pages before server starts
func InitializeCRMData(ctx context.Context) error {
	slog.Info("Starting CRM data initialization", "totalPages", len(PageProcessors))

	// Get all companies to map pages to companies
	companies, err := GetAllActiveCompanies(ctx)
	if err != nil {
		return fmt.Errorf("failed to get companies: %w", err)
	}

	// Create a map of pageID to companyID from companies with pages array
	pageToCompany := make(map[string]string)
	for _, company := range companies {
		for _, page := range company.Pages {
			if page.IsActive {
				pageToCompany[page.PageID] = company.CompanyID
			}
		}
	}

	// Process each page's CRM data in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, len(PageProcessors))

	for pageID := range PageProcessors {
		companyID, exists := pageToCompany[pageID]
		if !exists {
			slog.Warn("No company found for page", "pageID", pageID)
			continue
		}

		wg.Add(1)
		go func(pID, cID string) {
			defer wg.Done()

			slog.Info("Initializing CRM data", "pageID", pID, "companyID", cID)

			if err := FetchAndProcessCRMData(ctx, pID, cID); err != nil {
				errChan <- fmt.Errorf("failed to process page %s: %w", pID, err)
				return
			}

			slog.Info("Successfully initialized CRM data", "pageID", pID)
		}(pageID, companyID)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
		slog.Error("CRM initialization error", "error", err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("CRM initialization completed with %d errors", len(errors))
	}

	slog.Info("CRM data initialization completed successfully")
	return nil
}

// StartCRMUpdateScheduler starts the background scheduler for CRM updates
func StartCRMUpdateScheduler(ctx context.Context) {
	slog.Info("Starting CRM update scheduler")

	// Get all companies to map pages to companies
	companies, err := GetAllActiveCompanies(ctx)
	if err != nil {
		slog.Error("Failed to get companies for scheduler", "error", err)
		return
	}

	// Create a map of pageID to companyID from companies with pages array
	pageToCompany := make(map[string]string)
	for _, company := range companies {
		for _, page := range company.Pages {
			if page.IsActive {
				pageToCompany[page.PageID] = company.CompanyID
			}
		}
	}

	// Start a goroutine for each page with its own update interval
	for pageID, processor := range PageProcessors {
		companyID, exists := pageToCompany[pageID]
		if !exists {
			slog.Warn("No company found for page in scheduler", "pageID", pageID)
			continue
		}

		go func(pID, cID string, proc PageProcessor) {
			slog.Info("Starting update scheduler for page",
				"pageID", pID,
				"interval", proc.UpdateInterval.String(),
			)

			ticker := time.NewTicker(proc.UpdateInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					slog.Info("Running scheduled CRM update", "pageID", pID)

					updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
					if err := FetchAndProcessCRMData(updateCtx, pID, cID); err != nil {
						slog.Error("Failed scheduled CRM update",
							"error", err,
							"pageID", pID,
						)
					} else {
						slog.Info("Completed scheduled CRM update", "pageID", pID)
					}
					cancel()

				case <-ctx.Done():
					slog.Info("Stopping CRM updater for page", "pageID", pID)
					return
				}
			}
		}(pageID, companyID, processor)
	}

	slog.Info("CRM update scheduler started for all pages")
}

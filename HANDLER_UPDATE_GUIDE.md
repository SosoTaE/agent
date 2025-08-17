# Handler Update Guide for New Company Structure

## Overview
The company structure has changed from having a `Pages` array to having one document per page. Each company document now represents a single page with all the company settings.

## Key Changes

### 1. Getting All Pages for a Company
**OLD:**
```go
company, err := services.GetCompanyByID(ctx, companyID)
for _, page := range company.Pages {
    // do something with page
}
```

**NEW:**
```go
// Option 1: Get pages directly
pages, err := services.GetPagesByCompanyID(ctx, companyID)
for _, page := range pages {
    // do something with page
}

// Option 2: Get all company documents
companies, err := services.GetCompaniesByCompanyID(ctx, companyID)
for _, company := range companies {
    // Each company represents a page
    pageID := company.PageID
    pageName := company.PageName
}
```

### 2. Checking if a Page Belongs to a Company
**OLD:**
```go
company, err := services.GetCompanyByID(ctx, companyID)
pageFound := false
for _, p := range company.Pages {
    if p.PageID == pageID {
        pageFound = true
        pageName = p.PageName
        break
    }
}
```

**NEW:**
```go
// Use the helper function
pageName, err := services.ValidatePageOwnership(ctx, pageID, companyID)
if err != nil {
    // Page doesn't belong to company
}
```

### 3. Getting Page Count
**OLD:**
```go
company, err := services.GetCompanyByID(ctx, companyID)
pageCount := len(company.Pages)
```

**NEW:**
```go
pages, err := services.GetPagesByCompanyID(ctx, companyID)
pageCount := len(pages)
```

## Files That Need Updates

### customer_handler.go
- Lines with `company.Pages` need to use `services.GetPagesByCompanyID()`

### dashboard_handler.go  
- Lines with `company.Pages` need to use `services.GetPagesByCompanyID()`
- Dashboard stats need to aggregate across all company documents

### message_handler.go
- Page validation needs to use `services.ValidatePageOwnership()`

### websocket_handler.go
- Page access validation needs to use the new helper functions

### rag_debug_handler.go
- Access to first page needs to get pages list first

## Migration Steps

1. **Run the migration script first:**
   ```bash
   go run migrate_companies.go
   ```

2. **Update handlers one by one:**
   - Start with critical handlers (message, webhook)
   - Then update admin/dashboard handlers
   - Finally update utility handlers

3. **Test each handler after update:**
   - Verify page access control works
   - Check that all pages are visible
   - Ensure adding new pages works

## Helper Functions Available

```go
// Get all pages for a company
services.GetPagesByCompanyID(ctx, companyID) ([]models.FacebookPage, error)

// Get all company documents for a company ID
services.GetCompaniesByCompanyID(ctx, companyID) ([]models.Company, error)

// Validate if a page belongs to a company
services.ValidatePageOwnership(ctx, pageID, companyID) (pageName string, error)

// Get all page IDs for a company
services.GetCompanyPageIDs(ctx, companyID) ([]string, error)
```

## Benefits of New Structure

1. **Simpler queries** - Direct lookup by page_id
2. **Better scalability** - Each page is independent
3. **Easier management** - Add/remove pages without array manipulation
4. **Cleaner data** - Page-specific settings are isolated

## Rollback Plan

If needed, the original data is backed up in `companies_backup` collection and can be restored.
# Handler Updates Required

## Summary
The handlers need to be updated to work with the new company structure where each document represents a single page instead of having a pages array.

## Key Changes Required

### 1. Replace loops over company.Pages
OLD:
```go
for _, p := range company.Pages {
    if p.PageID == pageID {
        // do something
    }
}
```

NEW:
```go
// For single page checks
if company.PageID == pageID {
    // do something
}

// For multiple pages
pages, err := services.GetPagesByCompanyID(ctx, companyID)
for _, p := range pages {
    if p.PageID == pageID {
        // do something
    }
}
```

### 2. Admin Handler Page Addition
The admin handler's AddPage function needs to create a new company document instead of adding to an array.

### 3. Dashboard Handler
Dashboard needs to fetch all company documents for a company ID to show all pages.

## Files to Update:
1. handlers/admin_handler.go
2. handlers/customer_handler.go
3. handlers/dashboard_handler.go
4. handlers/message_handler.go
5. handlers/websocket_handler.go
6. handlers/rag_debug_handler.go

## Migration Command
After updating the code:
```bash
go run migrate_companies.go
```

This will:
1. Backup existing companies collection to companies_backup
2. Split each company with multiple pages into separate documents
3. Preserve all settings for each page
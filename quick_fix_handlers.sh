#!/bin/bash

# Quick fix script for updating handlers to new company structure
# This handles the most common patterns but manual review is still needed

echo "Creating backup of handlers..."
cp -r handlers handlers.backup

echo "Updating customer_handler.go..."
# Fix the common pattern in customer handler
sed -i 's/for _, p := range company\.Pages {/pages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)\n\tfor _, p := range pages {/g' handlers/customer_handler.go

echo "Updating dashboard_handler.go..."
# Fix page count references
sed -i 's/len(company\.Pages)/func() int { pages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID); return len(pages) }()/g' handlers/dashboard_handler.go

# Fix page iterations
sed -i 's/for _, page := range company\.Pages {/pages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)\n\tfor _, page := range pages {/g' handlers/dashboard_handler.go

echo "Updating message_handler.go..."
# Update page validation patterns
sed -i 's/for _, p := range company\.Pages {/pages, _ := services.GetPagesByCompanyID(ctx, companyID.(string))\n\tfor _, p := range pages {/g' handlers/message_handler.go

echo "Updating websocket_handler.go..."
sed -i 's/for _, page := range company\.Pages {/pages, _ := services.GetPagesByCompanyID(ctx, company.CompanyID)\n\tfor _, page := range pages {/g' handlers/websocket_handler.go

echo "Updating rag_debug_handler.go..."
# Fix first page access
sed -i 's/company\.Pages\[0\]\.PageID/company.PageID/g' handlers/rag_debug_handler.go
sed -i 's/len(company\.Pages) > 0/company.PageID != ""/g' handlers/rag_debug_handler.go

echo "Done! Please review the changes and test thoroughly."
echo "Backup created in handlers.backup/"
echo ""
echo "Note: This script handles common patterns but manual review is still needed for:"
echo "  - Error handling for the new GetPagesByCompanyID calls"
echo "  - Complex logic that depends on the Pages array"
echo "  - Any custom page validation logic"
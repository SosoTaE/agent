#!/bin/bash

# Configure Voyage for your company
# Usage: ./configure_voyage.sh <voyage_api_key> [model]

COMPANY_ID="company_economist_georgia_1754827518"
API_KEY="${1}"
MODEL="${2:-voyage-2}"  # Default to voyage-2 if not specified

if [ -z "$API_KEY" ]; then
    echo "Usage: $0 <voyage_api_key> [model]"
    echo "Available models: voyage-2, voyage-large-2, voyage-code-2"
    echo ""
    echo "To get a Voyage API key:"
    echo "1. Go to https://www.voyageai.com/"
    echo "2. Sign up for an account"
    echo "3. Get your API key from the dashboard"
    exit 1
fi

echo "Configuring Voyage for company: $COMPANY_ID"
echo "Model: $MODEL"

# Configure Voyage
curl -X PUT "http://localhost:8080/admin/companies/${COMPANY_ID}/voyage" \
  -H "Content-Type: application/json" \
  -d "{
    \"api_key\": \"${API_KEY}\",
    \"model\": \"${MODEL}\"
  }"

echo ""
echo "Testing Voyage configuration..."

# Test the configuration
curl -X POST "http://localhost:8080/admin/companies/${COMPANY_ID}/voyage/test" \
  -H "Content-Type: application/json" \
  -d '{"text": "Testing Voyage embeddings integration"}' | jq .

echo ""
echo "If successful, now reindexing all documents with Voyage embeddings..."

# Reindex documents
curl -X POST "http://localhost:8080/admin/companies/${COMPANY_ID}/reindex"

echo ""
echo "Voyage configuration complete!"
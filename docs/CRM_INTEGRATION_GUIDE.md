# CRM Integration Guide

## Overview

The CRM Integration feature allows the Facebook Bot Management System to connect with external Customer Relationship Management systems, fetch customer data, and use that information to provide personalized responses through the bot. The system supports multiple CRM endpoints per page with channel-specific configurations.

## Key Features

- **Multi-Channel Support**: Different CRM configurations for Facebook comments and Messenger
- **Dynamic Data Fetching**: Automatic synchronization of CRM data
- **API Authentication**: Support for API keys and custom headers
- **Active/Inactive Toggle**: Enable or disable CRM links without deletion
- **Scheduled Updates**: Background scheduler for periodic data refresh
- **Vector Storage**: CRM data is converted to embeddings for semantic search

## Architecture

### Data Flow
```
External CRM API → CRM Service → Vector Database → AI Response Generation
                         ↓
                  MongoDB Storage
```

### Storage Structure
- CRM configurations are stored in the `companies` collection under each page
- CRM data embeddings are stored in the `vector_documents` collection
- Each CRM link can be active for specific channels (Facebook, Messenger, or both)

## API Endpoints

### 1. Get CRM Links for Channel
```http
GET /api/dashboard/crm-links?page_id={pageID}&channel={channel}
```

**Query Parameters:**
- `page_id` (required): Facebook page ID
- `channel` (optional): "facebook" or "messenger"

**Response:**
```json
{
  "crm_links": [
    {
      "name": "Customer Database",
      "url": "https://api.crm.com/customers",
      "type": "api",
      "channels": ["facebook", "messenger"],
      "api_key": "***",
      "headers": {
        "X-Custom-Header": "value"
      },
      "description": "Main customer database",
      "is_active": true
    }
  ],
  "page_id": "123456789",
  "channel": "facebook",
  "total": 1,
  "active_count": 1,
  "inactive_count": 0
}
```

### 2. Add New CRM Link
```http
POST /api/dashboard/crm-links
```

**Request Body:**
```json
{
  "page_id": "123456789",
  "channels": ["facebook", "messenger"],
  "name": "Customer Database",
  "url": "https://api.crm.com/customers",
  "type": "api",
  "api_key": "your-api-key",
  "headers": {
    "X-Custom-Header": "value"
  },
  "description": "Main customer database",
  "is_active": true
}
```

**Parameters:**
- `page_id` (required): Facebook page ID
- `channels` (required): Array of channels ["facebook", "messenger"]
- `name` (required): Display name for the CRM link
- `url` (required): CRM endpoint URL
- `type` (optional): "api", "webhook", "database", or "file"
- `api_key` (optional): API key for authentication
- `headers` (optional): Custom headers for API requests
- `description` (optional): Description of the CRM link
- `is_active` (required): Whether the link is active

**Response:**
```json
{
  "message": "CRM link added successfully",
  "crm_link": { /* CRM link object */ },
  "channels": ["facebook", "messenger"],
  "page_id": "123456789"
}
```

### 3. Update CRM Link Status
```http
PUT /api/dashboard/crm-links/status
```

**Request Body:**
```json
{
  "page_id": "123456789",
  "url": "https://api.crm.com/customers",
  "channels": ["facebook"],
  "is_active": false
}
```

**Response:**
```json
{
  "message": "CRM link status updated successfully",
  "updated_count": 1,
  "url": "https://api.crm.com/customers",
  "channels": ["facebook"],
  "is_active": false
}
```

### 4. Delete CRM Link
```http
DELETE /api/dashboard/crm-links
```

**Request Body:**
```json
{
  "page_id": "123456789",
  "url": "https://api.crm.com/customers",
  "channels": ["facebook", "messenger"]
}
```

**Response:**
```json
{
  "message": "CRM link deleted successfully",
  "deleted_count": 2,
  "url": "https://api.crm.com/customers",
  "channels": ["facebook", "messenger"]
}
```

### 5. Get All CRM Data (All Pages)
```http
GET /api/dashboard/crm-links/all
```

**Response:**
```json
{
  "company_id": "company-001",
  "company_name": "My Company",
  "pages_data": [
    {
      "page_id": "123456789",
      "page_name": "My Facebook Page",
      "facebook_crm_links": [ /* CRM links */ ],
      "messenger_crm_links": [ /* CRM links */ ],
      "legacy_crm_links": [ /* Legacy links */ ],
      "total_active": 3,
      "total_inactive": 1
    }
  ],
  "total_pages": 1,
  "total_crm_links": 4,
  "total_active": 3,
  "total_inactive": 1
}
```

## CRM Data Types

### 1. API Type
For RESTful APIs that return JSON or plain text data.

**Example Configuration:**
```json
{
  "name": "Customer API",
  "url": "https://api.crm.com/v1/customers",
  "type": "api",
  "api_key": "sk-1234567890",
  "headers": {
    "Accept": "application/json"
  }
}
```

### 2. Webhook Type
For endpoints that push data to the system.

### 3. Database Type
For direct database connections (future implementation).

### 4. File Type
For static file-based CRM data.

## Data Processing

### 1. Initial Fetch
When a CRM link is added and marked as active:
1. System immediately fetches data from the CRM endpoint
2. Data is processed and converted to text format
3. Text is split into chunks if necessary
4. Each chunk is converted to embeddings
5. Embeddings are stored in the vector database

### 2. Scheduled Updates
The CRM scheduler runs periodically to:
1. Check all active CRM links
2. Fetch fresh data from each endpoint
3. Update existing embeddings or create new ones
4. Remove outdated embeddings

### 3. Data Format Handling

**JSON Response:**
```json
{
  "customers": [
    {
      "id": "cust-123",
      "name": "John Doe",
      "email": "john@example.com",
      "subscription": "Premium",
      "support_level": "Gold"
    }
  ]
}
```

**Processed Text Format:**
```
customers[0].id: cust-123
customers[0].name: John Doe
customers[0].email: john@example.com
customers[0].subscription: Premium
customers[0].support_level: Gold
```

## Channel-Specific Configuration

### Facebook Comments
CRM links configured for Facebook will be used when responding to comments on posts.

### Messenger
CRM links configured for Messenger will be used in chat conversations.

### Example: Different CRM for Each Channel
```javascript
// Facebook configuration
{
  "channels": ["facebook"],
  "url": "https://api.crm.com/social-customers"
}

// Messenger configuration
{
  "channels": ["messenger"],
  "url": "https://api.crm.com/chat-customers"
}

// Both channels
{
  "channels": ["facebook", "messenger"],
  "url": "https://api.crm.com/all-customers"
}
```

## Authentication Methods

### 1. Bearer Token
```json
{
  "api_key": "your-bearer-token"
}
```
Automatically added as: `Authorization: Bearer your-bearer-token`

### 2. Custom Headers
```json
{
  "headers": {
    "X-API-Key": "your-api-key",
    "X-Client-ID": "client-123"
  }
}
```

### 3. Combined Authentication
```json
{
  "api_key": "bearer-token",
  "headers": {
    "X-Additional-Auth": "extra-key"
  }
}
```

## Error Handling

### Common Errors

1. **401 Unauthorized**
   - Check API key validity
   - Verify authentication headers

2. **404 Not Found**
   - Verify CRM endpoint URL
   - Check if the resource exists

3. **429 Too Many Requests**
   - CRM scheduler will retry with backoff
   - Consider adjusting fetch frequency

4. **500 Server Error**
   - CRM link remains active but data not updated
   - System retries in next scheduled run

## Best Practices

### 1. Data Structure
- Keep CRM responses concise and relevant
- Include only customer-facing information
- Avoid sensitive data (SSN, passwords, etc.)

### 2. Update Frequency
- Set reasonable update intervals (e.g., every 30 minutes)
- Consider CRM API rate limits
- Use webhooks for real-time updates when possible

### 3. Security
- Store API keys securely in environment variables
- Use HTTPS endpoints only
- Rotate API keys regularly
- Limit CRM data to necessary fields

### 4. Performance
- Implement pagination for large datasets
- Cache frequently accessed data
- Use appropriate chunk sizes for embeddings
- Monitor vector database performance

## Integration Examples

### Example 1: Shopify Customer Data
```bash
curl -X POST http://localhost:8080/api/dashboard/crm-links \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer [session-token]" \
  -d '{
    "page_id": "123456789",
    "channels": ["messenger"],
    "name": "Shopify Customers",
    "url": "https://mystore.myshopify.com/admin/api/2024-01/customers.json",
    "type": "api",
    "api_key": "shppa_1234567890",
    "headers": {
      "X-Shopify-Access-Token": "shppa_1234567890"
    },
    "description": "Shopify customer database",
    "is_active": true
  }'
```

### Example 2: HubSpot CRM
```bash
curl -X POST http://localhost:8080/api/dashboard/crm-links \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer [session-token]" \
  -d '{
    "page_id": "123456789",
    "channels": ["facebook", "messenger"],
    "name": "HubSpot Contacts",
    "url": "https://api.hubapi.com/crm/v3/objects/contacts",
    "type": "api",
    "api_key": "pat-na1-1234567890",
    "headers": {
      "Content-Type": "application/json"
    },
    "description": "HubSpot CRM contacts",
    "is_active": true
  }'
```

### Example 3: Custom CRM System
```bash
curl -X POST http://localhost:8080/api/dashboard/crm-links \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer [session-token]" \
  -d '{
    "page_id": "123456789",
    "channels": ["facebook"],
    "name": "Internal CRM",
    "url": "https://crm.company.com/api/customers",
    "type": "api",
    "headers": {
      "X-API-Key": "internal-key-123",
      "X-Client-ID": "facebook-bot"
    },
    "description": "Company internal CRM",
    "is_active": true
  }'
```

## Monitoring & Debugging

### Check CRM Link Status
```bash
curl -X GET "http://localhost:8080/api/dashboard/crm-links?page_id=123456789" \
  -H "Authorization: Bearer [session-token]"
```

### View Vector Documents
Check if CRM data was successfully stored as embeddings:
```bash
curl -X GET "http://localhost:8080/api/dashboard/rag/documents/details?page_id=123456789&source=crm" \
  -H "Authorization: Bearer [session-token]"
```

### Debug Logs
Monitor application logs for CRM-related events:
```
INFO: CRM link added successfully
INFO: Starting CRM data fetch for page 123456789
INFO: Stored 45 embeddings from CRM data
WARN: Failed to fetch CRM data: 429 Too Many Requests
```

## Troubleshooting

### CRM Data Not Appearing in Responses
1. Verify CRM link is active
2. Check if data was fetched successfully (logs)
3. Verify embeddings were created in vector database
4. Ensure channel configuration matches usage

### Authentication Failures
1. Verify API key is correct
2. Check if headers are properly formatted
3. Test CRM endpoint directly with curl
4. Review CRM API documentation for changes

### Performance Issues
1. Reduce data size from CRM
2. Increase chunk size for embeddings
3. Implement caching layer
4. Optimize CRM query parameters

## Future Enhancements

- GraphQL support for CRM APIs
- OAuth 2.0 authentication flow
- Webhook receiver for real-time updates
- Data transformation pipelines
- CRM data validation and sanitization
- Multiple CRM source aggregation
- Incremental data updates
- CRM connection health dashboard
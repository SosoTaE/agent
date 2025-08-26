# CRM Link Management API

## Overview

The CRM Link Management API allows administrators to manage webhook/CRM URLs associated with Facebook pages. Each page can have multiple CRM links that can be individually enabled or disabled without removing them from the configuration.

## Data Model

### CRMLink Structure

```go
type CRMLink struct {
    Name        string            `json:"name"`        // Friendly name for the CRM link
    URL         string            `json:"url"`         // The webhook/API endpoint URL
    Type        string            `json:"type"`        // Type: api, webhook, database, file
    APIKey      string            `json:"api_key"`     // Optional API key for authentication
    Headers     map[string]string `json:"headers"`     // Optional custom headers
    Description string            `json:"description"` // Description of what this link does
    IsActive    bool              `json:"is_active"`   // Whether this link is currently active
}
```

## API Endpoints

### 1. Get CRM Links for a Page

Retrieve all CRM links configured for a specific page.

**Endpoint:** `GET /api/dashboard/pages/:pageID/crm-links`

**Authentication:** Required (session-based)

**Response:**
```json
{
  "crm_links": [
    {
      "name": "Sales CRM Webhook",
      "url": "https://crm.example.com/webhook/facebook",
      "type": "webhook",
      "api_key": "secret_key_123",
      "headers": {
        "X-Custom-Header": "value"
      },
      "description": "Sends customer data to sales CRM",
      "is_active": true
    },
    {
      "name": "Analytics API",
      "url": "https://analytics.example.com/api/events",
      "type": "api",
      "api_key": "analytics_key_456",
      "headers": {},
      "description": "Tracks customer interactions",
      "is_active": false
    }
  ],
  "page_id": "page_123456",
  "count": 2
}
```

**Error Responses:**
- `401 Unauthorized`: No valid session
- `400 Bad Request`: Page ID not provided
- `403 Forbidden`: Page doesn't belong to your company
- `500 Internal Server Error`: Database error

### 2. Toggle CRM Link Status

Enable or disable a specific CRM link without removing it from the configuration.

**Endpoint:** `PUT /api/dashboard/pages/:pageID/crm-links/toggle`

**Authentication:** Required (session-based)

**Request Body:**
```json
{
  "url": "https://crm.example.com/webhook/facebook",
  "is_active": false
}
```

**Response:**
```json
{
  "message": "CRM link status updated successfully",
  "url": "https://crm.example.com/webhook/facebook",
  "is_active": false
}
```

**Error Responses:**
- `401 Unauthorized`: No valid session
- `400 Bad Request`: Missing required fields
- `403 Forbidden`: Page doesn't belong to your company
- `404 Not Found`: CRM link with specified URL not found
- `500 Internal Server Error`: Database error

### 3. Update or Add CRM Link

Update an existing CRM link or add a new one to a page.

**Endpoint:** `PUT /api/dashboard/pages/:pageID/crm-links`

**Authentication:** Required (session-based)

**Request Body:**
```json
{
  "name": "New CRM Integration",
  "url": "https://newcrm.example.com/webhook",
  "type": "webhook",
  "api_key": "new_secret_key",
  "headers": {
    "Authorization": "Bearer token123",
    "Content-Type": "application/json"
  },
  "description": "New CRM integration for lead management",
  "is_active": true
}
```

**Response:**
```json
{
  "message": "CRM link added successfully",
  "crm_link": {
    "name": "New CRM Integration",
    "url": "https://newcrm.example.com/webhook",
    "type": "webhook",
    "api_key": "new_secret_key",
    "headers": {
      "Authorization": "Bearer token123",
      "Content-Type": "application/json"
    },
    "description": "New CRM integration for lead management",
    "is_active": true
  }
}
```

**Notes:**
- If a CRM link with the same URL already exists, it will be updated
- If the URL is new, a new CRM link will be added to the page

**Error Responses:**
- `401 Unauthorized`: No valid session
- `400 Bad Request`: Invalid request body or missing URL
- `403 Forbidden`: Page doesn't belong to your company
- `500 Internal Server Error`: Database error

## Usage Examples

### Example 1: Disable a Webhook Temporarily

When you need to perform maintenance on your CRM system, you can temporarily disable the webhook:

```bash
curl -X PUT https://your-bot.com/api/dashboard/pages/123456/crm-links/toggle \
  -H "Content-Type: application/json" \
  -H "Cookie: session=your_session_cookie" \
  -d '{
    "url": "https://crm.example.com/webhook/facebook",
    "is_active": false
  }'
```

### Example 2: Re-enable a Webhook

After maintenance is complete, re-enable the webhook:

```bash
curl -X PUT https://your-bot.com/api/dashboard/pages/123456/crm-links/toggle \
  -H "Content-Type: application/json" \
  -H "Cookie: session=your_session_cookie" \
  -d '{
    "url": "https://crm.example.com/webhook/facebook",
    "is_active": true
  }'
```

### Example 3: Add a New CRM Integration

Add a new CRM endpoint to your page:

```bash
curl -X PUT https://your-bot.com/api/dashboard/pages/123456/crm-links \
  -H "Content-Type: application/json" \
  -H "Cookie: session=your_session_cookie" \
  -d '{
    "name": "Backup CRM",
    "url": "https://backup-crm.example.com/api/messages",
    "type": "api",
    "api_key": "backup_key_789",
    "headers": {
      "X-Source": "Facebook"
    },
    "description": "Backup CRM for redundancy",
    "is_active": true
  }'
```

### Example 4: List All CRM Links

Get all CRM links for a page to see their status:

```bash
curl -X GET https://your-bot.com/api/dashboard/pages/123456/crm-links \
  -H "Cookie: session=your_session_cookie"
```

## JavaScript/TypeScript Client Example

```javascript
class CRMManager {
  constructor(baseUrl, sessionCookie) {
    this.baseUrl = baseUrl;
    this.headers = {
      'Content-Type': 'application/json',
      'Cookie': `session=${sessionCookie}`
    };
  }

  // Get all CRM links for a page
  async getCRMLinks(pageId) {
    const response = await fetch(
      `${this.baseUrl}/api/dashboard/pages/${pageId}/crm-links`,
      { headers: this.headers }
    );
    return response.json();
  }

  // Toggle CRM link status
  async toggleCRMLink(pageId, url, isActive) {
    const response = await fetch(
      `${this.baseUrl}/api/dashboard/pages/${pageId}/crm-links/toggle`,
      {
        method: 'PUT',
        headers: this.headers,
        body: JSON.stringify({ url, is_active: isActive })
      }
    );
    return response.json();
  }

  // Add or update CRM link
  async updateCRMLink(pageId, crmLink) {
    const response = await fetch(
      `${this.baseUrl}/api/dashboard/pages/${pageId}/crm-links`,
      {
        method: 'PUT',
        headers: this.headers,
        body: JSON.stringify(crmLink)
      }
    );
    return response.json();
  }

  // Disable all CRM links for a page
  async disableAllCRMLinks(pageId) {
    const { crm_links } = await this.getCRMLinks(pageId);
    const promises = crm_links
      .filter(link => link.is_active)
      .map(link => this.toggleCRMLink(pageId, link.url, false));
    return Promise.all(promises);
  }

  // Enable specific CRM links by name pattern
  async enableCRMLinksByPattern(pageId, pattern) {
    const { crm_links } = await this.getCRMLinks(pageId);
    const promises = crm_links
      .filter(link => link.name.includes(pattern) && !link.is_active)
      .map(link => this.toggleCRMLink(pageId, link.url, true));
    return Promise.all(promises);
  }
}

// Usage
const crmManager = new CRMManager('https://your-bot.com', 'session_cookie_value');

// Disable a specific webhook
await crmManager.toggleCRMLink('page_123', 'https://crm.example.com/webhook', false);

// Get all CRM links
const links = await crmManager.getCRMLinks('page_123');
console.log(`Total CRM links: ${links.count}`);
console.log(`Active links: ${links.crm_links.filter(l => l.is_active).length}`);

// Add new CRM link
await crmManager.updateCRMLink('page_123', {
  name: 'New Integration',
  url: 'https://new.example.com/webhook',
  type: 'webhook',
  is_active: true
});
```

## Security Considerations

1. **Authentication Required**: All endpoints require a valid session cookie
2. **Company Isolation**: Users can only manage CRM links for pages belonging to their company
3. **Page Ownership Validation**: Each request validates that the page belongs to the authenticated company
4. **URL Validation**: The system uses URL as the unique identifier for CRM links
5. **Cache Clearing**: The company cache is automatically cleared when CRM links are modified

## Database Impact

### MongoDB Update Operations

When toggling a CRM link, the system:
1. Finds the company document
2. Locates the specific page within the pages array
3. Finds the CRM link by URL within the page's crm_links array
4. Updates only the `is_active` field
5. Updates the company's `updated_at` timestamp
6. Clears the company cache

### Example MongoDB Query

```javascript
// Toggle CRM link status
db.companies.updateOne(
  {
    "company_id": "company_123",
    "pages.page_id": "page_456"
  },
  {
    "$set": {
      "pages.0.crm_links.1.is_active": false,
      "updated_at": ISODate("2024-01-15T10:30:00Z")
    }
  }
)
```

## Best Practices

1. **Regular Status Checks**: Periodically check CRM link status to ensure critical integrations are active
2. **Maintenance Windows**: Disable CRM links during maintenance to prevent failed webhook calls
3. **Fallback URLs**: Configure multiple CRM links for redundancy
4. **Monitoring**: Log and monitor CRM link status changes
5. **Testing**: Test CRM links after re-enabling to ensure they're working correctly

## Error Handling

### Common Errors and Solutions

| Error | Cause | Solution |
|-------|-------|----------|
| "CRM link with URL not found" | URL doesn't exist in configuration | Verify URL or add new CRM link |
| "Page not found or access denied" | Invalid page ID or no permission | Check page ID and company ownership |
| "No documents were updated" | Database update failed | Retry operation or check database connection |
| "Company or page not found" | Invalid company/page combination | Verify company and page IDs |

## Monitoring

Track these events for audit and debugging:

- CRM link status changes (enable/disable)
- New CRM links added
- CRM link updates
- Failed toggle attempts
- Cache clearing events

### Log Examples

```
INFO: CRM link status updated
  companyID: company_123
  pageID: page_456
  crmURL: https://crm.example.com/webhook
  isActive: false

ERROR: Failed to toggle CRM link status
  companyID: company_123
  pageID: page_456
  url: https://invalid.example.com
  error: CRM link with URL https://invalid.example.com not found
```

## Migration Notes

For existing installations:
1. The `IsActive` field already exists in the CRMLink model
2. Default value for existing CRM links is `false` (inactive)
3. Set existing CRM links to `is_active: true` if they should be enabled

## Future Enhancements

Potential improvements to consider:
- Bulk operations (enable/disable multiple CRM links)
- CRM link health checks
- Automatic retry for failed webhook calls
- CRM link execution history
- Rate limiting per CRM link
- Webhook payload transformation
- CRM link testing endpoint
# CRM and RAG All Data API Documentation

## Overview
These endpoints provide comprehensive views of all CRM links and RAG documents across all pages in a company. They're useful for dashboard overviews, auditing, and bulk management operations.

## Authentication
Both endpoints require authentication via session cookie.

## Base URL
```
http://localhost:3000/api/dashboard
```

---

## 1. Get All CRM Data

**Endpoint:** `GET /crm-links/all`

**Description:** Retrieves all CRM links from all pages within the authenticated company, organized by page and channel.

### Request
```bash
curl -X GET "http://localhost:3000/api/dashboard/crm-links/all" \
  -H "Cookie: session=your_session_cookie"
```

### Response
```json
{
  "company_id": "comp_123456",
  "company_name": "Example Company",
  "pages_data": [
    {
      "page_id": "461998383671026",
      "page_name": "Example Store",
      "facebook_crm_links": [
        {
          "name": "Product API",
          "url": "https://api.example.com/products",
          "type": "api",
          "channels": ["facebook"],
          "is_active": true,
          "api_key": "sk-xxx",
          "headers": {
            "X-Custom-Header": "value"
          },
          "description": "Product catalog API"
        },
        {
          "name": "Inventory API",
          "url": "https://api.example.com/inventory",
          "type": "api",
          "channels": ["facebook"],
          "is_active": false,
          "api_key": "sk-yyy",
          "headers": {},
          "description": "Real-time inventory data"
        }
      ],
      "messenger_crm_links": [
        {
          "name": "Customer Support API",
          "url": "https://api.example.com/support",
          "type": "api",
          "channels": ["messenger"],
          "is_active": true,
          "api_key": "sk-zzz",
          "headers": {},
          "description": "Customer support tickets"
        }
      ],
      "legacy_crm_links": [
        {
          "name": "Old Product Feed",
          "url": "https://legacy.example.com/feed",
          "type": "webhook",
          "channels": ["facebook", "messenger"],
          "is_active": false,
          "api_key": "",
          "headers": {},
          "description": "Legacy product feed (deprecated)"
        }
      ],
      "total_active": 2,
      "total_inactive": 2
    },
    {
      "page_id": "789012345678901",
      "page_name": "Example Support",
      "facebook_crm_links": [],
      "messenger_crm_links": [
        {
          "name": "FAQ Database",
          "url": "https://api.example.com/faq",
          "type": "database",
          "channels": ["messenger"],
          "is_active": true,
          "api_key": "",
          "headers": {},
          "description": "Frequently asked questions"
        }
      ],
      "legacy_crm_links": [],
      "total_active": 1,
      "total_inactive": 0
    }
  ],
  "total_pages": 2,
  "total_crm_links": 5,
  "total_active": 3,
  "total_inactive": 2
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `company_id` | string | The authenticated company's ID |
| `company_name` | string | The company's display name |
| `pages_data` | array | Array of page-specific CRM data |
| `pages_data[].page_id` | string | Facebook page ID |
| `pages_data[].page_name` | string | Facebook page name |
| `pages_data[].facebook_crm_links` | array | CRM links active for Facebook comments |
| `pages_data[].messenger_crm_links` | array | CRM links active for Messenger |
| `pages_data[].legacy_crm_links` | array | CRM links from legacy configuration |
| `pages_data[].total_active` | number | Count of active links for this page |
| `pages_data[].total_inactive` | number | Count of inactive links for this page |
| `total_pages` | number | Total number of pages in the company |
| `total_crm_links` | number | Total count of all CRM links |
| `total_active` | number | Total count of active CRM links |
| `total_inactive` | number | Total count of inactive CRM links |

### CRM Link Object

Each CRM link contains:
- `name` - Display name of the CRM link
- `url` - The API endpoint or webhook URL
- `type` - Type of integration ("api", "webhook", "database", "file")
- `channels` - Array of channels where CRM link is active (["facebook"], ["messenger"], or ["facebook", "messenger"])
- `is_active` - Whether the link is currently active
- `api_key` - API key for authentication (if applicable)
- `headers` - Custom headers for the request
- `description` - Description of the CRM link's purpose

### Status Codes
- `200 OK` - Success
- `401 Unauthorized` - Missing or invalid session
- `404 Not Found` - Company not found
- `500 Internal Server Error` - Server error

---

## 2. Get All RAG Documents

**Endpoint:** `GET /rag/documents/all`

**Description:** Retrieves all RAG documents from all pages within the authenticated company, with detailed statistics and grouping.

### Request
```bash
curl -X GET "http://localhost:3000/api/dashboard/rag/documents/all" \
  -H "Cookie: session=your_session_cookie"
```

### Response
```json
{
  "company_id": "comp_123456",
  "company_name": "Example Company",
  "pages_data": [
    {
      "page_id": "461998383671026",
      "page_name": "Example Store",
      "documents": [
        {
          "_id": "507f1f77bcf86cd799439011",
          "company_id": "comp_123456",
          "page_id": "461998383671026",
          "content": "Our premium products include...",
          "channels": ["facebook", "messenger"],
          "source": "manual_upload",
          "is_active": true,
          "metadata": {
            "filename": "products.pdf",
            "chunk": "1/5",
            "uploaded_by": "admin@example.com",
            "upload_time": "2024-01-15T10:30:00Z"
          },
          "created_at": "2024-01-15T10:30:00Z",
          "updated_at": "2024-01-15T10:30:00Z"
        },
        {
          "_id": "507f1f77bcf86cd799439012",
          "company_id": "comp_123456",
          "page_id": "461998383671026",
          "content": "Product specifications and details...",
          "channels": ["facebook", "messenger"],
          "source": "manual_upload",
          "is_active": true,
          "metadata": {
            "filename": "products.pdf",
            "chunk": "2/5",
            "uploaded_by": "admin@example.com",
            "upload_time": "2024-01-15T10:30:00Z"
          },
          "created_at": "2024-01-15T10:30:00Z",
          "updated_at": "2024-01-15T10:30:00Z"
        }
      ],
      "document_groups": {
        "products.pdf": {
          "total_chunks": 5,
          "active_chunks": 5,
          "channels": ["facebook", "messenger"],
          "total_size": 15420,
          "source": "manual_upload",
          "created_at": "2024-01-15T10:30:00Z"
        },
        "shipping_policy.txt": {
          "total_chunks": 2,
          "active_chunks": 2,
          "channels": ["messenger"],
          "total_size": 4200,
          "source": "manual_upload",
          "created_at": "2024-01-16T14:20:00Z"
        },
        "faq.md": {
          "total_chunks": 3,
          "active_chunks": 0,
          "channels": [],
          "total_size": 8900,
          "source": "manual_upload",
          "created_at": "2024-01-10T09:15:00Z"
        }
      },
      "total_documents": 10,
      "active_documents": 7,
      "inactive_documents": 3,
      "total_size": 28520,
      "unique_files": 3
    },
    {
      "page_id": "789012345678901",
      "page_name": "Example Support",
      "documents": [],
      "document_groups": {},
      "total_documents": 0,
      "active_documents": 0,
      "inactive_documents": 0,
      "total_size": 0,
      "unique_files": 0
    }
  ],
  "total_pages": 2,
  "total_documents": 10,
  "total_active": 7,
  "total_inactive": 3,
  "total_size": 28520,
  "size_mb": 0.0272
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `company_id` | string | The authenticated company's ID |
| `company_name` | string | The company's display name |
| `pages_data` | array | Array of page-specific RAG document data |
| `pages_data[].page_id` | string | Facebook page ID |
| `pages_data[].page_name` | string | Facebook page name |
| `pages_data[].documents` | array | All document chunks for this page |
| `pages_data[].document_groups` | object | Documents grouped by filename with statistics |
| `pages_data[].total_documents` | number | Total document chunks for this page |
| `pages_data[].active_documents` | number | Count of active document chunks |
| `pages_data[].inactive_documents` | number | Count of inactive document chunks |
| `pages_data[].total_size` | number | Total size in bytes for this page |
| `pages_data[].unique_files` | number | Number of unique files uploaded |
| `total_pages` | number | Total number of pages in the company |
| `total_documents` | number | Total count of all document chunks |
| `total_active` | number | Total count of active document chunks |
| `total_inactive` | number | Total count of inactive document chunks |
| `total_size` | number | Total size of all documents in bytes |
| `size_mb` | number | Total size converted to megabytes |

### Document Object

Each document contains:
- `_id` - MongoDB ObjectID of the document
- `company_id` - Company identifier
- `page_id` - Facebook page ID
- `content` - The actual text content of the document chunk
- `channels` - Array of channels where document is active (["facebook"], ["messenger"], or both)
- `source` - Source type ("manual_upload", "crm", etc.)
- `is_active` - Whether the document is currently active
- `metadata` - Additional metadata including filename, chunk info, uploader
- `created_at` - Document creation timestamp
- `updated_at` - Last update timestamp

### Document Group Object

Each document group contains:
- `total_chunks` - Total number of chunks for this file
- `active_chunks` - Number of active chunks
- `channels` - Combined channels from all chunks
- `total_size` - Total size of all chunks in bytes
- `source` - Document source type
- `created_at` - When the first chunk was created

### Status Codes
- `200 OK` - Success
- `401 Unauthorized` - Missing or invalid session
- `404 Not Found` - Company not found
- `500 Internal Server Error` - Server error

---

## Use Cases

### 1. Dashboard Overview
Use these endpoints to create a comprehensive dashboard showing:
- Total CRM integrations across all pages
- Active vs inactive CRM links
- RAG document coverage by page
- Storage usage statistics
- Channel-specific configurations

### 2. Audit and Compliance
- Review all active integrations
- Check for outdated or inactive CRM links
- Verify document channel assignments
- Monitor storage usage

### 3. Bulk Operations Planning
- Identify pages without CRM links
- Find inactive documents to clean up
- Plan channel migrations
- Estimate resource usage

### 4. Debugging and Support
- Quick overview of customer configuration
- Identify misconfigured integrations
- Check document availability by page
- Verify channel-specific setups

---

## Example Usage

### JavaScript/Fetch Example
```javascript
// Get all CRM data
async function getAllCRMData() {
  const response = await fetch('http://localhost:3000/api/dashboard/crm-links/all', {
    method: 'GET',
    credentials: 'include' // Include cookies
  });
  
  if (response.ok) {
    const data = await response.json();
    console.log(`Total CRM Links: ${data.total_crm_links}`);
    console.log(`Active: ${data.total_active}, Inactive: ${data.total_inactive}`);
    
    // Process each page
    data.pages_data.forEach(page => {
      console.log(`Page ${page.page_name}:`);
      console.log(`  Facebook CRM Links: ${page.facebook_crm_links.length}`);
      console.log(`  Messenger CRM Links: ${page.messenger_crm_links.length}`);
    });
  }
}

// Get all RAG documents
async function getAllRAGDocuments() {
  const response = await fetch('http://localhost:3000/api/dashboard/rag/documents/all', {
    method: 'GET',
    credentials: 'include'
  });
  
  if (response.ok) {
    const data = await response.json();
    console.log(`Total Documents: ${data.total_documents}`);
    console.log(`Storage Used: ${data.size_mb.toFixed(2)} MB`);
    
    // Process each page
    data.pages_data.forEach(page => {
      console.log(`Page ${page.page_name}:`);
      console.log(`  Documents: ${page.total_documents}`);
      console.log(`  Unique Files: ${page.unique_files}`);
      
      // Show document groups
      Object.entries(page.document_groups).forEach(([filename, info]) => {
        console.log(`    ${filename}: ${info.active_chunks}/${info.total_chunks} chunks active`);
      });
    });
  }
}
```

### Python Example
```python
import requests

# Session cookie from login
cookies = {'session': 'your_session_cookie'}

# Get all CRM data
response = requests.get(
    'http://localhost:3000/api/dashboard/crm-links/all',
    cookies=cookies
)

if response.status_code == 200:
    data = response.json()
    print(f"Company: {data['company_name']}")
    print(f"Total CRM Links: {data['total_crm_links']}")
    
    for page in data['pages_data']:
        print(f"\nPage: {page['page_name']} ({page['page_id']})")
        print(f"  Active: {page['total_active']}, Inactive: {page['total_inactive']}")

# Get all RAG documents
response = requests.get(
    'http://localhost:3000/api/dashboard/rag/documents/all',
    cookies=cookies
)

if response.status_code == 200:
    data = response.json()
    print(f"\nTotal Storage: {data['size_mb']:.2f} MB")
    print(f"Total Documents: {data['total_documents']}")
    
    for page in data['pages_data']:
        if page['total_documents'] > 0:
            print(f"\nPage: {page['page_name']}")
            print(f"  Files: {page['unique_files']}")
            print(f"  Documents: {page['active_documents']} active / {page['total_documents']} total")
```

---

## Performance Notes

### Response Size
- These endpoints return comprehensive data and may have large response sizes
- For companies with many pages or documents, responses can be several MB
- Consider implementing pagination if response size becomes an issue

### Caching
- Results are computed in real-time from the database
- Consider caching responses for frequently accessed data
- Cache invalidation should occur when CRM links or documents are modified

### Timeout Considerations
- The RAG documents endpoint has a 30-second timeout due to potentially large datasets
- The CRM links endpoint has a 10-second timeout
- For very large companies, consider implementing streaming responses

---

## Security Considerations

1. **Authentication Required**: Both endpoints require valid session authentication
2. **Company Isolation**: Users can only access data for their authenticated company
3. **Sensitive Data**: Response may contain API keys and sensitive configuration
4. **Audit Logging**: Consider logging access to these comprehensive data endpoints
5. **Rate Limiting**: Consider implementing rate limits for these resource-intensive endpoints

---

## Troubleshooting

### Empty Response Data
If `pages_data` is empty:
- Verify the company has pages configured
- Check that pages have been properly added to the company
- Ensure the session has the correct company_id

### Missing Documents
If documents are not appearing:
- Verify documents exist in the vector_documents collection
- Check that company_id and page_id match exactly
- Ensure documents haven't been deleted

### Performance Issues
If the endpoints are slow:
- Check MongoDB indexes on company_id and page_id
- Consider adding pagination for large datasets
- Monitor MongoDB query performance
- Check network latency between application and database

---

## Related Endpoints

- `GET /crm-links` - Get CRM links for a specific channel
- `GET /rag/documents/details` - Get detailed RAG documents with filtering
- `POST /crm-links` - Add new CRM link
- `POST /rag/upload` - Upload new RAG document
- `PUT /crm-links/status` - Update CRM link status
- `PUT /rag/document/channels` - Update document channels
# RAG Endpoints API Documentation

This document covers the essential RAG (Retrieval-Augmented Generation) endpoints for managing documents in the Facebook bot system.

## Base URL
```
https://your-domain.com/dashboard
```

## Authentication
All endpoints require Bearer token authentication:
```
Authorization: Bearer <your-token>
```

---

## 1. Upload RAG Document

Upload a file for RAG processing. Files are automatically split into chunks and vectorized for retrieval.

### Endpoint
```http
POST /dashboard/rag/upload
```

### Headers
```
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

### Request Body (Form Data)
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file` | File | Yes | The file to upload (.txt, .pdf, .md, .csv, .json) |
| `page_id` | String | Yes | Facebook page ID |
| `channels` | String | Yes | Comma-separated channels: "facebook", "messenger", or "facebook,messenger" |
| `source` | String | No | Source type (default: "manual_upload") |
| `metadata` | String | No | Comma-separated key=value pairs |

### Example Request
```javascript
const formData = new FormData();
formData.append('file', fileInput.files[0]);
formData.append('page_id', 'page_123456');
formData.append('channels', 'facebook,messenger');

fetch('/dashboard/rag/upload', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer YOUR_TOKEN'
  },
  body: formData
});
```

### Success Response (202 Accepted)
```json
{
  "message": "File received and queued for processing",
  "filename": "product_catalog.pdf",
  "size": 204800,
  "source": "manual_upload",
  "page_id": "page_123456",
  "channels": ["facebook", "messenger"],
  "status": "processing"
}
```

### Error Responses
```json
// 400 Bad Request - No file
{
  "error": "No file uploaded"
}

// 400 Bad Request - Invalid channel
{
  "error": "invalid channel: instagram (must be 'facebook' or 'messenger')"
}

// 400 Bad Request - File too large
{
  "error": "File size exceeds 10MB limit"
}
```

---

## 2. Delete RAG Document

Delete a document from the vector database by filename.

### Endpoint
```http
DELETE /dashboard/rag/document
```

### Query Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `company_id` | String | Yes | Company identifier |
| `page_id` | String | Yes | Facebook page ID |
| `filename` | String | Yes* | Name of file to delete |
| `crm_url` | String | Yes* | CRM URL (alternative to filename) |

*Either `filename` or `crm_url` must be provided

### Example Request
```javascript
const params = new URLSearchParams({
  company_id: 'comp_123',
  page_id: 'page_456',
  filename: 'old_document.pdf'
});

fetch(`/dashboard/rag/document?${params}`, {
  method: 'DELETE',
  headers: {
    'Authorization': 'Bearer YOUR_TOKEN'
  }
});
```

### Success Response (200 OK)
```json
{
  "message": "Successfully deleted 15 document(s)",
  "deleted": 15
}
```

### Error Responses
```json
// 400 Bad Request
{
  "error": "Either content or filename must be provided"
}

// 404 Not Found
{
  "error": "Page not found or access denied"
}
```

---

## 3. Toggle Document Channel

Enable or disable a document for a specific platform (Facebook or Messenger).

### Endpoint
```http
PUT /dashboard/rag/document/toggle-channel
```

### Headers
```
Authorization: Bearer <token>
Content-Type: application/json
```

### Request Body
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `document_name` | String | Yes | Filename of the document |
| `platform_name` | String | Yes | Platform: "facebook" or "messenger" |
| `value` | Boolean | Yes | true to enable, false to disable |

### Example Request
```javascript
fetch('/dashboard/rag/document/toggle-channel', {
  method: 'PUT',
  headers: {
    'Authorization': 'Bearer YOUR_TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    document_name: 'faq.pdf',
    platform_name: 'facebook',
    value: true
  })
});
```

### Success Response (200 OK)
```json
{
  "message": "Document channel updated successfully",
  "document_name": "faq.pdf",
  "platform": "facebook",
  "value": true,
  "updated_count": 12,
  "pages_updated": ["page_123", "page_456"]
}
```

### Error Responses
```json
// 400 Bad Request - Missing field
{
  "error": "document_name is required"
}

// 400 Bad Request - Invalid platform
{
  "error": "invalid platform_name: instagram (must be 'facebook' or 'messenger')"
}

// 404 Not Found
{
  "error": "No documents found with name 'unknown.pdf'"
}
```

---

## 4. Get All RAG Files

Retrieve all unique RAG files with their channel status.

### Endpoint
```http
GET /dashboard/rag/files
```

### Headers
```
Authorization: Bearer <token>
```

### Query Parameters
None required - automatically uses authenticated company ID

### Example Request
```javascript
fetch('/dashboard/rag/files', {
  method: 'GET',
  headers: {
    'Authorization': 'Bearer YOUR_TOKEN'
  }
});
```

### Success Response (200 OK)
```json
[
  {
    "filename": "product_catalog.pdf",
    "facebook": true,
    "messenger": false
  },
  {
    "filename": "faq.txt",
    "facebook": true,
    "messenger": true
  },
  {
    "filename": "terms_of_service.md",
    "facebook": false,
    "messenger": false
  }
]
```

### Response Fields
| Field | Type | Description |
|-------|------|-------------|
| `filename` | String | Name of the uploaded file |
| `facebook` | Boolean | Whether file is active for Facebook comments |
| `messenger` | Boolean | Whether file is active for Messenger |

### Error Responses
```json
// 401 Unauthorized
{
  "error": "Company ID not found in session"
}

// 500 Internal Server Error
{
  "error": "Failed to get documents: database connection error"
}
```

---

## Complete Integration Example

Here's how to use all endpoints together in a document management flow:

```javascript
class RAGDocumentManager {
  constructor(token) {
    this.token = token;
    this.baseURL = '/dashboard';
  }

  // 1. Get all files
  async getAllFiles() {
    const response = await fetch(`${this.baseURL}/rag/files`, {
      headers: { 'Authorization': `Bearer ${this.token}` }
    });
    return response.json();
  }

  // 2. Upload new file
  async uploadFile(file, pageId, channels) {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('page_id', pageId);
    formData.append('channels', channels.join(','));

    const response = await fetch(`${this.baseURL}/rag/upload`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${this.token}` },
      body: formData
    });
    return response.json();
  }

  // 3. Toggle channel for file
  async toggleChannel(filename, platform, enabled) {
    const response = await fetch(`${this.baseURL}/rag/document/toggle-channel`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${this.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        document_name: filename,
        platform_name: platform,
        value: enabled
      })
    });
    return response.json();
  }

  // 4. Delete file
  async deleteFile(companyId, pageId, filename) {
    const params = new URLSearchParams({
      company_id: companyId,
      page_id: pageId,
      filename: filename
    });

    const response = await fetch(`${this.baseURL}/rag/document?${params}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${this.token}` }
    });
    return response.json();
  }
}

// Usage
const manager = new RAGDocumentManager('YOUR_TOKEN');

// Load and display files
const files = await manager.getAllFiles();
console.log(`Total files: ${files.length}`);

// Upload new file
const fileInput = document.getElementById('file-input');
const result = await manager.uploadFile(
  fileInput.files[0],
  'page_123',
  ['facebook', 'messenger']
);

// Toggle Facebook for a file
await manager.toggleChannel('faq.pdf', 'facebook', true);

// Delete old file
await manager.deleteFile('comp_123', 'page_456', 'old_doc.pdf');
```

---

## Important Notes

### File Processing
- Files are processed asynchronously after upload
- Large files are automatically split into chunks (typically 2000 characters)
- Processing usually takes 3-5 seconds
- Each chunk is stored separately but grouped by filename

### Channel Behavior
- Channels are independent - a file can be active on Facebook, Messenger, both, or neither
- When toggling channels, all chunks of a file are updated
- If any chunk is enabled for a channel, the entire file shows as enabled

### File Formats
- Supported: `.txt`, `.pdf`, `.md`, `.csv`, `.json`
- Maximum file size: 10MB
- Unsupported formats will return an error

### Rate Limiting
- Upload endpoint: 10 requests per minute
- Other endpoints: 100 requests per minute

### Best Practices
1. **After Upload**: Wait 3-5 seconds before refreshing the file list
2. **Toggle Confirmation**: Show user feedback immediately (optimistic UI)
3. **Delete Confirmation**: Always confirm with user before deleting
4. **Error Handling**: Implement retry logic for network failures
5. **Batch Operations**: Add small delays (100ms) between multiple operations

---

## Error Codes Summary

| Code | Meaning | Action |
|------|---------|--------|
| 200 | Success | Continue |
| 202 | Accepted (processing) | Wait and check later |
| 400 | Bad Request | Check parameters |
| 401 | Unauthorized | Refresh token |
| 403 | Forbidden | Check permissions |
| 404 | Not Found | Verify resource exists |
| 500 | Server Error | Retry after delay |

---

## Testing Endpoints

### cURL Examples

```bash
# Get all files
curl -X GET "https://your-domain.com/dashboard/rag/files" \
  -H "Authorization: Bearer YOUR_TOKEN"

# Upload file
curl -X POST "https://your-domain.com/dashboard/rag/upload" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@document.pdf" \
  -F "page_id=page_123" \
  -F "channels=facebook,messenger"

# Toggle channel
curl -X PUT "https://your-domain.com/dashboard/rag/document/toggle-channel" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"document_name":"faq.pdf","platform_name":"facebook","value":true}'

# Delete file
curl -X DELETE "https://your-domain.com/dashboard/rag/document?company_id=comp_123&page_id=page_456&filename=old.pdf" \
  -H "Authorization: Bearer YOUR_TOKEN"
```
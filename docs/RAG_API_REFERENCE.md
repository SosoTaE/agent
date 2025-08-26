# RAG Document Management API Reference

## Overview

This document provides detailed API documentation for all RAG (Retrieval-Augmented Generation) document management endpoints. These endpoints allow you to upload, manage, and control documents that enhance the AI bot's knowledge base.

**Base URL:** `/api/dashboard/rag`

**Authentication:** All endpoints require authentication via session token in the Authorization header or cookie.

---

## 1. Upload RAG Document

Upload a document file for RAG processing. The file will be processed asynchronously in the background.

### Endpoint
```
POST /api/dashboard/rag/upload
```

### Request

**Headers:**
```
Authorization: Bearer [session-token]
Content-Type: multipart/form-data
```

**Form Data Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `file` | File | Yes | The document file to upload (max 10MB) |
| `page_id` | String | Yes | Facebook page ID |
| `channels` | String | Yes | Comma-separated list of channels: "facebook", "messenger", or "facebook,messenger" |
| `source` | String | No | Source identifier (default: "manual_upload") |
| `metadata` | String | No | Key-value pairs in format "key1=value1,key2=value2" |

**Supported File Types:**
- `.txt` - Plain text files
- `.md` - Markdown files  
- `.csv` - Comma-separated values
- `.json` - JSON data files

### Response

**Success Response (202 Accepted):**
```json
{
  "message": "File received and queued for processing",
  "filename": "company_policies.txt",
  "size": 45678,
  "source": "manual_upload",
  "page_id": "123456789",
  "channels": ["facebook", "messenger"],
  "status": "processing"
}
```

**Error Responses:**

*400 Bad Request - Missing Required Fields:*
```json
{
  "error": "page_id is required"
}
```

*400 Bad Request - Invalid Channel:*
```json
{
  "error": "invalid channel: twitter (must be 'facebook' or 'messenger')"
}
```

*400 Bad Request - File Too Large:*
```json
{
  "error": "File size exceeds 10MB limit"
}
```

*400 Bad Request - Unsupported File Type:*
```json
{
  "error": "unsupported file type. Supported types: .txt, .md, .csv, .json"
}
```

*401 Unauthorized:*
```json
{
  "error": "Company ID not found in session"
}
```

*403 Forbidden:*
```json
{
  "error": "Page not found or access denied"
}
```

### Example

```bash
curl -X POST http://localhost:8080/api/dashboard/rag/upload \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -F "file=@/path/to/faq.txt" \
  -F "page_id=123456789" \
  -F "channels=facebook,messenger" \
  -F "source=faq_docs" \
  -F "metadata=version=2.0,category=support"
```

### Notes
- Documents are processed asynchronously in the background
- Large documents are automatically split into chunks (2000 chars each)
- Each chunk is stored with metadata including chunk number
- Processing status can be monitored through application logs

---

## 2. Delete RAG Document

Delete a document from the vector database by filename or content.

### Endpoint
```
DELETE /api/dashboard/rag/document
```

### Request

**Headers:**
```
Authorization: Bearer [session-token]
Content-Type: application/json
```

**Body Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page_id` | String | Yes | Facebook page ID |
| `filename` | String | No* | Filename to delete (all chunks) |
| `content` | String | No* | Specific content to delete |

*Note: Either `filename` or `content` must be provided, but not both.

### Response

**Success Response (200 OK):**
```json
{
  "message": "Successfully deleted 5 document(s)",
  "deleted": 5
}
```

**Error Responses:**

*400 Bad Request - Missing Identifier:*
```json
{
  "error": "Either content or filename must be provided"
}
```

*403 Forbidden:*
```json
{
  "error": "Page not found or access denied"
}
```

*404 Not Found:*
```json
{
  "message": "Successfully deleted 0 document(s)",
  "deleted": 0
}
```

### Examples

**Delete by Filename:**
```bash
curl -X DELETE http://localhost:8080/api/dashboard/rag/document \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "123456789",
    "filename": "old_policies.txt"
  }'
```

**Delete by Content:**
```bash
curl -X DELETE http://localhost:8080/api/dashboard/rag/document \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "123456789",
    "content": "This specific content needs to be removed"
  }'
```

### Notes
- Deleting by filename removes all chunks associated with that file
- Deleting by content only removes exact matches
- This operation is permanent and cannot be undone

---

## 3. List RAG Documents

Retrieve all RAG documents for a specific page, grouped by filename.

### Endpoint
```
GET /api/dashboard/rag/documents
```

### Request

**Headers:**
```
Authorization: Bearer [session-token]
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page_id` | String | Yes | Facebook page ID |

### Response

**Success Response (200 OK):**
```json
{
  "documents": [
    {
      "id": "507f1f77bcf86cd799439011",
      "content": "Frequently Asked Questions\n\nQ: What are your hours?...",
      "source": "manual_upload",
      "is_active": true,
      "created_at": "2024-01-15T10:30:00Z",
      "metadata": {
        "filename": "faq.txt",
        "chunk": "1/3",
        "uploaded_by": "admin@company.com",
        "upload_time": "2024-01-15T10:30:00Z"
      }
    }
  ],
  "document_groups": {
    "faq.txt": [
      {
        "id": "507f1f77bcf86cd799439011",
        "content": "FAQ chunk 1...",
        "source": "manual_upload",
        "is_active": true,
        "created_at": "2024-01-15T10:30:00Z",
        "metadata": {
          "filename": "faq.txt",
          "chunk": "1/3"
        }
      },
      {
        "id": "507f1f77bcf86cd799439012",
        "content": "FAQ chunk 2...",
        "source": "manual_upload",
        "is_active": true,
        "created_at": "2024-01-15T10:30:00Z",
        "metadata": {
          "filename": "faq.txt",
          "chunk": "2/3"
        }
      }
    ],
    "policies.md": [
      {
        "id": "507f1f77bcf86cd799439013",
        "content": "Company policies content...",
        "source": "manual_upload",
        "is_active": false,
        "created_at": "2024-01-14T09:00:00Z",
        "metadata": {
          "filename": "policies.md"
        }
      }
    ]
  },
  "total": 3
}
```

**Error Responses:**

*400 Bad Request:*
```json
{
  "error": "page_id is required"
}
```

*403 Forbidden:*
```json
{
  "error": "Page not found or access denied"
}
```

### Example

```bash
curl -X GET "http://localhost:8080/api/dashboard/rag/documents?page_id=123456789" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

### Notes
- Documents are grouped by filename for easier management
- Non-file documents are grouped by content preview (first 50 chars)
- Includes all metadata associated with each document chunk

---

## 4. Toggle RAG Document Status

Toggle the active status of RAG documents by filename or document ID.

### Endpoint
```
PUT /api/dashboard/rag/document/toggle
```

### Request

**Headers:**
```
Authorization: Bearer [session-token]
Content-Type: application/json
```

**Body Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page_id` | String | Yes | Facebook page ID |
| `filename` | String | No* | Toggle all chunks with this filename |
| `document_id` | String | No* | Toggle specific document by ID |
| `is_active` | Boolean | Yes | New active status |

*Note: Either `filename` or `document_id` must be provided, but not both.

### Response

**Success Response (200 OK):**
```json
{
  "message": "Successfully updated 3 document(s)",
  "updated": 3,
  "is_active": false
}
```

**Error Responses:**

*400 Bad Request - Missing Identifier:*
```json
{
  "error": "Either filename or document_id must be provided"
}
```

*404 Not Found:*
```json
{
  "error": "No documents found to update"
}
```

### Examples

**Toggle by Filename:**
```bash
curl -X PUT http://localhost:8080/api/dashboard/rag/document/toggle \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "123456789",
    "filename": "seasonal_promotions.txt",
    "is_active": false
  }'
```

**Toggle by Document ID:**
```bash
curl -X PUT http://localhost:8080/api/dashboard/rag/document/toggle \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "123456789",
    "document_id": "507f1f77bcf86cd799439011",
    "is_active": true
  }'
```

### Notes
- Toggling by filename affects all chunks of that document
- Toggling by ID affects only the specific chunk
- Inactive documents are not used in AI response generation

---

## 5. Update Document Channels

Update which channels (Facebook/Messenger) a document is active for.

### Endpoint
```
PUT /api/dashboard/rag/document/channels
```

### Request

**Headers:**
```
Authorization: Bearer [session-token]
Content-Type: application/json
```

**Body Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page_id` | String | Yes | Facebook page ID |
| `filename` | String | Yes | Document filename |
| `channels` | Array | Yes | Array of channels: ["facebook"], ["messenger"], or ["facebook", "messenger"]. Empty array deactivates document |

### Response

**Success Response (200 OK):**
```json
{
  "message": "Document channels updated successfully",
  "updated_count": 3,
  "channels": ["messenger"],
  "document_active": true
}
```

**Error Responses:**

*400 Bad Request - Invalid Channel:*
```json
{
  "error": "invalid channel: twitter"
}
```

*404 Not Found:*
```json
{
  "message": "Document channels updated successfully",
  "updated_count": 0,
  "channels": ["messenger"],
  "document_active": false
}
```

### Example

**Set Document to Messenger Only:**
```bash
curl -X PUT http://localhost:8080/api/dashboard/rag/document/channels \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "123456789",
    "filename": "chat_support_guide.txt",
    "channels": ["messenger"]
  }'
```

**Deactivate Document for All Channels:**
```bash
curl -X PUT http://localhost:8080/api/dashboard/rag/document/channels \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "123456789",
    "filename": "outdated_info.txt",
    "channels": []
  }'
```

### Notes
- Updates all chunks of the specified document
- Empty channels array effectively deactivates the document
- Document remains in database but won't be used for any channel

---

## 6. Get RAG Document Details

Get detailed information about RAG documents with various filtering options.

### Endpoint
```
GET /api/dashboard/rag/documents/details
```

### Request

**Headers:**
```
Authorization: Bearer [session-token]
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page_id` | String | Yes | Facebook page ID |
| `channel` | String | No | Filter by channel: "facebook" or "messenger" |
| `filename` | String | No | Filter by specific filename |
| `source` | String | No | Filter by source (e.g., "manual_upload", "crm") |
| `active` | Boolean | No | If true, show only active documents |

### Response

**Success Response (200 OK):**
```json
{
  "documents": [
    {
      "_id": "507f1f77bcf86cd799439011",
      "company_id": "company-001",
      "page_id": "123456789",
      "content": "Document content here...",
      "embedding": [0.123, 0.456, ...],
      "source": "manual_upload",
      "metadata": {
        "filename": "faq.txt",
        "chunk": "1/3",
        "uploaded_by": "admin@company.com",
        "upload_time": "2024-01-15T10:30:00Z"
      },
      "channels": ["facebook", "messenger"],
      "is_active": true,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "groups": [
    {
      "filename": "faq.txt",
      "total_chunks": 3,
      "active_chunks": 3,
      "channels": ["facebook", "messenger"],
      "source": "manual_upload",
      "uploaded_by": "admin@company.com",
      "uploaded_at": "2024-01-15T10:30:00Z",
      "total_size": 12567
    },
    {
      "filename": "policies.md",
      "total_chunks": 1,
      "active_chunks": 0,
      "channels": ["facebook"],
      "source": "manual_upload",
      "uploaded_by": "manager@company.com",
      "uploaded_at": "2024-01-14T09:00:00Z",
      "total_size": 4532
    }
  ],
  "total_documents": 4,
  "total_groups": 2,
  "active_documents": 3,
  "page_id": "123456789",
  "filters": {
    "channel": null,
    "filename": null,
    "source": null,
    "active": false
  }
}
```

### Examples

**Get All Active Documents:**
```bash
curl -X GET "http://localhost:8080/api/dashboard/rag/documents/details?page_id=123456789&active=true" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

**Filter by Channel:**
```bash
curl -X GET "http://localhost:8080/api/dashboard/rag/documents/details?page_id=123456789&channel=messenger" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

**Filter by Filename:**
```bash
curl -X GET "http://localhost:8080/api/dashboard/rag/documents/details?page_id=123456789&filename=faq.txt" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

**Combined Filters:**
```bash
curl -X GET "http://localhost:8080/api/dashboard/rag/documents/details?page_id=123456789&channel=facebook&source=manual_upload&active=true" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

### Notes
- Groups provide aggregated information about documents with the same filename
- The response includes both individual documents and grouped statistics
- Useful for dashboard views and document management interfaces

---

## 7. Toggle Document by ID

Toggle the active status of a specific document by its database ID.

### Endpoint
```
PUT /api/dashboard/rag/document/toggle-by-id
```

### Request

**Headers:**
```
Authorization: Bearer [session-token]
Content-Type: application/json
```

**Body Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `document_id` | String | Yes | Document database ID |
| `is_active` | Boolean | Yes | New active status |

### Response

**Success Response (200 OK):**
```json
{
  "message": "Document status updated successfully",
  "document_id": "507f1f77bcf86cd799439011",
  "is_active": true
}
```

**Error Responses:**

*400 Bad Request:*
```json
{
  "error": "document_id is required"
}
```

*404 Not Found:*
```json
{
  "error": "Document not found"
}
```

### Example

```bash
curl -X PUT http://localhost:8080/api/dashboard/rag/document/toggle-by-id \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "document_id": "507f1f77bcf86cd799439011",
    "is_active": false
  }'
```

### Notes
- This endpoint doesn't require page_id as it uses the document's unique ID
- Only affects the specific document chunk, not other chunks from the same file
- Useful for fine-grained control over individual document chunks

---

## 8. Get Documents by Filename

Retrieve all document chunks associated with a specific filename.

### Endpoint
```
GET /api/dashboard/rag/documents/by-filename
```

### Request

**Headers:**
```
Authorization: Bearer [session-token]
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page_id` | String | Yes | Facebook page ID |
| `filename` | String | Yes | Document filename to retrieve |

### Response

**Success Response (200 OK):**
```json
{
  "filename": "company_faq.txt",
  "chunks": [
    {
      "_id": "507f1f77bcf86cd799439011",
      "company_id": "company-001",
      "page_id": "123456789",
      "content": "FAQ Section 1: General Questions...",
      "source": "manual_upload",
      "metadata": {
        "filename": "company_faq.txt",
        "chunk": "1/3",
        "uploaded_by": "admin@company.com"
      },
      "channels": ["facebook", "messenger"],
      "is_active": true,
      "created_at": "2024-01-15T10:30:00Z"
    },
    {
      "_id": "507f1f77bcf86cd799439012",
      "company_id": "company-001",
      "page_id": "123456789",
      "content": "FAQ Section 2: Product Information...",
      "source": "manual_upload",
      "metadata": {
        "filename": "company_faq.txt",
        "chunk": "2/3",
        "uploaded_by": "admin@company.com"
      },
      "channels": ["facebook", "messenger"],
      "is_active": true,
      "created_at": "2024-01-15T10:30:00Z"
    },
    {
      "_id": "507f1f77bcf86cd799439013",
      "company_id": "company-001",
      "page_id": "123456789",
      "content": "FAQ Section 3: Support & Returns...",
      "source": "manual_upload",
      "metadata": {
        "filename": "company_faq.txt",
        "chunk": "3/3",
        "uploaded_by": "admin@company.com"
      },
      "channels": ["facebook", "messenger"],
      "is_active": true,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "total_chunks": 3,
  "active_chunks": 3,
  "total_size": 8745,
  "channels": ["facebook", "messenger"],
  "combined_content": "FAQ Section 1: General Questions...\n\nFAQ Section 2: Product Information...\n\nFAQ Section 3: Support & Returns...",
  "page_id": "123456789"
}
```

**Error Responses:**

*400 Bad Request:*
```json
{
  "error": "filename is required"
}
```

*404 Not Found:*
```json
{
  "filename": "nonexistent.txt",
  "chunks": [],
  "total_chunks": 0,
  "active_chunks": 0,
  "total_size": 0,
  "channels": [],
  "combined_content": "",
  "page_id": "123456789"
}
```

### Example

```bash
curl -X GET "http://localhost:8080/api/dashboard/rag/documents/by-filename?page_id=123456789&filename=company_faq.txt" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

### Notes
- Returns all chunks in order if the document was split
- `combined_content` provides the full document text reconstructed from all chunks
- Useful for viewing or editing the complete document
- Channel information shows which channels can use this document

---

## Error Handling

### Common Error Codes

| Status Code | Description |
|-------------|-------------|
| 200 | Success |
| 202 | Accepted (for async operations) |
| 400 | Bad Request - Invalid parameters or missing required fields |
| 401 | Unauthorized - Invalid or missing authentication |
| 403 | Forbidden - Access denied to resource |
| 404 | Not Found - Document or page not found |
| 413 | Payload Too Large - File exceeds size limit |
| 500 | Internal Server Error - Server-side error |

### Standard Error Response Format

```json
{
  "error": "Descriptive error message"
}
```

---

## Rate Limiting

- Upload endpoint: Maximum 10 requests per minute per user
- Other endpoints: Maximum 60 requests per minute per user
- Bulk operations should include delays between requests

---

## Best Practices

1. **File Uploads**
   - Keep files under 5MB for optimal processing
   - Use descriptive filenames for easier management
   - Include metadata for better organization

2. **Channel Management**
   - Assign documents to appropriate channels based on content
   - Use Facebook for public information
   - Use Messenger for detailed support content

3. **Document Updates**
   - Delete old versions before uploading new ones
   - Use metadata to track document versions
   - Toggle documents inactive instead of deleting when possible

4. **Performance**
   - List operations may be slow with many documents
   - Use filtering parameters to reduce response size
   - Consider pagination for large document sets

5. **Security**
   - Never upload sensitive information (passwords, credit cards)
   - Regularly audit uploaded documents
   - Use appropriate access controls

---

## Webhook Notifications (Future Enhancement)

*Note: This feature is planned but not yet implemented*

Webhook notifications for document processing completion:
```json
{
  "event": "document.processed",
  "page_id": "123456789",
  "filename": "new_document.txt",
  "chunks_created": 5,
  "status": "success",
  "timestamp": "2024-01-15T10:35:00Z"
}
```
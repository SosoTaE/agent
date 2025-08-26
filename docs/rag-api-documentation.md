# RAG (Retrieval-Augmented Generation) API Documentation

## Overview
These endpoints manage RAG documents for Facebook bot pages. RAG documents are used to provide context and knowledge to the bot for generating relevant responses.

## Authentication
All endpoints require authentication via Bearer token in the Authorization header:
```
Authorization: Bearer <token>
```

---

## 1. Upload RAG Document
Upload a file for RAG processing and vectorization.

**Endpoint:** `POST /dashboard/rag/upload`

### Request
**Headers:**
```
Content-Type: multipart/form-data
Authorization: Bearer <token>
```

**Form Data:**
- `file` (required): The file to upload (PDF, TXT, DOC, DOCX, etc.)
- `company_id` (required): Company identifier
- `page_id` (required): Facebook page identifier
- `source` (optional): Source type (default: "manual")
- `channels` (optional): JSON string of channels map, e.g., `{"facebook": true, "messenger": false}`

### Response
**Success (200 OK):**
```json
{
  "success": true,
  "message": "Document uploaded and processed successfully",
  "data": {
    "document_count": 5,
    "total_chunks": 15,
    "filename": "product_catalog.pdf",
    "source": "manual",
    "channels": {
      "facebook": true,
      "messenger": true
    }
  }
}
```

**Error (400 Bad Request):**
```json
{
  "error": "File is required"
}
```

**Error (500 Internal Server Error):**
```json
{
  "error": "Failed to process document: <error_details>"
}
```

---

## 2. Delete RAG Document
Delete RAG documents by filename or CRM URL.

**Endpoint:** `DELETE /dashboard/rag/document`

### Request
**Query Parameters:**
- `company_id` (required): Company identifier
- `page_id` (required): Facebook page identifier
- `filename` (optional): Name of the file to delete
- `crm_url` (optional): CRM URL of documents to delete

*Note: Either `filename` or `crm_url` must be provided*

### Response
**Success (200 OK):**
```json
{
  "success": true,
  "message": "Documents deleted successfully",
  "deleted_count": 10
}
```

**Error (400 Bad Request):**
```json
{
  "error": "Either filename or crm_url is required"
}
```

---

## 3. List RAG Documents
Retrieve all RAG documents for a specific page.

**Endpoint:** `GET /dashboard/rag/documents`

### Request
**Query Parameters:**
- `company_id` (required): Company identifier
- `page_id` (required): Facebook page identifier
- `source` (optional): Filter by source type
- `is_active` (optional): Filter by active status (true/false)

### Response
**Success (200 OK):**
```json
{
  "success": true,
  "documents": [
    {
      "id": "507f1f77bcf86cd799439011",
      "content": "Product description...",
      "source": "manual",
      "is_active": true,
      "channels": {
        "facebook": true,
        "messenger": false
      },
      "metadata": {
        "filename": "products.pdf",
        "uploaded_by": "admin@example.com",
        "page_number": "1"
      },
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "total": 25,
  "page": 1,
  "limit": 20
}
```

---

## 4. Toggle RAG Document Status
Toggle the active status of RAG documents by filename.

**Endpoint:** `PUT /dashboard/rag/document/toggle`

### Request
**Headers:**
```
Content-Type: application/json
Authorization: Bearer <token>
```

**Body:**
```json
{
  "company_id": "comp_123",
  "page_id": "page_456",
  "filename": "product_catalog.pdf",
  "is_active": true
}
```

### Response
**Success (200 OK):**
```json
{
  "success": true,
  "message": "Document status updated successfully",
  "updated_count": 15,
  "is_active": true
}
```

---

## 5. Update RAG Document Channels
Toggle a specific platform (Facebook/Messenger) on or off for a document.

**Endpoint:** `PUT /dashboard/rag/document/channels`

### Request
**Headers:**
```
Content-Type: application/json
Authorization: Bearer <token>
```

**Body:**
```json
{
  "company_id": "comp_123",
  "page_id": "page_456",
  "filename": "product_catalog.pdf",
  "platform": "facebook",
  "value": true
}
```

**Parameters:**
- `company_id` (required): Company identifier
- `page_id` (required): Facebook page identifier
- `filename` (required): Name of the file to update
- `platform` (required): The platform to update ("facebook" or "messenger")
- `value` (required): Boolean value indicating whether to enable (true) or disable (false) the platform

### Response
**Success (200 OK):**
```json
{
  "message": "Document channel updated successfully",
  "updated_count": 15,
  "platform": "facebook",
  "value": true,
  "filename": "product_catalog.pdf"
}
```

**Error (400 Bad Request):**
```json
{
  "error": "invalid platform: instagram (must be 'facebook' or 'messenger')"
}
```

**Error (404 Not Found):**
```json
{
  "error": "No documents found with the specified filename"
}
```

### Usage Example
To enable a document for Facebook comments only:
1. First call: `{"platform": "facebook", "value": true}`
2. Second call: `{"platform": "messenger", "value": false}`

To enable for both platforms:
1. First call: `{"platform": "facebook", "value": true}`
2. Second call: `{"platform": "messenger", "value": true}`

**Note:** Each call updates only the specified platform without affecting the other platform's status.

---

## 6. Toggle Document Channel (Simplified)
Simple endpoint to toggle a document's platform status using just the document name.

**Endpoint:** `PUT /dashboard/rag/document/toggle-channel`

### Request
**Headers:**
```
Content-Type: application/json
Authorization: Bearer <token>
```

**Body:**
```json
{
  "document_name": "product_catalog.pdf",
  "platform_name": "facebook",
  "value": true
}
```

**Parameters:**
- `document_name` (required): The filename of the document to update
- `platform_name` (required): The platform to toggle ("facebook" or "messenger")
- `value` (required): Boolean value - true to enable, false to disable

### Response
**Success (200 OK):**
```json
{
  "message": "Document channel updated successfully",
  "document_name": "product_catalog.pdf",
  "platform": "facebook",
  "value": true,
  "updated_count": 15,
  "pages_updated": ["page_123", "page_456"]
}
```

**Error (404 Not Found):**
```json
{
  "error": "No documents found with name 'unknown.pdf'"
}
```

**Error (400 Bad Request):**
```json
{
  "error": "invalid platform_name: instagram (must be 'facebook' or 'messenger')"
}
```

### Notes
- This endpoint updates documents across ALL pages the company has access to
- The `pages_updated` field shows which page IDs had documents updated
- Document name must match exactly (case-sensitive)
- Platform names are case-insensitive

### Example Usage
```javascript
// Enable document for Facebook
fetch('/dashboard/rag/document/toggle-channel', {
  method: 'PUT',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <token>'
  },
  body: JSON.stringify({
    document_name: 'faq.pdf',
    platform_name: 'facebook', 
    value: true
  })
});

// Disable document for Messenger
fetch('/dashboard/rag/document/toggle-channel', {
  method: 'PUT',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <token>'
  },
  body: JSON.stringify({
    document_name: 'faq.pdf',
    platform_name: 'messenger',
    value: false
  })
});
```

---

## 7. Get RAG Document Details
Get detailed information about RAG documents, grouped by source.

**Endpoint:** `GET /dashboard/rag/documents/details`

### Request
**Query Parameters:**
- `company_id` (required): Company identifier
- `page_id` (required): Facebook page identifier

### Response
**Success (200 OK):**
```json
{
  "success": true,
  "grouped_documents": [
    {
      "filename": "products.pdf",
      "total_chunks": 15,
      "active_chunks": 12,
      "channels": ["facebook", "messenger"],
      "source": "manual",
      "uploaded_by": "admin@example.com",
      "uploaded_at": "2024-01-15T10:30:00Z",
      "total_size": 45678
    },
    {
      "filename": "faq.txt",
      "total_chunks": 8,
      "active_chunks": 8,
      "channels": ["facebook"],
      "source": "manual",
      "uploaded_by": "user@example.com",
      "uploaded_at": "2024-01-14T15:20:00Z",
      "total_size": 12345
    }
  ],
  "summary": {
    "total_documents": 2,
    "total_chunks": 23,
    "active_chunks": 20,
    "total_size": 58023
  }
}
```

---

## 7. Toggle RAG Document by ID
Toggle the active status of a specific document by its ID.

**Endpoint:** `PUT /dashboard/rag/document/toggle-by-id`

### Request
**Headers:**
```
Content-Type: application/json
Authorization: Bearer <token>
```

**Body:**
```json
{
  "document_id": "507f1f77bcf86cd799439011",
  "is_active": false
}
```

### Response
**Success (200 OK):**
```json
{
  "success": true,
  "message": "Document status updated successfully",
  "document_id": "507f1f77bcf86cd799439011",
  "is_active": false
}
```

**Error (404 Not Found):**
```json
{
  "error": "Document not found"
}
```

---

## 8. Get RAG Documents by Filename
Retrieve all document chunks associated with a specific filename.

**Endpoint:** `GET /dashboard/rag/documents/by-filename`

### Request
**Query Parameters:**
- `company_id` (required): Company identifier
- `page_id` (required): Facebook page identifier
- `filename` (required): The filename to search for

### Response
**Success (200 OK):**
```json
{
  "success": true,
  "filename": "product_catalog.pdf",
  "documents": [
    {
      "id": "507f1f77bcf86cd799439011",
      "content": "Chapter 1: Introduction to our products...",
      "chunk_index": 0,
      "is_active": true,
      "channels": {
        "facebook": true,
        "messenger": true
      },
      "metadata": {
        "page_number": "1",
        "total_pages": "50"
      }
    },
    {
      "id": "507f1f77bcf86cd799439012",
      "content": "Our flagship product line includes...",
      "chunk_index": 1,
      "is_active": true,
      "channels": {
        "facebook": true,
        "messenger": true
      },
      "metadata": {
        "page_number": "2",
        "total_pages": "50"
      }
    }
  ],
  "summary": {
    "total_chunks": 15,
    "active_chunks": 12,
    "total_size": 45678,
    "channels": ["facebook", "messenger"],
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-16T14:20:00Z"
  }
}
```

---

## 9. Get All RAG Files
Get a list of all unique RAG files (not individual chunks) across all pages for the company.

**Endpoint:** `GET /dashboard/rag/files`

### Request
**Query Parameters:**
None required - retrieves all files for the authenticated company.

### Response
**Success (200 OK):**
```json
{
  "files": [
    {
      "name": "product_catalog.pdf",
      "channels": {
        "facebook": true,
        "messenger": false
      }
    },
    {
      "name": "faq.txt",
      "channels": {
        "facebook": true,
        "messenger": true
      }
    },
    {
      "name": "user_manual.pdf",
      "channels": {
        "facebook": false,
        "messenger": false
      }
    }
  ],
  "total_files": 3
}
```

**Fields:**
- `files`: Array of file objects
  - `name`: The filename
  - `channels`: Map showing whether the file is enabled for each platform
    - `facebook`: Boolean indicating if file is active for Facebook comments
    - `messenger`: Boolean indicating if file is active for Messenger
- `total_files`: Total count of unique files

### Notes
- Returns unique filenames only (multiple chunks of the same file are grouped)
- Shows the combined channel status across all chunks
- If any chunk of a file is enabled for a channel, the file shows as enabled for that channel
- Files are retrieved across ALL pages owned by the company
- Only files with filenames in metadata are included (excludes raw text documents without files)

### Example Usage
```javascript
// Get all RAG files for the company
fetch('/dashboard/rag/files', {
  method: 'GET',
  headers: {
    'Authorization': 'Bearer <token>'
  }
})
.then(response => response.json())
.then(data => {
  console.log(`Total files: ${data.total_files}`);
  
  // Filter active files for Facebook
  const facebookFiles = data.files.filter(f => f.channels.facebook);
  console.log(`Files active for Facebook: ${facebookFiles.length}`);
  
  // Find inactive files
  const inactiveFiles = data.files.filter(f => 
    !f.channels.facebook && !f.channels.messenger
  );
  console.log(`Inactive files: ${inactiveFiles.length}`);
});
```

### Use Cases
1. **Dashboard Overview**: Display all uploaded RAG files in a management interface
2. **Bulk Operations**: Get a list of files to perform bulk enable/disable operations
3. **File Management**: Identify which files are active for which channels
4. **Cleanup**: Find inactive files that can be removed

---

## Common Error Responses

### 401 Unauthorized
```json
{
  "error": "Invalid or missing authentication token"
}
```

### 403 Forbidden
```json
{
  "error": "You don't have permission to access this resource"
}
```

### 404 Not Found
```json
{
  "error": "Resource not found"
}
```

### 500 Internal Server Error
```json
{
  "error": "An internal server error occurred"
}
```

---

## Data Types

### Channels Map
The `channels` field is a map indicating which communication channels the document is active for:
```json
{
  "facebook": true,    // Document is used for Facebook comments
  "messenger": false   // Document is not used for Messenger
}
```

### Source Types
- `manual`: Manually uploaded documents
- `crm`: Documents from CRM integration
- `api`: Documents uploaded via API
- `webhook`: Documents from webhook integration

### Metadata Fields
Common metadata fields include:
- `filename`: Original filename
- `uploaded_by`: User who uploaded the document
- `page_number`: Page number (for PDFs)
- `total_pages`: Total pages (for PDFs)
- `chunk_index`: Index of the chunk within the document
- `content_type`: MIME type of the original file

---

## Rate Limiting
- Default rate limit: 100 requests per minute per API key
- Upload endpoints: 10 requests per minute per API key
- Bulk operations may have additional limits

---

## Channel Management System

The RAG system uses a channel-based approach to control where documents are active:

### How Channels Work
Each document can be independently enabled or disabled for:
- **Facebook**: Document is used when responding to Facebook comments
- **Messenger**: Document is used when responding to Messenger messages

### Channel States
Documents maintain a `channels` map with boolean values:
```json
{
  "channels": {
    "facebook": true,   // Document is active for Facebook
    "messenger": false  // Document is inactive for Messenger
  }
}
```

### Updating Channels
To update a document's channel status, use the **Update RAG Document Channels** endpoint with:
1. The platform name (`"facebook"` or `"messenger"`)
2. The desired value (`true` to enable, `false` to disable)

Each update only affects the specified platform, preserving the other platform's status.

### Example Workflow
```javascript
// Enable document for Facebook only
await updateChannel({
  company_id: "comp_123",
  page_id: "page_456",
  filename: "faq.pdf",
  platform: "facebook",
  value: true
});

// Later, also enable for Messenger
await updateChannel({
  company_id: "comp_123",
  page_id: "page_456",
  filename: "faq.pdf",
  platform: "messenger",
  value: true
});

// Disable for Facebook while keeping Messenger active
await updateChannel({
  company_id: "comp_123",
  page_id: "page_456",
  filename: "faq.pdf",
  platform: "facebook",
  value: false
});
```

---

## Best Practices

1. **Chunking**: Large documents are automatically split into chunks for better retrieval. Each chunk is typically 500-1000 tokens.

2. **Channels**: Always specify which channels (Facebook/Messenger) a document should be active for to ensure proper context retrieval. Use the platform-specific toggle to manage channel states.

3. **Metadata**: Include relevant metadata when uploading documents to improve search and organization.

4. **Active Status**: Regularly review and toggle document status based on relevance and accuracy.

5. **Error Handling**: Always check the response status and handle errors appropriately in your client application.

6. **File Formats**: Supported formats include PDF, TXT, DOC, DOCX, RTF, and markdown files. Maximum file size is typically 10MB.

7. **Platform Toggle**: When updating channels, remember that each platform is toggled independently. This allows fine-grained control over document availability across different communication channels.
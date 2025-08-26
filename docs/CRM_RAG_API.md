# CRM and RAG Management API Documentation

## Overview
This document describes the API endpoints for managing CRM links and RAG documents in the Facebook Bot system. These endpoints allow you to add, toggle, and manage CRM links and RAG documents with channel-specific configurations.

## Authentication
All endpoints require authentication via session cookie.

## Base URL
```
http://localhost:3000/api/dashboard
```

---

## CRM Link Management

### 1. Get CRM Links for Channel
**Endpoint:** `GET /crm-links`

**Query Parameters:**
- `page_id` (required): Facebook page ID
- `channel` (optional): "facebook" or "messenger"

**Example Request:**
```bash
curl -X GET "http://localhost:3000/api/dashboard/crm-links?page_id=461998383671026&channel=facebook" \
  -H "Cookie: session=your_session_cookie"
```

**Response:**
```json
{
  "crm_links": [
    {
      "name": "Product API",
      "url": "https://api.example.com/products",
      "type": "api",
      "is_active": true,
      "api_key": "sk-xxx",
      "headers": {},
      "description": "Product catalog API"
    }
  ],
  "page_id": "461998383671026",
  "channel": "facebook",
  "total": 1,
  "active_count": 1,
  "inactive_count": 0
}
```

### 2. Add CRM Link
**Endpoint:** `POST /crm-links`

**Request Body:**
```json
{
  "page_id": "461998383671026",
  "channels": ["facebook", "messenger"],
  "name": "Product API",
  "url": "https://api.example.com/products",
  "type": "api",
  "api_key": "sk-xxx",
  "headers": {
    "X-Custom-Header": "value"
  },
  "description": "Product catalog API",
  "is_active": true
}
```

**Example Request:**
```bash
curl -X POST "http://localhost:3000/api/dashboard/crm-links" \
  -H "Cookie: session=your_session_cookie" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "461998383671026",
    "channels": ["facebook", "messenger"],
    "name": "Product API",
    "url": "https://api.example.com/products",
    "type": "api",
    "is_active": true
  }'
```

**Response:**
```json
{
  "message": "CRM link added successfully",
  "crm_link": {
    "name": "Product API",
    "url": "https://api.example.com/products",
    "type": "api",
    "is_active": true
  },
  "channels": ["facebook", "messenger"],
  "page_id": "461998383671026"
}
```

### 3. Update CRM Link Status
**Endpoint:** `PUT /crm-links/status`

**Request Body:**
```json
{
  "page_id": "461998383671026",
  "url": "https://api.example.com/products",
  "channels": ["facebook", "messenger"],
  "is_active": false
}
```

**Example Request:**
```bash
curl -X PUT "http://localhost:3000/api/dashboard/crm-links/status" \
  -H "Cookie: session=your_session_cookie" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "461998383671026",
    "url": "https://api.example.com/products",
    "channels": ["facebook"],
    "is_active": false
  }'
```

**Response:**
```json
{
  "message": "CRM link status updated successfully",
  "updated_count": 1,
  "url": "https://api.example.com/products",
  "channels": ["facebook"],
  "is_active": false
}
```

### 4. Delete CRM Link
**Endpoint:** `DELETE /crm-links`

**Request Body:**
```json
{
  "page_id": "461998383671026",
  "url": "https://api.example.com/products",
  "channels": ["facebook", "messenger"]
}
```

**Example Request:**
```bash
curl -X DELETE "http://localhost:3000/api/dashboard/crm-links" \
  -H "Cookie: session=your_session_cookie" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "461998383671026",
    "url": "https://api.example.com/products",
    "channels": ["facebook", "messenger"]
  }'
```

**Response:**
```json
{
  "message": "CRM link deleted successfully",
  "deleted_count": 2,
  "url": "https://api.example.com/products",
  "channels": ["facebook", "messenger"]
}
```

---

## RAG Document Management

### 1. Get RAG Document Details
**Endpoint:** `GET /rag/documents/details`

**Query Parameters:**
- `page_id` (required): Facebook page ID
- `channel` (optional): Filter by channel
- `filename` (optional): Filter by filename
- `source` (optional): Filter by source
- `active` (optional): "true" to show only active documents

**Example Request:**
```bash
curl -X GET "http://localhost:3000/api/dashboard/rag/documents/details?page_id=461998383671026&channel=facebook&active=true" \
  -H "Cookie: session=your_session_cookie"
```

**Response:**
```json
{
  "documents": [
    {
      "_id": "507f1f77bcf86cd799439011",
      "content": "Product information...",
      "channels": ["facebook", "messenger"],
      "source": "manual_upload",
      "is_active": true,
      "metadata": {
        "filename": "products.pdf",
        "uploaded_by": "admin@company.com"
      },
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "groups": [
    {
      "filename": "products.pdf",
      "total_chunks": 5,
      "active_chunks": 5,
      "channels": ["facebook", "messenger"],
      "source": "manual_upload",
      "uploaded_by": "admin@company.com",
      "uploaded_at": "2024-01-15T10:30:00Z",
      "total_size": 12500
    }
  ],
  "total_documents": 5,
  "total_groups": 1,
  "active_documents": 5,
  "page_id": "461998383671026",
  "filters": {
    "channel": "facebook",
    "filename": null,
    "source": null,
    "active": true
  }
}
```

### 2. Toggle RAG Document by ID
**Endpoint:** `PUT /rag/document/toggle-by-id`

**Request Body:**
```json
{
  "document_id": "507f1f77bcf86cd799439011",
  "is_active": false
}
```

**Example Request:**
```bash
curl -X PUT "http://localhost:3000/api/dashboard/rag/document/toggle-by-id" \
  -H "Cookie: session=your_session_cookie" \
  -H "Content-Type: application/json" \
  -d '{
    "document_id": "507f1f77bcf86cd799439011",
    "is_active": false
  }'
```

**Response:**
```json
{
  "message": "Document status updated successfully",
  "document_id": "507f1f77bcf86cd799439011",
  "is_active": false
}
```

### 3. Get RAG Documents by Filename
**Endpoint:** `GET /rag/documents/by-filename`

**Query Parameters:**
- `page_id` (required): Facebook page ID
- `filename` (required): Document filename

**Example Request:**
```bash
curl -X GET "http://localhost:3000/api/dashboard/rag/documents/by-filename?page_id=461998383671026&filename=products.pdf" \
  -H "Cookie: session=your_session_cookie"
```

**Response:**
```json
{
  "filename": "products.pdf",
  "chunks": [
    {
      "_id": "507f1f77bcf86cd799439011",
      "content": "Product catalog chunk 1...",
      "channels": ["facebook", "messenger"],
      "is_active": true,
      "metadata": {
        "filename": "products.pdf",
        "chunk": "1/5"
      }
    },
    {
      "_id": "507f1f77bcf86cd799439012",
      "content": "Product catalog chunk 2...",
      "channels": ["facebook", "messenger"],
      "is_active": true,
      "metadata": {
        "filename": "products.pdf",
        "chunk": "2/5"
      }
    }
  ],
  "total_chunks": 5,
  "active_chunks": 5,
  "total_size": 12500,
  "channels": ["facebook", "messenger"],
  "combined_content": "Product catalog chunk 1...\n\nProduct catalog chunk 2...\n\n...",
  "page_id": "461998383671026"
}
```

### 4. Update RAG Document Channels
**Endpoint:** `PUT /rag/document/channels`

**Request Body:**
```json
{
  "page_id": "461998383671026",
  "filename": "products.pdf",
  "channels": ["messenger"]
}
```

**Example Request:**
```bash
curl -X PUT "http://localhost:3000/api/dashboard/rag/document/channels" \
  -H "Cookie: session=your_session_cookie" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "461998383671026",
    "filename": "products.pdf",
    "channels": ["messenger"]
  }'
```

**Response:**
```json
{
  "message": "Document channels updated successfully",
  "updated_count": 5,
  "channels": ["messenger"],
  "document_active": true
}
```

**To disable a document for all channels:**
```json
{
  "page_id": "461998383671026",
  "filename": "products.pdf",
  "channels": []
}
```

---

## Channel Management

### Channel Array Concept
The `channels` field is an array that determines which channels a CRM link or RAG document is active for:

- `["facebook", "messenger"]` - Active for both channels
- `["facebook"]` - Active only for Facebook comments
- `["messenger"]` - Active only for Messenger
- `[]` - Disabled for all channels

### Examples

#### Enable CRM link for both channels:
```bash
curl -X POST "http://localhost:3000/api/dashboard/crm-links" \
  -H "Cookie: session=your_session_cookie" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "461998383671026",
    "channels": ["facebook", "messenger"],
    "name": "Shared API",
    "url": "https://api.example.com/shared",
    "type": "api",
    "is_active": true
  }'
```

#### Disable RAG document temporarily:
```bash
curl -X PUT "http://localhost:3000/api/dashboard/rag/document/channels" \
  -H "Cookie: session=your_session_cookie" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "461998383671026",
    "filename": "outdated.pdf",
    "channels": []
  }'
```

#### Move CRM link from Facebook to Messenger only:
```bash
curl -X PUT "http://localhost:3000/api/dashboard/crm-links/status" \
  -H "Cookie: session=your_session_cookie" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "461998383671026",
    "url": "https://api.example.com/sensitive",
    "channels": ["messenger"],
    "is_active": true
  }'
```

---

## Error Responses

All endpoints return appropriate HTTP status codes:

- `200 OK` - Successful GET request
- `201 Created` - Successfully created resource
- `400 Bad Request` - Invalid request parameters
- `401 Unauthorized` - Missing or invalid session
- `403 Forbidden` - Access denied
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

Error response format:
```json
{
  "error": "Detailed error message"
}
```

---

## Testing

### Test CRM Link Flow
```bash
# 1. Add CRM link
curl -X POST "http://localhost:3000/api/dashboard/crm-links" \
  -H "Cookie: session=your_session" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "461998383671026",
    "channels": ["facebook", "messenger"],
    "name": "Test API",
    "url": "https://api.test.com/data",
    "type": "api",
    "is_active": true
  }'

# 2. Get CRM links
curl -X GET "http://localhost:3000/api/dashboard/crm-links?page_id=461998383671026&channel=facebook" \
  -H "Cookie: session=your_session"

# 3. Toggle status
curl -X PUT "http://localhost:3000/api/dashboard/crm-links/status" \
  -H "Cookie: session=your_session" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "461998383671026",
    "url": "https://api.test.com/data",
    "channels": ["facebook"],
    "is_active": false
  }'

# 4. Delete CRM link
curl -X DELETE "http://localhost:3000/api/dashboard/crm-links" \
  -H "Cookie: session=your_session" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "461998383671026",
    "url": "https://api.test.com/data",
    "channels": ["facebook", "messenger"]
  }'
```

### Test RAG Document Flow
```bash
# 1. Upload document (existing endpoint)
curl -X POST "http://localhost:3000/api/dashboard/rag/upload" \
  -H "Cookie: session=your_session" \
  -F "file=@test.txt" \
  -F "page_id=461998383671026" \
  -F "channels=facebook,messenger"

# 2. Get document details
curl -X GET "http://localhost:3000/api/dashboard/rag/documents/details?page_id=461998383671026" \
  -H "Cookie: session=your_session"

# 3. Get by filename
curl -X GET "http://localhost:3000/api/dashboard/rag/documents/by-filename?page_id=461998383671026&filename=test.txt" \
  -H "Cookie: session=your_session"

# 4. Update channels
curl -X PUT "http://localhost:3000/api/dashboard/rag/document/channels" \
  -H "Cookie: session=your_session" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "461998383671026",
    "filename": "test.txt",
    "channels": ["messenger"]
  }'

# 5. Toggle by ID
curl -X PUT "http://localhost:3000/api/dashboard/rag/document/toggle-by-id" \
  -H "Cookie: session=your_session" \
  -H "Content-Type: application/json" \
  -d '{
    "document_id": "507f1f77bcf86cd799439011",
    "is_active": false
  }'
```
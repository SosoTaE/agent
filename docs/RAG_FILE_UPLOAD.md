# RAG File Upload Documentation

## Overview
The RAG (Retrieval-Augmented Generation) file upload feature allows users to upload documents that will be processed, embedded, and stored in the vector database. These documents can then be used to provide contextual information when the bot responds to messages.

## Features

- Upload text-based files (.txt, .md, .csv, .json)
- Automatic text extraction and chunking for large files
- Embedding generation using Voyage AI
- Storage in MongoDB vector database
- Document management (list, delete)
- Metadata tracking (filename, upload time, uploader)

## API Endpoints

### 1. Upload RAG Document
**POST** `/api/dashboard/rag/upload`

Uploads a file for RAG processing.

**Headers:**
- `Authorization: Bearer <token>` (required)
- `Content-Type: multipart/form-data`

**Form Data:**
- `file` (required): The file to upload (max 10MB)
- `page_id` (required): Facebook page ID
- `source` (optional): Source identifier (default: "manual_upload")
- `metadata` (optional): Comma-separated key=value pairs (e.g., "category=products,type=catalog")

**Response:**
```json
{
  "message": "File processed successfully",
  "filename": "product_catalog.txt",
  "chunks_total": 5,
  "chunks_stored": 5,
  "source": "manual_upload",
  "page_id": "123456789"
}
```

**Error Response:**
```json
{
  "error": "File size exceeds 10MB limit"
}
```

### 2. List RAG Documents
**GET** `/api/dashboard/rag/documents?page_id=<page_id>`

Lists all RAG documents for a specific page.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Query Parameters:**
- `page_id` (required): Facebook page ID

**Response:**
```json
{
  "documents": [
    {
      "id": "507f1f77bcf86cd799439011",
      "content": "Product description text...",
      "source": "manual_upload",
      "is_active": true,
      "created_at": "2025-01-19T10:00:00Z",
      "metadata": {
        "filename": "products.txt",
        "upload_time": "2025-01-19T10:00:00Z",
        "uploaded_by": "admin@example.com",
        "chunk": "1/3"
      }
    }
  ],
  "document_groups": {
    "products.txt": [
      {
        "id": "507f1f77bcf86cd799439011",
        "content": "Chunk 1 content...",
        "source": "manual_upload",
        "is_active": true,
        "created_at": "2025-01-19T10:00:00Z",
        "metadata": {...}
      },
      {
        "id": "507f1f77bcf86cd799439012",
        "content": "Chunk 2 content...",
        "source": "manual_upload",
        "is_active": true,
        "created_at": "2025-01-19T10:00:00Z",
        "metadata": {...}
      }
    ]
  },
  "total": 6
}
```

### 3. Toggle RAG Document Status
**PUT** `/api/dashboard/rag/document/toggle`

Toggles the active/inactive status of RAG documents.

**Headers:**
- `Authorization: Bearer <token>` (required)
- `Content-Type: application/json`

**Request Body:**
```json
{
  "page_id": "123456789",
  "filename": "products.txt",  // Toggle all chunks with this filename
  // OR
  "document_id": "507f1f77bcf86cd799439011", // Toggle specific document
  "is_active": false  // Set active status
}
```

**Response:**
```json
{
  "message": "Successfully updated 3 document(s)",
  "updated": 3,
  "is_active": false
}
```

### 4. Delete RAG Document
**DELETE** `/api/dashboard/rag/document`

Deletes RAG documents by filename or content.

**Headers:**
- `Authorization: Bearer <token>` (required)
- `Content-Type: application/json`

**Request Body:**
```json
{
  "page_id": "123456789",
  "filename": "products.txt"  // OR "content": "exact content to delete"
}
```

**Response:**
```json
{
  "message": "Successfully deleted 3 document(s)",
  "deleted": 3
}
```

## File Processing

### Supported File Types
- `.txt` - Plain text files
- `.md` - Markdown files  
- `.csv` - CSV files (processed as text)
- `.json` - JSON files (processed as text)

### Text Chunking
Large files are automatically split into chunks to optimize:
- Embedding generation efficiency
- Search relevance
- Token usage

**Chunk Size:** 2000 characters per chunk

**Chunking Strategy:**
- Splits on word boundaries to maintain context
- Preserves complete sentences when possible
- Each chunk includes metadata about its position (e.g., "chunk": "2/5")

### Embedding Generation
- Uses Voyage AI API configured for the page
- Falls back to mock embeddings if Voyage API is not configured
- Each chunk gets its own embedding vector
- Embeddings are stored alongside the text content

## Usage Examples

### Upload a Product Catalog
```bash
curl -X POST https://your-api.com/api/dashboard/rag/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@product_catalog.txt" \
  -F "page_id=123456789" \
  -F "source=product_catalog" \
  -F "metadata=category=products,version=2025"
```

### List Documents for a Page
```bash
curl -X GET "https://your-api.com/api/dashboard/rag/documents?page_id=123456789" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Toggle Document Status (Deactivate)
```bash
curl -X PUT https://your-api.com/api/dashboard/rag/document/toggle \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "123456789",
    "filename": "old_catalog.txt",
    "is_active": false
  }'
```

### Toggle Document Status (Reactivate)
```bash
curl -X PUT https://your-api.com/api/dashboard/rag/document/toggle \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "123456789",
    "filename": "old_catalog.txt",
    "is_active": true
  }'
```

### Delete a Document
```bash
curl -X DELETE https://your-api.com/api/dashboard/rag/document \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "page_id": "123456789",
    "filename": "old_catalog.txt"
  }'
```

## Integration with Bot Responses

When a customer sends a message:
1. The system checks if there are active CRM/RAG documents using `HasActiveCRMDocuments()`
2. If documents exist, it searches for relevant content using `GetRAGContext()`
3. The bot uses the retrieved context to provide informed responses

### How RAG Context is Used
- The bot searches for similar documents based on the customer's message
- Top relevant chunks are retrieved using cosine similarity
- The context is passed to Claude AI along with the customer's message
- Claude generates a response incorporating the relevant information

## Best Practices

### File Preparation
1. **Clean Text**: Remove unnecessary formatting, headers, footers
2. **Structured Content**: Use clear sections and headings
3. **Relevant Information**: Only upload content relevant to customer queries
4. **Regular Updates**: Delete old documents before uploading updated versions

### Metadata Usage
Use metadata to categorize and organize documents:
```
metadata=category=faq,language=en,version=2025-01
```

### Document Management
1. **Regular Cleanup**: Remove outdated documents
2. **Version Control**: Use metadata to track document versions
3. **Page-Specific**: Upload documents to the correct page
4. **Active Status**: Documents are active by default
5. **Toggle Status**: Deactivate documents temporarily instead of deleting when testing
6. **Batch Toggle**: Use filename to toggle all chunks at once

## Security Considerations

1. **Authentication Required**: All endpoints require valid JWT token
2. **Page Ownership**: Users can only manage documents for their pages
3. **File Size Limit**: 10MB maximum to prevent abuse
4. **Content Validation**: Files are validated before processing
5. **Company Isolation**: Documents are isolated by company and page

## Troubleshooting

### Common Issues

1. **"No text content found in file"**
   - Ensure file contains readable text
   - Check file encoding (UTF-8 recommended)

2. **"Failed to generate embeddings"**
   - Verify Voyage API key is configured
   - Check API quota limits

3. **"File size exceeds limit"**
   - Split large files before uploading
   - Compress content if possible

4. **Documents not appearing in bot context**
   - Verify documents are marked as active
   - Check if embeddings were generated successfully
   - Ensure correct page_id is used

## Database Schema

Documents are stored in the `vector_documents` collection:

```javascript
{
  _id: ObjectId,
  company_id: String,
  page_id: String,
  content: String,
  embedding: Array[Float32],
  metadata: {
    filename: String,
    upload_time: String,
    uploaded_by: String,
    chunk: String,  // "1/3" format if chunked
    ...custom fields
  },
  source: String,
  is_active: Boolean,
  created_at: Date,
  updated_at: Date
}
```

## Future Enhancements

- Support for PDF files
- Support for Word documents (.docx)
- Support for Excel files (.xlsx)
- Automatic OCR for image-based PDFs
- Bulk upload functionality
- Scheduled document updates
- Document versioning system
- Advanced chunking strategies
- Multi-language support
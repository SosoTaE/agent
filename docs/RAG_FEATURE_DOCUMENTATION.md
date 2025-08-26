# RAG Document Management Feature Documentation

## Overview
The RAG (Retrieval-Augmented Generation) Document Management feature enables uploading, processing, and managing documents that serve as a knowledge base for the AI chatbot. The system processes documents into vector embeddings for efficient semantic search and retrieval.

## Architecture

### Components
1. **File Upload Handler** - Accepts document uploads via HTTP
2. **Background Processor** - Asynchronously chunks and processes documents
3. **Vector Database** - Stores document embeddings with metadata
4. **Claude AI Integration** - Uses RAG context for generating responses

### Data Flow
```
Upload → Validation → Background Processing → Chunking → Embedding → Vector Storage → Retrieval
```

## API Endpoints

### 1. Upload RAG Document
**Endpoint:** `POST /api/dashboard/rag/upload`

**Purpose:** Upload a document for RAG processing

**Request:**
- Method: `POST`
- Content-Type: `multipart/form-data`
- Authentication: Required

**Form Fields:**
- `file` (required): The document file to upload
- `page_id` (required): Facebook page ID
- `source` (optional): Source identifier (default: "manual_upload")

**Response:**
```json
{
  "message": "File received and being processed",
  "filename": "document.pdf",
  "company_id": "comp123",
  "page_id": "461998383671026"
}
```

**Processing Steps:**
1. File validation (size, type)
2. Content extraction
3. Background processing initiated
4. Document chunking (1000 chars with 200 char overlap)
5. Embedding generation via Voyage AI
6. Storage in MongoDB vector collection

**Supported File Types:**
- Text files (.txt)
- PDF documents (.pdf)
- Word documents (.docx)
- Markdown files (.md)

### 2. Delete RAG Document
**Endpoint:** `DELETE /api/dashboard/rag/document`

**Purpose:** Delete a RAG document from the vector database

**Request:**
```json
{
  "company_id": "comp123",
  "page_id": "461998383671026",
  "filename": "document.pdf"
}
```

**Response:**
```json
{
  "message": "Document deleted successfully",
  "deleted_count": 5
}
```

### 3. List RAG Documents
**Endpoint:** `GET /api/dashboard/rag/documents?page_id={page_id}`

**Purpose:** List all RAG documents for a specific page

**Response:**
```json
{
  "documents": [
    {
      "_id": "507f1f77bcf86cd799439011",
      "content": "Document content preview...",
      "metadata": {
        "filename": "product_catalog.pdf",
        "upload_date": "2024-01-15T10:30:00Z",
        "uploaded_by": "admin@company.com",
        "chunk_index": 0,
        "total_chunks": 5
      },
      "company_id": "comp123",
      "page_id": "461998383671026",
      "source": "manual_upload",
      "is_active": true,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "total": 15,
  "active": 12,
  "inactive": 3
}
```

### 4. Toggle RAG Document Status
**Endpoint:** `PUT /api/dashboard/rag/document/toggle`

**Purpose:** Enable or disable a RAG document without deleting it

**Request:**
```json
{
  "company_id": "comp123",
  "page_id": "461998383671026",
  "filename": "document.pdf",
  "is_active": false
}
```

**Response:**
```json
{
  "message": "Document status updated successfully",
  "updated_count": 5,
  "is_active": false
}
```

## Database Schema

### Vector Documents Collection
```javascript
{
  _id: ObjectId,
  content: String,           // Document chunk content
  embedding: [Float],        // 1024-dimensional vector from Voyage AI
  metadata: {
    filename: String,        // Original filename
    upload_date: Date,       // When uploaded
    uploaded_by: String,     // User email who uploaded
    chunk_index: Number,     // Index of this chunk
    total_chunks: Number,    // Total chunks for this document
    additional_info: Object  // Any extra metadata
  },
  company_id: String,        // Company identifier
  page_id: String,          // Facebook page ID
  source: String,           // Source type (manual_upload, crm, etc.)
  is_active: Boolean,       // Whether document is active
  created_at: Date,         // Creation timestamp
  updated_at: Date          // Last update timestamp
}
```

## Integration with AI Responses

### Context Retrieval
When processing messages, the system:
1. Checks for active RAG documents: `HasActiveCRMDocuments()`
2. Retrieves relevant context: `GetRAGContext(message, companyID, pageID)`
3. Performs semantic search using vector similarity
4. Returns top 5 most relevant chunks
5. Passes context to Claude AI with strict instructions

### System Prompt Integration
The AI receives structured input:
```xml
<context>
[RAG document content here]
</context>

<chatHistory>
[Previous conversation]
</chatHistory>

<question>
[Customer question]
</question>
```

### Strict RAG Boundaries
The system enforces strict knowledge boundaries:
- AI must ONLY answer questions covered in the RAG data
- For any question outside the knowledge base, AI responds: "I can only help with [topics from knowledge base]"
- Explicit restrictions on general knowledge, weather, math, news, etc.

## Configuration

### Environment Variables
```bash
VOYAGE_API_KEY=your_voyage_api_key       # For embedding generation
ANTHROPIC_API_KEY=your_claude_api_key    # For AI responses
MONGODB_URI=mongodb://localhost:27017     # Database connection
```

### File Processing Settings
```go
const (
    maxFileSize = 10 * 1024 * 1024  // 10MB max file size
    chunkSize = 1000                 // Characters per chunk
    chunkOverlap = 200               // Overlap between chunks
    embeddingDim = 1024              // Voyage embedding dimensions
)
```

## Error Handling

### Common Errors
1. **File Too Large**: Files over 10MB are rejected
2. **Unsupported Format**: Only text-based formats supported
3. **Processing Failure**: Background job failures logged but don't block response
4. **Embedding API Error**: Retries with exponential backoff

### Error Responses
```json
{
  "error": "File size exceeds maximum allowed size of 10MB"
}
```

## Security Considerations

1. **Authentication**: All endpoints require authenticated session
2. **Company Isolation**: Documents filtered by company_id and page_id
3. **File Validation**: Type and size validation before processing
4. **Content Sanitization**: Removal of potentially harmful content
5. **Access Control**: Users can only manage their company's documents

## Performance Optimization

1. **Asynchronous Processing**: Documents processed in background
2. **Chunking Strategy**: Optimal chunk size for semantic search
3. **Vector Indexing**: MongoDB Atlas vector search indexes
4. **Caching**: Frequently accessed documents cached
5. **Batch Operations**: Multiple chunks processed together

## Usage Example

### Uploading a Product Catalog
```bash
curl -X POST http://localhost:3000/api/dashboard/rag/upload \
  -H "Cookie: session=your_session_cookie" \
  -F "file=@product_catalog.pdf" \
  -F "page_id=461998383671026" \
  -F "source=product_catalog"
```

### Checking Upload Status
```bash
curl http://localhost:3000/api/dashboard/rag/documents?page_id=461998383671026 \
  -H "Cookie: session=your_session_cookie"
```

### Disabling a Document
```bash
curl -X PUT http://localhost:3000/api/dashboard/rag/document/toggle \
  -H "Content-Type: application/json" \
  -H "Cookie: session=your_session_cookie" \
  -d '{
    "company_id": "comp123",
    "page_id": "461998383671026", 
    "filename": "old_catalog.pdf",
    "is_active": false
  }'
```

## Monitoring and Debugging

### Logs to Monitor
```go
// Successful upload
slog.Info("Processing RAG document in background",
    "filename", filename,
    "company_id", companyID,
    "chunks", len(chunks))

// Embedding generation
slog.Info("Generated embeddings for chunk",
    "chunk_index", i,
    "embedding_dim", len(embedding))

// Retrieval usage
slog.Info("RAG context retrieved",
    "contextLength", len(ragContext),
    "companyID", companyID,
    "pageID", pageID)
```

### Debug Environment Variables
```bash
DEBUG_CLAUDE=true  # Shows all Claude AI interactions
DEBUG_RAG=true     # Shows RAG processing details
```

## Best Practices

1. **Document Preparation**
   - Use clear, structured content
   - Include specific product/service details
   - Avoid contradictory information
   - Update regularly to maintain accuracy

2. **Chunk Size Optimization**
   - Adjust chunk size based on content type
   - Ensure meaningful context in each chunk
   - Test retrieval quality with sample queries

3. **Active Management**
   - Regularly review document list
   - Disable outdated documents
   - Monitor which documents are retrieved most

4. **Testing**
   - Test with various query types
   - Verify correct document retrieval
   - Ensure AI stays within knowledge boundaries

## Troubleshooting

### Issue: Documents Not Being Retrieved
**Solution:** Check if documents are active and properly indexed

### Issue: AI Answering Outside Knowledge Base
**Solution:** Verify system prompts are properly configured with strict boundaries

### Issue: Slow Document Processing
**Solution:** Check Voyage AI API limits and MongoDB performance

### Issue: Upload Fails Silently
**Solution:** Check background job logs for processing errors

## Future Enhancements

1. **Document Versioning**: Track changes over time
2. **Automatic Updates**: Sync with external data sources
3. **Multi-language Support**: Process documents in various languages
4. **Advanced Chunking**: Intelligent semantic chunking
5. **Analytics Dashboard**: Track document usage and effectiveness
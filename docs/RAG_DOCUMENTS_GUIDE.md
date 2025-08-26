# RAG Documents Guide

## Overview

The RAG (Retrieval-Augmented Generation) Documents feature enables the Facebook Bot Management System to incorporate custom knowledge from uploaded documents into AI responses. This allows businesses to provide accurate, context-specific information based on their documentation, policies, FAQs, and other text-based resources.

## Key Features

- **Multi-Format Support**: Upload TXT, MD, CSV, and JSON files
- **Channel-Specific Activation**: Documents can be active for Facebook, Messenger, or both
- **Intelligent Chunking**: Automatic splitting of large documents
- **Semantic Search**: Vector embeddings for accurate information retrieval
- **Real-time Toggle**: Enable/disable documents without deletion
- **Metadata Tracking**: Track uploader, timestamp, and document properties
- **Background Processing**: Asynchronous document processing for better performance

## Architecture

### Data Flow
```
Document Upload → Text Extraction → Chunking → Embedding Generation → Vector Storage
                                                          ↓
                                              Semantic Search during AI Response
```

### Storage Structure
- Document metadata stored in MongoDB `vector_documents` collection
- Each document chunk stored as separate embedding
- Chunks linked by filename for grouped management
- Channel assignment determines usage context

## Supported File Types

| Extension | Type | Max Size | Use Case |
|-----------|------|----------|----------|
| .txt | Plain Text | 10MB | General documentation, policies |
| .md | Markdown | 10MB | Technical docs, READMEs |
| .csv | Comma-Separated | 10MB | Product catalogs, data tables |
| .json | JSON | 10MB | Structured data, configurations |

## API Endpoints

### 1. Upload RAG Document
```http
POST /api/dashboard/rag/upload
```

**Form Data:**
- `file` (required): The document file
- `page_id` (required): Facebook page ID
- `channels` (required): Comma-separated channels ("facebook,messenger")
- `source` (optional): Source identifier (default: "manual_upload")
- `metadata` (optional): Key-value pairs (format: "key1=value1,key2=value2")

**Example Request:**
```bash
curl -X POST http://localhost:8080/api/dashboard/rag/upload \
  -H "Authorization: Bearer [token]" \
  -F "file=@company_faq.txt" \
  -F "page_id=123456789" \
  -F "channels=facebook,messenger" \
  -F "source=faq" \
  -F "metadata=category=support,version=2.0"
```

**Response:**
```json
{
  "message": "File received and queued for processing",
  "filename": "company_faq.txt",
  "size": 45678,
  "source": "faq",
  "page_id": "123456789",
  "channels": ["facebook", "messenger"],
  "status": "processing"
}
```

### 2. List RAG Documents
```http
GET /api/dashboard/rag/documents?page_id={pageID}
```

**Query Parameters:**
- `page_id` (required): Facebook page ID

**Response:**
```json
{
  "documents": [
    {
      "id": "507f1f77bcf86cd799439011",
      "content": "Document chunk content...",
      "source": "manual_upload",
      "is_active": true,
      "created_at": "2024-01-15T10:30:00Z",
      "metadata": {
        "filename": "company_faq.txt",
        "chunk": "1/5",
        "uploaded_by": "admin@company.com"
      }
    }
  ],
  "document_groups": {
    "company_faq.txt": [
      /* Array of document chunks */
    ]
  },
  "total": 5
}
```

### 3. Toggle Document Status
```http
PUT /api/dashboard/rag/document/toggle
```

**Request Body:**
```json
{
  "page_id": "123456789",
  "filename": "company_faq.txt",
  "is_active": false
}
```

**Response:**
```json
{
  "message": "Successfully updated 5 document(s)",
  "updated": 5,
  "is_active": false
}
```

### 4. Toggle Document by ID
```http
PUT /api/dashboard/rag/document/toggle-by-id
```

**Request Body:**
```json
{
  "document_id": "507f1f77bcf86cd799439011",
  "is_active": true
}
```

### 5. Update Document Channels
```http
PUT /api/dashboard/rag/document/channels
```

**Request Body:**
```json
{
  "page_id": "123456789",
  "filename": "company_faq.txt",
  "channels": ["messenger"]
}
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

### 6. Delete RAG Document
```http
DELETE /api/dashboard/rag/document
```

**Request Body:**
```json
{
  "page_id": "123456789",
  "filename": "company_faq.txt"
}
```

**Alternative - Delete by Content:**
```json
{
  "page_id": "123456789",
  "content": "Specific content to delete"
}
```

**Response:**
```json
{
  "message": "Successfully deleted 5 document(s)",
  "deleted": 5
}
```

### 7. Get Document Details
```http
GET /api/dashboard/rag/documents/details?page_id={pageID}&channel={channel}&filename={filename}&active={true}
```

**Query Parameters:**
- `page_id` (required): Facebook page ID
- `channel` (optional): Filter by channel
- `filename` (optional): Filter by filename
- `source` (optional): Filter by source
- `active` (optional): Filter active documents only

**Response:**
```json
{
  "documents": [ /* Document array */ ],
  "groups": [
    {
      "filename": "company_faq.txt",
      "total_chunks": 5,
      "active_chunks": 5,
      "channels": ["facebook", "messenger"],
      "source": "manual_upload",
      "uploaded_by": "admin@company.com",
      "uploaded_at": "2024-01-15T10:30:00Z",
      "total_size": 45678
    }
  ],
  "total_documents": 5,
  "total_groups": 1,
  "active_documents": 5,
  "page_id": "123456789",
  "filters": {
    "channel": null,
    "filename": null,
    "source": null,
    "active": false
  }
}
```

### 8. Get Documents by Filename
```http
GET /api/dashboard/rag/documents/by-filename?page_id={pageID}&filename={filename}
```

**Response:**
```json
{
  "filename": "company_faq.txt",
  "chunks": [ /* Array of document chunks */ ],
  "total_chunks": 5,
  "active_chunks": 5,
  "total_size": 45678,
  "channels": ["facebook", "messenger"],
  "combined_content": "Full document content...",
  "page_id": "123456789"
}
```

### 9. Get All RAG Documents (All Pages)
```http
GET /api/dashboard/rag/documents/all
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
      "documents": [ /* Document array */ ],
      "document_groups": { /* Grouped by filename */ },
      "total_documents": 15,
      "active_documents": 12,
      "inactive_documents": 3,
      "total_size": 234567,
      "unique_files": 3
    }
  ],
  "total_pages": 1,
  "total_documents": 15,
  "total_active": 12,
  "total_inactive": 3,
  "total_size": 234567,
  "size_mb": 0.22
}
```

## Document Processing

### 1. Upload Process
1. **File Validation**: Check file type and size
2. **Text Extraction**: Extract text content from file
3. **Background Processing**: Queue for asynchronous processing
4. **Chunking**: Split into manageable chunks (2000 chars)
5. **Embedding Generation**: Create vector embeddings
6. **Storage**: Save to vector database with metadata

### 2. Chunking Strategy

Documents are automatically split into chunks when they exceed 2000 characters:

```
Original Document (8000 chars)
    ↓
Chunk 1 (2000 chars) - metadata: chunk="1/4"
Chunk 2 (2000 chars) - metadata: chunk="2/4"
Chunk 3 (2000 chars) - metadata: chunk="3/4"
Chunk 4 (2000 chars) - metadata: chunk="4/4"
```

**Benefits:**
- Maintains context within reasonable bounds
- Improves search accuracy
- Optimizes embedding generation
- Allows partial document activation

### 3. Metadata Management

Each document chunk includes metadata:
```json
{
  "filename": "company_faq.txt",
  "upload_time": "2024-01-15T10:30:00Z",
  "uploaded_by": "admin@company.com",
  "chunk": "2/5",
  "custom_field": "custom_value"
}
```

## Channel Configuration

### Channel-Specific Documents

Documents can be configured for specific channels:

1. **Facebook Only**
   ```json
   { "channels": ["facebook"] }
   ```
   Used only for Facebook comment responses

2. **Messenger Only**
   ```json
   { "channels": ["messenger"] }
   ```
   Used only for Messenger chat responses

3. **Both Channels**
   ```json
   { "channels": ["facebook", "messenger"] }
   ```
   Used for both Facebook and Messenger

### Use Case Examples

**Customer Support FAQ** - Both channels
```bash
curl -X POST /api/dashboard/rag/upload \
  -F "file=@support_faq.txt" \
  -F "channels=facebook,messenger"
```

**Product Catalog** - Messenger only
```bash
curl -X POST /api/dashboard/rag/upload \
  -F "file=@products.csv" \
  -F "channels=messenger"
```

**Community Guidelines** - Facebook only
```bash
curl -X POST /api/dashboard/rag/upload \
  -F "file=@guidelines.md" \
  -F "channels=facebook"
```

## Document Examples

### Example 1: FAQ Document (TXT)
```text
# Frequently Asked Questions

Q: What are your business hours?
A: We are open Monday-Friday 9AM-6PM EST, Saturday 10AM-4PM EST.

Q: What is your return policy?
A: We accept returns within 30 days of purchase with original receipt.

Q: Do you offer international shipping?
A: Yes, we ship to over 50 countries worldwide.
```

### Example 2: Product Catalog (CSV)
```csv
Product,Price,Description,Availability
Widget A,$19.99,High-quality widget for everyday use,In Stock
Widget B,$29.99,Premium widget with advanced features,Limited
Widget C,$39.99,Professional-grade widget,Pre-order
```

### Example 3: Policy Document (Markdown)
```markdown
# Privacy Policy

## Data Collection
We collect only essential information needed to provide our services.

## Data Usage
Your data is used solely for improving customer experience.

## Data Protection
We implement industry-standard security measures.
```

### Example 4: Configuration (JSON)
```json
{
  "support_levels": {
    "basic": {
      "response_time": "24 hours",
      "channels": ["email"]
    },
    "premium": {
      "response_time": "1 hour",
      "channels": ["email", "phone", "chat"]
    }
  }
}
```

## Best Practices

### 1. Document Preparation
- **Keep it Concise**: Focus on customer-facing information
- **Use Clear Language**: Avoid technical jargon
- **Structure Content**: Use headings and bullet points
- **Update Regularly**: Keep information current

### 2. File Organization
- **Naming Convention**: Use descriptive filenames
- **Version Control**: Include version in metadata
- **Category Tags**: Use metadata for categorization
- **Regular Audits**: Review and update documents periodically

### 3. Channel Strategy
- **Facebook Comments**: Public-facing information, community guidelines
- **Messenger Chat**: Detailed support, personalized assistance
- **Both Channels**: Core information, FAQs, policies

### 4. Performance Optimization
- **File Size**: Keep individual files under 5MB
- **Chunk Size**: Default 2000 chars works for most cases
- **Active Documents**: Only activate necessary documents
- **Regular Cleanup**: Remove outdated documents

## Search and Retrieval

### How RAG Search Works

1. **Query Processing**: User message is converted to embedding
2. **Similarity Search**: Find most relevant document chunks
3. **Context Building**: Combine relevant chunks
4. **Response Generation**: AI uses context to generate response

### Search Relevance Factors

- **Semantic Similarity**: Meaning-based matching
- **Channel Context**: Only searches channel-specific documents
- **Active Status**: Only active documents are searched
- **Recency**: Newer documents may be prioritized

## Monitoring and Debugging

### Check Document Status
```bash
# View all documents for a page
curl -X GET "http://localhost:8080/api/dashboard/rag/documents?page_id=123456789" \
  -H "Authorization: Bearer [token]"
```

### Monitor Processing
Check application logs for processing status:
```
INFO: RAG document received, processing in background
INFO: Starting background RAG document processing
INFO: Stored embedding for chunk 1/5
INFO: RAG document processing completed successfully
```

### Common Log Messages
```
INFO: Document upload successful - filename: faq.txt
WARN: Large document split into 10 chunks
ERROR: Failed to extract text from file: unsupported format
INFO: Document activated for channels: [facebook, messenger]
```

## Troubleshooting

### Document Not Affecting Responses

1. **Check Active Status**
   ```bash
   curl -X GET "/api/dashboard/rag/documents/details?page_id=123&active=true"
   ```

2. **Verify Channel Assignment**
   - Ensure document is active for the correct channel
   - Check if channels array includes the intended channel

3. **Review Content Relevance**
   - Document content should relate to user queries
   - Consider adding more specific information

### Upload Failures

1. **File Too Large**
   - Split into multiple smaller files
   - Compress content while maintaining readability

2. **Unsupported Format**
   - Convert to supported format (TXT, MD, CSV, JSON)
   - Check file extension is correct

3. **Processing Timeout**
   - Check logs for background processing errors
   - Reduce file size or complexity

### Performance Issues

1. **Slow Response Times**
   - Reduce number of active documents
   - Optimize document content
   - Check vector database performance

2. **Inaccurate Responses**
   - Review document content for clarity
   - Ensure proper chunking
   - Update documents with more specific information

## Advanced Features

### 1. Bulk Upload Script
```bash
#!/bin/bash
# Upload multiple documents
for file in ./documents/*.txt; do
  curl -X POST http://localhost:8080/api/dashboard/rag/upload \
    -H "Authorization: Bearer [token]" \
    -F "file=@$file" \
    -F "page_id=123456789" \
    -F "channels=facebook,messenger"
  sleep 2  # Rate limiting
done
```

### 2. Document Versioning
Use metadata to track versions:
```bash
curl -X POST /api/dashboard/rag/upload \
  -F "file=@policy_v2.txt" \
  -F "metadata=version=2.0,replaces=policy_v1.txt"
```

### 3. Conditional Activation
Activate documents based on conditions:
```javascript
// Pseudocode for conditional activation
if (customerTier === 'premium') {
  activateDocument('premium_features.txt');
} else {
  activateDocument('basic_features.txt');
}
```

## Security Considerations

### 1. Content Guidelines
- Never upload sensitive data (passwords, SSN, credit cards)
- Avoid personal customer information
- Review documents for compliance (GDPR, CCPA)

### 2. Access Control
- Only company admins can upload documents
- Document management requires authentication
- Page-level isolation ensures data security

### 3. Data Retention
- Documents remain until explicitly deleted
- Consider retention policies for compliance
- Regular audits of uploaded content

## Integration Examples

### Example 1: Product Documentation
```bash
# Upload product manual
curl -X POST http://localhost:8080/api/dashboard/rag/upload \
  -H "Authorization: Bearer [token]" \
  -F "file=@product_manual.md" \
  -F "page_id=123456789" \
  -F "channels=messenger" \
  -F "metadata=product=WidgetX,version=3.0"
```

### Example 2: Seasonal Promotions
```bash
# Upload holiday promotions
curl -X POST http://localhost:8080/api/dashboard/rag/upload \
  -H "Authorization: Bearer [token]" \
  -F "file=@black_friday_deals.txt" \
  -F "page_id=123456789" \
  -F "channels=facebook,messenger" \
  -F "metadata=campaign=black_friday,year=2024"

# Deactivate after promotion ends
curl -X PUT http://localhost:8080/api/dashboard/rag/document/toggle \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer [token]" \
  -d '{
    "page_id": "123456789",
    "filename": "black_friday_deals.txt",
    "is_active": false
  }'
```

### Example 3: Multi-Language Support
```bash
# Upload documents in different languages
for lang in en es fr; do
  curl -X POST http://localhost:8080/api/dashboard/rag/upload \
    -H "Authorization: Bearer [token]" \
    -F "file=@faq_$lang.txt" \
    -F "page_id=123456789" \
    -F "channels=facebook,messenger" \
    -F "metadata=language=$lang"
done
```

## Future Enhancements

- PDF file support
- Word document (.docx) support
- Automatic document updates from URLs
- Document approval workflow
- Version control integration
- A/B testing for document effectiveness
- Analytics on document usage
- Multi-language document management
- Document templates
- Auto-categorization using AI
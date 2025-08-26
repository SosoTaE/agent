# RAG Document Upload Feature

## Overview
The RAG (Retrieval-Augmented Generation) document upload feature allows you to upload knowledge base documents that the bot can use to provide more accurate and contextual responses to users on both Facebook and Messenger platforms.

## How It Works

### Document Processing
1. Documents are uploaded via the API endpoint `/api/rag/upload`
2. The system automatically splits large documents into smaller chunks (2000 characters each)
3. Each chunk is converted into embeddings using Voyage AI
4. Embeddings are stored in the vector database for fast similarity search
5. When users ask questions, the system searches for relevant content and includes it as context for the AI

### Automatic Platform Detection
- **No channel configuration needed**: By default, uploaded documents are automatically enabled for both Facebook and Messenger platforms
- The system checks the vector database for relevant documents when processing messages
- If relevant documents are found, they are automatically used as context for generating responses

## API Endpoints

### Upload Document
**POST** `/api/rag/upload`

Upload a document to the RAG system.

**Form Data Parameters:**
- `file` (required): The file to upload (supported: .txt, .md, .csv, .json)
- `page_id` (required): The Facebook page ID
- `channels` (optional): Comma-separated list of channels ("facebook", "messenger"). Defaults to both platforms if not specified
- `source` (optional): Source identifier (defaults to "manual_upload")
- `metadata` (optional): Key=value pairs separated by commas

**Example Request:**
```bash
curl -X POST https://your-api.com/api/rag/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@knowledge.txt" \
  -F "page_id=123456789"
```

**Example with specific channels:**
```bash
curl -X POST https://your-api.com/api/rag/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@knowledge.txt" \
  -F "page_id=123456789" \
  -F "channels=facebook,messenger"
```

### List Documents
**GET** `/api/rag/list?page_id=YOUR_PAGE_ID`

Retrieve all RAG documents for a specific page.

**Response includes:**
- Document content and metadata
- Channel availability (Facebook/Messenger)
- Active status
- Creation timestamps

### Toggle Document Channels
**POST** `/api/rag/toggle-channel`

Enable or disable a document for specific platforms.

**Request Body:**
```json
{
  "document_name": "knowledge.txt",
  "platform_name": "facebook",
  "value": true
}
```

### Delete Document
**DELETE** `/api/rag/delete`

Remove a document from the vector database.

**Request Body:**
```json
{
  "page_id": "123456789",
  "filename": "knowledge.txt"
}
```

## Integration with Message Handling

### Automatic Context Retrieval
When a message is received from either Facebook or Messenger:

1. The system automatically queries the vector database for relevant documents
2. If documents exist and match the user's query, they are retrieved as context
3. The AI uses this context to generate more accurate, knowledge-based responses
4. No configuration needed - if documents exist, they will be used

### Code Example
The message handler automatically checks for RAG documents:

```go
// Check vector database for available RAG documents and retrieve context if found
var ragContext string

// Try to get relevant context from vector database
ragContext, err = services.GetRAGContextForChannel(ctx, messageText, company.CompanyID, pageID, "messenger")
if err != nil {
    slog.Warn("Failed to fetch RAG context from vector DB", "error", err)
    // Continue without RAG context
} else if ragContext != "" {
    slog.Info("RAG context retrieved from vector DB",
        "contextLength", len(ragContext),
        "companyID", company.CompanyID,
        "pageID", pageID,
        "channel", "messenger",
    )
}
```

## File Support

### Currently Supported File Types
- `.txt` - Plain text files
- `.md` - Markdown files
- `.csv` - CSV files
- `.json` - JSON files

### File Size Limits
- Maximum file size: 10MB
- Large files are automatically split into chunks for processing

## Best Practices

### Content Organization
1. **Use clear, structured content**: Well-organized documents with headers and sections work best
2. **Keep related information together**: The chunking algorithm preserves context within sections
3. **Include keywords**: Use relevant terms that users might ask about

### Document Management
1. **Regular updates**: Keep your knowledge base current by uploading updated documents
2. **Monitor relevance**: Check which documents are being used through the logs
3. **Platform-specific content**: You can enable/disable documents per platform if needed

### Performance Optimization
1. **Chunk size**: Documents are automatically split into 2000-character chunks for optimal performance
2. **Similarity threshold**: The system uses cosine similarity to find the most relevant content
3. **Context limits**: Maximum 10,000 characters of context are included to prevent token overflow

## Troubleshooting

### Documents Not Being Used
If your uploaded documents aren't being used:
1. Check that documents are uploaded for the correct page_id
2. Verify documents are active (not toggled off)
3. Ensure the user's query has some relevance to the document content
4. Check logs for "RAG context retrieved" messages

### Upload Failures
Common causes:
1. File size exceeds 10MB
2. Unsupported file format
3. Missing Voyage API key configuration
4. Network timeouts for large files

### Platform-Specific Issues
- Documents are enabled for both platforms by default
- Use the toggle-channel endpoint to disable for specific platforms if needed
- Check the channels field in document listings to verify platform availability

## Security Considerations

1. **Authentication**: All endpoints require proper authentication
2. **Company/Page validation**: System verifies ownership before allowing operations
3. **Content validation**: Documents are processed for text content only
4. **Metadata tracking**: Upload user and timestamp are recorded

## Future Enhancements

Planned improvements:
- Support for PDF and DOCX files
- Automatic document refresh from external sources
- Advanced chunking strategies
- Multi-language support
- Analytics on document usage
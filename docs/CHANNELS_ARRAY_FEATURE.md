# Channels Array Feature Documentation

## Overview
The Channels Array feature allows RAG documents and CRM links to be selectively enabled for Facebook comments, Messenger, or both channels simultaneously. Instead of duplicating documents for each channel, a single document can be active for multiple channels using an array field.

## Key Concept
**Channels are controlled by array presence:**
- `channels: ["facebook", "messenger"]` - Document active for BOTH channels
- `channels: ["facebook"]` - Document active ONLY for Facebook comments
- `channels: ["messenger"]` - Document active ONLY for Messenger
- `channels: []` - Document is OFF for all channels (effectively disabled)

## Data Model

### Vector Document Structure
```javascript
{
  "_id": ObjectId("..."),
  "company_id": "comp123",
  "page_id": "461998383671026",
  "content": "Product information...",
  "embedding": [0.123, 0.456, ...],
  "channels": ["facebook", "messenger"],  // Active for both channels
  "source": "manual_upload",
  "is_active": true,
  "metadata": {
    "filename": "products.pdf",
    "uploaded_by": "admin@company.com"
  },
  "created_at": ISODate("2024-01-15"),
  "updated_at": ISODate("2024-01-16")
}
```

## API Endpoints

### 1. Upload Document with Channels
**Endpoint:** `POST /api/dashboard/rag/upload`

**Request:**
```bash
# Upload for both channels
curl -X POST http://localhost:3000/api/dashboard/rag/upload \
  -H "Cookie: session=your_session_cookie" \
  -F "file=@document.pdf" \
  -F "page_id=461998383671026" \
  -F "channels=facebook,messenger" \
  -F "source=manual_upload"

# Upload for Facebook only
curl -X POST http://localhost:3000/api/dashboard/rag/upload \
  -H "Cookie: session=your_session_cookie" \
  -F "file=@facebook_only.pdf" \
  -F "page_id=461998383671026" \
  -F "channels=facebook" \
  -F "source=manual_upload"

# Upload for Messenger only
curl -X POST http://localhost:3000/api/dashboard/rag/upload \
  -H "Cookie: session=your_session_cookie" \
  -F "file=@messenger_only.pdf" \
  -F "page_id=461998383671026" \
  -F "channels=messenger" \
  -F "source=manual_upload"
```

**Form Fields:**
- `file` (required): Document file
- `page_id` (required): Facebook page ID
- `channels` (required): Comma-separated list of channels
  - Valid values: `"facebook"`, `"messenger"`, or `"facebook,messenger"`
- `source` (optional): Source identifier
- `metadata` (optional): Additional metadata

**Response:**
```json
{
  "message": "File received and queued for processing",
  "filename": "document.pdf",
  "size": 245632,
  "source": "manual_upload",
  "page_id": "461998383671026",
  "channels": ["facebook", "messenger"],
  "status": "processing"
}
```

### 2. Update Document Channels
**Endpoint:** `PUT /api/dashboard/rag/document/channels`

**Purpose:** Change which channels a document is active for

**Request Body:**
```json
{
  "page_id": "461998383671026",
  "filename": "products.pdf",
  "channels": ["facebook"]  // Change to Facebook only
}
```

**Examples:**

Enable for both channels:
```json
{
  "page_id": "461998383671026",
  "filename": "products.pdf",
  "channels": ["facebook", "messenger"]
}
```

Disable for all channels (turn off):
```json
{
  "page_id": "461998383671026",
  "filename": "products.pdf",
  "channels": []  // Empty array = document is OFF
}
```

Switch from Facebook to Messenger only:
```json
{
  "page_id": "461998383671026",
  "filename": "products.pdf",
  "channels": ["messenger"]
}
```

**Response:**
```json
{
  "message": "Document channels updated successfully",
  "updated_count": 5,  // Number of chunks updated
  "channels": ["facebook"],
  "document_active": true  // false if channels is empty
}
```

### 3. List Documents with Channel Information
**Endpoint:** `GET /api/dashboard/rag/documents?page_id={page_id}`

**Response:**
```json
{
  "documents": [
    {
      "_id": "507f1f77bcf86cd799439011",
      "content": "Product catalog...",
      "channels": ["facebook", "messenger"],  // Active for both
      "source": "manual_upload",
      "is_active": true,
      "metadata": {
        "filename": "products.pdf"
      }
    },
    {
      "_id": "507f1f77bcf86cd799439012",
      "content": "Support documentation...",
      "channels": ["messenger"],  // Active only for Messenger
      "source": "manual_upload",
      "is_active": true,
      "metadata": {
        "filename": "support.pdf"
      }
    },
    {
      "_id": "507f1f77bcf86cd799439013",
      "content": "Public FAQ...",
      "channels": [],  // Disabled for all channels
      "source": "manual_upload",
      "is_active": true,
      "metadata": {
        "filename": "faq.pdf"
      }
    }
  ]
}
```

## How It Works

### Channel Detection in Handlers

#### Message Handler (Messenger)
```go
// When processing a Messenger message
if shouldUseRAG {
    // Get RAG context for messenger channel
    // This will only return documents where "messenger" is in the channels array
    ragContext, err = services.GetRAGContextForChannel(
        ctx, messageText, companyID, pageID, "messenger"
    )
}
```

#### Comment Handler (Facebook)
```go
// When processing a Facebook comment
if shouldUseRAG {
    // Get RAG context for facebook channel
    // This will only return documents where "facebook" is in the channels array
    ragContext, err = services.GetRAGContextForChannel(
        ctx, message, companyID, pageID, "facebook"
    )
}
```

### MongoDB Query Examples

#### Find documents active for Messenger:
```javascript
db.vector_documents.find({
  "company_id": "comp123",
  "page_id": "461998383671026",
  "channels": "messenger",  // MongoDB checks if "messenger" is in the array
  "is_active": true
})
```

#### Find documents active for Facebook:
```javascript
db.vector_documents.find({
  "company_id": "comp123",
  "page_id": "461998383671026",
  "channels": "facebook",  // MongoDB checks if "facebook" is in the array
  "is_active": true
})
```

#### Find documents active for both channels:
```javascript
db.vector_documents.find({
  "company_id": "comp123",
  "page_id": "461998383671026",
  "channels": { $all: ["facebook", "messenger"] },
  "is_active": true
})
```

#### Find disabled documents:
```javascript
db.vector_documents.find({
  "company_id": "comp123",
  "page_id": "461998383671026",
  "channels": { $size: 0 },  // Empty array
  "is_active": true
})
```

## Use Cases

### 1. Shared Knowledge Base
Upload documents that should be available for both channels:
```bash
curl -X POST .../rag/upload \
  -F "file=@company_info.pdf" \
  -F "channels=facebook,messenger"
```

### 2. Channel-Specific Content
Upload sensitive information only for private Messenger:
```bash
curl -X POST .../rag/upload \
  -F "file=@customer_accounts.pdf" \
  -F "channels=messenger"
```

Upload public information only for Facebook comments:
```bash
curl -X POST .../rag/upload \
  -F "file=@public_announcements.pdf" \
  -F "channels=facebook"
```

### 3. Temporary Disable
Disable a document without deleting it:
```json
PUT /api/dashboard/rag/document/channels
{
  "page_id": "461998383671026",
  "filename": "outdated_info.pdf",
  "channels": []  // Empty array disables for all channels
}
```

### 4. Channel Migration
Move document from Facebook to Messenger:
```json
// Step 1: Current state
{ "channels": ["facebook"] }

// Step 2: Update channels
PUT /api/dashboard/rag/document/channels
{
  "filename": "sensitive_doc.pdf",
  "channels": ["messenger"]
}

// Step 3: New state
{ "channels": ["messenger"] }
```

## Benefits of Array Approach

### 1. **Storage Efficiency**
- Single document serves multiple channels
- No need to duplicate embeddings
- Reduced storage costs

### 2. **Flexible Control**
- Easy to enable/disable channels
- Can toggle channels without re-uploading
- Granular control per document

### 3. **Simple Management**
- One document = one entity
- Clear channel assignment
- Easy bulk operations

### 4. **Future Scalability**
- Easy to add new channels (Instagram, WhatsApp)
- Array can grow without schema changes
- Backward compatible

## Migration from Single Channel

### For Existing Documents
```javascript
// Add channels field to existing documents
db.vector_documents.updateMany(
  { channels: { $exists: false } },
  { 
    $set: { 
      channels: ["facebook", "messenger"]  // Default to both
    } 
  }
)

// Or migrate based on old channel field
db.vector_documents.updateMany(
  { channel: "facebook", channels: { $exists: false } },
  { 
    $set: { channels: ["facebook"] },
    $unset: { channel: "" }
  }
)
```

## Best Practices

### 1. **Default Channels**
- When in doubt, enable for both channels
- Explicitly disable sensitive content
- Document channel decisions

### 2. **Naming Conventions**
- Use descriptive filenames
- Consider prefixes: `public_`, `private_`, `both_`
- Include channel info in metadata

### 3. **Regular Audits**
- Review channel assignments monthly
- Check for accidentally exposed content
- Verify channel-specific behavior

### 4. **Testing**
```bash
# Test Facebook context
curl -X POST .../test/rag-context \
  -d '{"message": "test query", "channel": "facebook"}'

# Test Messenger context  
curl -X POST .../test/rag-context \
  -d '{"message": "test query", "channel": "messenger"}'
```

## Performance Considerations

### Indexing
Ensure MongoDB has proper indexes:
```javascript
db.vector_documents.createIndex({
  "company_id": 1,
  "page_id": 1,
  "channels": 1,
  "is_active": 1
})
```

### Query Optimization
- Use array contains for single channel queries
- Use `$all` operator for multi-channel requirements
- Use `$size: 0` for finding disabled documents

## Security Notes

1. **Channel Validation**: Always validate channel values
2. **Empty Array Handling**: Empty array = disabled (safe default)
3. **Access Control**: Verify user can modify channels
4. **Audit Logging**: Log all channel changes

## Examples

### Complete Upload Flow
```javascript
// 1. Upload document for both channels
const formData = new FormData();
formData.append('file', file);
formData.append('page_id', '461998383671026');
formData.append('channels', 'facebook,messenger');

fetch('/api/dashboard/rag/upload', {
  method: 'POST',
  body: formData
});

// 2. Later, disable for Facebook
fetch('/api/dashboard/rag/document/channels', {
  method: 'PUT',
  body: JSON.stringify({
    page_id: '461998383671026',
    filename: 'document.pdf',
    channels: ['messenger']  // Now only Messenger
  })
});

// 3. Temporarily disable completely
fetch('/api/dashboard/rag/document/channels', {
  method: 'PUT',
  body: JSON.stringify({
    page_id: '461998383671026',
    filename: 'document.pdf',
    channels: []  // Disabled
  })
});

// 4. Re-enable for both
fetch('/api/dashboard/rag/document/channels', {
  method: 'PUT',
  body: JSON.stringify({
    page_id: '461998383671026',
    filename: 'document.pdf',
    channels: ['facebook', 'messenger']
  })
});
```

## Summary

The channels array approach provides:
- **Flexibility**: Documents can be active for any combination of channels
- **Efficiency**: Single storage, multiple uses
- **Simplicity**: Array presence determines activation
- **Scalability**: Easy to add new channels in the future

Key principle: **If a channel is in the array, the document is active for that channel. Empty array = disabled.**
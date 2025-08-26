# Facebook Post RAG Processing

## Overview
Facebook post content is now automatically processed and stored as RAG (Retrieval-Augmented Generation) documents using the same algorithm as manual file uploads. This enables the bot to have contextual awareness of post content when responding to comments.

## Implementation Details

### Automatic Processing
When a comment webhook is received:
1. The system fetches the Facebook post content
2. If the post content is substantial (>100 characters), it's processed as a RAG document
3. Processing happens asynchronously in the background

### Chunking Algorithm
Posts are processed using the same chunking algorithm as document uploads:
- **Chunk size:** 2000 characters per chunk
- **Splitting method:** Word-based splitting to preserve context
- **Metadata preservation:** Each chunk maintains post metadata

### Storage Structure
Post content is stored with:
- **Source:** `facebook_post`
- **Channels:** Both `facebook` and `messenger` enabled
- **Metadata:**
  - `post_id`: Facebook post ID
  - `source_type`: `facebook_post`
  - `created_at`: Processing timestamp
  - `chunk`: Chunk number (if multiple chunks)

### Deduplication
- System checks if post content is already processed before storing
- Uses `post_id` metadata to prevent duplicate processing
- Subsequent comments on the same post won't trigger reprocessing

### Integration with RAG Context
- Post content becomes part of the knowledge base
- Available for semantic search during comment responses
- Uses the same `GetRAGContextForChannel` method as messenger handler
- Automatically retrieves relevant context from vector database
- Filtered by channel (Facebook) when processing comments

## Code Components

### Comment Handler (`handlers/comment_handler.go`)
- **Function:** `HandleComment`
  - Triggers post processing when handling comments
  - Calls `processPostContentAsRAG` asynchronously

- **Function:** `processPostContentAsRAG`
  - Processes Facebook post content
  - Splits into chunks
  - Stores embeddings with channels

- **Function:** `splitPostIntoChunks`
  - Identical algorithm to document upload chunking
  - Word-based splitting with 2000 char limit

### Services (`services/vectordb.go`)
- **Function:** `CheckRAGDocumentExists`
  - Checks if post already processed
  - Prevents duplicate storage

## Benefits
1. **Contextual Responses:** Bot understands post content when responding to comments
2. **Consistency:** Uses same processing algorithm as manual uploads
3. **Efficiency:** Asynchronous processing doesn't block comment handling
4. **Deduplication:** Prevents redundant storage of same post content
5. **Channel Support:** Automatically available for both Facebook and Messenger contexts
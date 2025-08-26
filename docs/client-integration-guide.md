# RAG System Client Integration Guide

This guide helps frontend developers integrate with the RAG (Retrieval-Augmented Generation) document management system.

## Overview

The RAG system allows you to upload documents that the bot uses as knowledge sources when responding to Facebook comments and Messenger messages. Documents can be independently enabled/disabled for each platform.

## Core Concepts

- **Documents**: Individual text chunks stored in MongoDB (typically 2000 chars each)
- **Files**: Original uploaded files that get split into documents/chunks
- **Channels**: Facebook comments and Messenger - each file can be active on either, both, or neither
- **Company**: Your organization that owns multiple Facebook pages
- **Pages**: Individual Facebook pages where the bot operates

---

## Essential Endpoints for Client UI

### 1. Get All Files Overview
**Use this to display the main document list in your UI**

```http
GET /dashboard/rag/files
Authorization: Bearer <token>
```

**Response:**
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
  }
]
```

**UI Implementation:**
```javascript
// Fetch and display files
async function loadFiles() {
  const response = await fetch('/dashboard/rag/files', {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  const files = await response.json();
  
  // Render file list
  files.forEach(file => {
    renderFileRow(file);
  });
}

function renderFileRow(file) {
  return `
    <tr>
      <td>${file.filename}</td>
      <td>
        <input type="checkbox" 
               ${file.facebook ? 'checked' : ''} 
               onchange="toggleChannel('${file.filename}', 'facebook', this.checked)">
      </td>
      <td>
        <input type="checkbox" 
               ${file.messenger ? 'checked' : ''} 
               onchange="toggleChannel('${file.filename}', 'messenger', this.checked)">
      </td>
    </tr>
  `;
}
```

---

### 2. Toggle File Channel
**Use this when users click checkboxes to enable/disable a file for a platform**

```http
PUT /dashboard/rag/document/toggle-channel
Authorization: Bearer <token>
Content-Type: application/json

{
  "document_name": "product_catalog.pdf",
  "platform_name": "facebook",
  "value": true
}
```

**Response:**
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

**UI Implementation:**
```javascript
async function toggleChannel(filename, platform, enabled) {
  try {
    const response = await fetch('/dashboard/rag/document/toggle-channel', {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        document_name: filename,
        platform_name: platform,
        value: enabled
      })
    });
    
    if (response.ok) {
      showToast(`${filename} ${enabled ? 'enabled' : 'disabled'} for ${platform}`);
    } else {
      const error = await response.json();
      showError(error.error);
    }
  } catch (err) {
    showError('Failed to update document');
  }
}
```

---

### 3. Upload New Document
**Use this for the upload form**

```http
POST /dashboard/rag/upload
Authorization: Bearer <token>
Content-Type: multipart/form-data

Form Data:
- file: <file>
- page_id: "page_123"
- channels: "facebook,messenger"
```

**Response:**
```json
{
  "message": "File received and queued for processing",
  "filename": "new_document.pdf",
  "size": 102400,
  "status": "processing"
}
```

**UI Implementation:**
```javascript
async function uploadFile(file, pageId, enabledChannels) {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('page_id', pageId);
  formData.append('channels', enabledChannels.join(','));
  
  const response = await fetch('/dashboard/rag/upload', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`
    },
    body: formData
  });
  
  if (response.ok) {
    const result = await response.json();
    showToast(`${result.filename} uploaded successfully`);
    // Refresh file list after a delay (processing happens in background)
    setTimeout(() => loadFiles(), 3000);
  }
}
```

---

### 4. Delete Document
**Use this for the delete button**

```http
DELETE /dashboard/rag/document
Authorization: Bearer <token>

Query Parameters:
- company_id: "comp_123"
- page_id: "page_456"
- filename: "product_catalog.pdf"
```

**Response:**
```json
{
  "success": true,
  "message": "Documents deleted successfully",
  "deleted_count": 10
}
```

**UI Implementation:**
```javascript
async function deleteFile(filename) {
  if (!confirm(`Delete ${filename}?`)) return;
  
  const params = new URLSearchParams({
    company_id: companyId,
    page_id: pageId,
    filename: filename
  });
  
  const response = await fetch(`/dashboard/rag/document?${params}`, {
    method: 'DELETE',
    headers: { 'Authorization': `Bearer ${token}` }
  });
  
  if (response.ok) {
    showToast(`${filename} deleted`);
    loadFiles(); // Refresh list
  }
}
```

---

## Complete UI Example

### React Component

```jsx
import React, { useState, useEffect } from 'react';

function RAGDocumentManager() {
  const [files, setFiles] = useState([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);

  useEffect(() => {
    loadFiles();
  }, []);

  async function loadFiles() {
    setLoading(true);
    try {
      const response = await fetch('/dashboard/rag/files', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      const data = await response.json();
      setFiles(data);
    } catch (err) {
      console.error('Failed to load files:', err);
    }
    setLoading(false);
  }

  async function handleToggle(filename, platform, value) {
    try {
      const response = await fetch('/dashboard/rag/document/toggle-channel', {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          document_name: filename,
          platform_name: platform,
          value: value
        })
      });
      
      if (response.ok) {
        // Update local state
        setFiles(files.map(f => 
          f.filename === filename 
            ? { ...f, [platform]: value }
            : f
        ));
      }
    } catch (err) {
      console.error('Toggle failed:', err);
    }
  }

  async function handleUpload(event) {
    const file = event.target.files[0];
    if (!file) return;

    setUploading(true);
    const formData = new FormData();
    formData.append('file', file);
    formData.append('page_id', pageId);
    formData.append('channels', 'facebook,messenger');

    try {
      const response = await fetch('/dashboard/rag/upload', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` },
        body: formData
      });

      if (response.ok) {
        setTimeout(() => loadFiles(), 3000); // Refresh after processing
      }
    } catch (err) {
      console.error('Upload failed:', err);
    }
    setUploading(false);
  }

  async function handleDelete(filename) {
    if (!confirm(`Delete ${filename}?`)) return;

    try {
      const params = new URLSearchParams({
        company_id: companyId,
        page_id: pageId,
        filename: filename
      });

      const response = await fetch(`/dashboard/rag/document?${params}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });

      if (response.ok) {
        setFiles(files.filter(f => f.filename !== filename));
      }
    } catch (err) {
      console.error('Delete failed:', err);
    }
  }

  return (
    <div className="rag-manager">
      <div className="header">
        <h2>RAG Documents</h2>
        <input 
          type="file" 
          onChange={handleUpload}
          disabled={uploading}
          accept=".txt,.pdf,.md,.csv,.json"
        />
      </div>

      {loading ? (
        <div>Loading...</div>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Filename</th>
              <th>Facebook</th>
              <th>Messenger</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {files.map(file => (
              <tr key={file.filename}>
                <td>{file.filename}</td>
                <td>
                  <input
                    type="checkbox"
                    checked={file.facebook}
                    onChange={(e) => handleToggle(file.filename, 'facebook', e.target.checked)}
                  />
                </td>
                <td>
                  <input
                    type="checkbox"
                    checked={file.messenger}
                    onChange={(e) => handleToggle(file.filename, 'messenger', e.target.checked)}
                  />
                </td>
                <td>
                  <button onClick={() => handleDelete(file.filename)}>
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
```

---

## UI/UX Best Practices

### 1. Visual States
```css
/* Active/Inactive states */
.channel-enabled { background: #4CAF50; }
.channel-disabled { background: #f44336; }

/* File status indicators */
.file-row.all-disabled { opacity: 0.5; }
.file-row.partially-enabled { border-left: 3px solid orange; }
.file-row.fully-enabled { border-left: 3px solid green; }
```

### 2. Loading States
- Show spinner during file upload
- Display progress for large files
- Show "Processing..." status after upload
- Auto-refresh list after 3-5 seconds

### 3. Error Handling
```javascript
function handleApiError(error) {
  const messages = {
    401: 'Please login again',
    403: 'You don\'t have permission',
    404: 'File not found',
    500: 'Server error, please try again'
  };
  
  showNotification(messages[error.status] || 'An error occurred');
}
```

### 4. Confirmation Dialogs
```javascript
// Before delete
if (!confirm(`This will remove ${filename} from all pages. Continue?`)) {
  return;
}

// Before disabling all channels
if (!file.facebook && !file.messenger) {
  if (!confirm('This will completely deactivate the file. Continue?')) {
    return;
  }
}
```

---

## Common Workflows

### 1. Initial Setup
```javascript
// 1. Load files on page load
await loadFiles();

// 2. Setup auto-refresh if needed
setInterval(loadFiles, 30000); // Refresh every 30 seconds
```

### 2. Bulk Operations
```javascript
// Enable all files for Facebook
async function enableAllForFacebook() {
  const files = await getFiles();
  
  for (const file of files) {
    if (!file.facebook) {
      await toggleChannel(file.filename, 'facebook', true);
      await delay(100); // Rate limiting
    }
  }
}

// Disable inactive files
async function disableInactiveFiles() {
  const files = await getFiles();
  
  const inactive = files.filter(f => !f.facebook && !f.messenger);
  for (const file of inactive) {
    await deleteFile(file.filename);
  }
}
```

### 3. Search and Filter
```javascript
// Client-side filtering
function filterFiles(files, query) {
  return files.filter(file => {
    const matchesName = file.filename.toLowerCase().includes(query.toLowerCase());
    const matchesStatus = 
      (query === 'active' && (file.facebook || file.messenger)) ||
      (query === 'inactive' && (!file.facebook && !file.messenger));
    
    return matchesName || matchesStatus;
  });
}
```

---

## Testing Your Implementation

### 1. Test Upload
```javascript
// Test with different file types
const testFiles = [
  'test.txt',      // Should work
  'test.pdf',      // Should work
  'test.docx',     // Should fail
  'large_file.pdf' // Test size limits
];
```

### 2. Test Toggle States
```javascript
// Test all combinations
const testCases = [
  { facebook: false, messenger: false }, // Both off
  { facebook: true, messenger: false },  // Facebook only
  { facebook: false, messenger: true },   // Messenger only
  { facebook: true, messenger: true }     // Both on
];
```

### 3. Test Error Cases
- Upload without authentication
- Toggle non-existent file
- Delete already deleted file
- Upload unsupported file type

---

## Performance Tips

1. **Batch Operations**: When updating multiple files, add a small delay between requests
2. **Caching**: Cache file list client-side and refresh periodically
3. **Optimistic Updates**: Update UI immediately, then sync with server
4. **Debouncing**: Debounce rapid toggle clicks to prevent excessive API calls

```javascript
const toggleChannel = debounce(async (filename, platform, value) => {
  // API call here
}, 500);
```

---

## Support & Troubleshooting

### Common Issues

**Files not appearing after upload**
- Documents are processed asynchronously
- Wait 3-5 seconds and refresh
- Check file size and format

**Toggle not working**
- Verify authentication token
- Check network console for errors
- Ensure filename matches exactly

**Deleted files reappearing**
- Clear browser cache
- Check if file exists on other pages
- Verify deletion was successful

### Debug Mode
```javascript
// Enable debug logging
const DEBUG = true;

function apiCall(endpoint, options) {
  if (DEBUG) {
    console.log('API Call:', endpoint, options);
  }
  
  return fetch(endpoint, options).then(res => {
    if (DEBUG) {
      console.log('Response:', res.status, res);
    }
    return res;
  });
}
```
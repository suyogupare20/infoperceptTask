package store

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
)

type FileStore struct {
    root string
}

func NewFileStore(root string) *FileStore {
    return &FileStore{root: root}
}

func (fs *FileStore) Handler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodPut:
        fs.handlePut(w, r)
    case http.MethodGet:
        fs.handleGet(w, r)
    case http.MethodHead:
        fs.handleHead(w, r)
    case http.MethodDelete:
        fs.handleDelete(w, r)
    default:
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
    }
}

func (fs *FileStore) fullPath(bucket, key string) string {
    return filepath.Join(fs.root, bucket, key)
}

func (fs *FileStore) handlePut(w http.ResponseWriter, r *http.Request) {
    bucket, key, err := fs.parsePath(r.URL.Path)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    path := fs.fullPath(bucket, key)
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        http.Error(w, "failed to create directory", http.StatusInternalServerError)
        return
    }

    file, err := os.Create(path)
    if err != nil {
        http.Error(w, "failed to create file", http.StatusInternalServerError)
        return
    }
    defer file.Close()

    hasher := sha256.New()
    mw := io.MultiWriter(file, hasher)

    if _, err := io.Copy(mw, r.Body); err != nil {
        os.Remove(path) // cleanup on error
        http.Error(w, "failed to write file", http.StatusInternalServerError)
        return
    }

    etag := hex.EncodeToString(hasher.Sum(nil))
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"etag": etag})
}

func (fs *FileStore) handleGet(w http.ResponseWriter, r *http.Request) {
    bucket, key, err := fs.parsePath(r.URL.Path)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    path := fs.fullPath(bucket, key)
    file, err := os.Open(path)
    if err != nil {
        if os.IsNotExist(err) {
            http.NotFound(w, r)
        } else {
            http.Error(w, "failed to open file", http.StatusInternalServerError)
        }
        return
    }
    defer file.Close()

    stat, err := file.Stat()
    if err != nil {
        http.Error(w, "failed to stat file", http.StatusInternalServerError)
        return
    }

    // Set headers
    w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
    w.Header().Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
    
    // Calculate ETag if needed
    if etag := fs.calculateETag(file); etag != "" {
        w.Header().Set("ETag", `"`+etag+`"`)
    }

    // Handle range requests
    if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
        fs.handleRangeRequest(w, r, file, stat.Size(), rangeHeader)
        return
    }

    // Reset file position after ETag calculation
    file.Seek(0, 0)
    http.ServeContent(w, r, key, stat.ModTime(), file)
}

func (fs *FileStore) handleHead(w http.ResponseWriter, r *http.Request) {
    bucket, key, err := fs.parsePath(r.URL.Path)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    path := fs.fullPath(bucket, key)
    file, err := os.Open(path)
    if err != nil {
        if os.IsNotExist(err) {
            http.NotFound(w, r)
        } else {
            http.Error(w, "failed to open file", http.StatusInternalServerError)
        }
        return
    }
    defer file.Close()

    stat, err := file.Stat()
    if err != nil {
        http.Error(w, "failed to stat file", http.StatusInternalServerError)
        return
    }

    // Set same headers as GET but no body
    w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
    w.Header().Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
    
    if etag := fs.calculateETag(file); etag != "" {
        w.Header().Set("ETag", `"`+etag+`"`)
    }

    w.WriteHeader(http.StatusOK)
}

func (fs *FileStore) handleDelete(w http.ResponseWriter, r *http.Request) {
    bucket, key, err := fs.parsePath(r.URL.Path)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    path := fs.fullPath(bucket, key)
    if err := os.Remove(path); err != nil {
        if os.IsNotExist(err) {
            http.NotFound(w, r)
        } else {
            http.Error(w, "failed to delete file", http.StatusInternalServerError)
        }
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (fs *FileStore) parsePath(path string) (bucket, key string, err error) {
    path = strings.Trim(path, "/")
    if path == "" {
        return "", "", fmt.Errorf("missing bucket and key")
    }

    parts := strings.SplitN(path, "/", 2)
    if len(parts) < 2 {
        return "", "", fmt.Errorf("missing key")
    }

    bucket, key = parts[0], parts[1]
    if bucket == "" {
        return "", "", fmt.Errorf("empty bucket name")
    }
    if key == "" {
        return "", "", fmt.Errorf("empty key name")
    }

    return bucket, key, nil
}

func (fs *FileStore) calculateETag(file *os.File) string {
    hasher := sha256.New()
    currentPos, _ := file.Seek(0, 1) // save current position
    file.Seek(0, 0)                 // go to start
    
    if _, err := io.Copy(hasher, file); err != nil {
        return ""
    }
    
    file.Seek(currentPos, 0) // restore position
    return hex.EncodeToString(hasher.Sum(nil))
}

func (fs *FileStore) handleRangeRequest(w http.ResponseWriter, r *http.Request, file *os.File, fileSize int64, rangeHeader string) {
    // Parse Range header: "bytes=start-end"
    if !strings.HasPrefix(rangeHeader, "bytes=") {
        http.Error(w, "invalid range header", http.StatusBadRequest)
        return
    }

    rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
    parts := strings.Split(rangeSpec, "-")
    if len(parts) != 2 {
        http.Error(w, "invalid range format", http.StatusBadRequest)
        return
    }

    var start, end int64
    var err error

    if parts[0] != "" {
        start, err = strconv.ParseInt(parts[0], 10, 64)
        if err != nil || start < 0 {
            http.Error(w, "invalid range start", http.StatusBadRequest)
            return
        }
    }

    if parts[1] != "" {
        end, err = strconv.ParseInt(parts[1], 10, 64)
        if err != nil || end < 0 {
            http.Error(w, "invalid range end", http.StatusBadRequest)
            return
        }
    } else {
        end = fileSize - 1
    }

    if start > end || start >= fileSize {
        w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
        http.Error(w, "range not satisfiable", http.StatusRequestedRangeNotSatisfiable)
        return
    }

    if end >= fileSize {
        end = fileSize - 1
    }

    contentLength := end - start + 1
    
    w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
    w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
    w.WriteHeader(http.StatusPartialContent)

    file.Seek(start, 0)
    io.CopyN(w, file, contentLength)
}
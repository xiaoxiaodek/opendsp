package filegateway

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/opendsp/opendsp/internal/data/dbsqlc"
)

func (s *Service) UploadReceiverHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/upload/")
		if path == r.URL.Path {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		ns, fileID := parts[0], parts[1]

		bucket, ok := namespaceBuckets[ns]
		if !ok {
			http.Error(w, "unknown namespace", http.StatusNotFound)
			return
		}

		storageKey := fmt.Sprintf("%s/%s/%s", bucket, fileID[:2], fileID)

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// Read entire body first (SeaweedFS S3 requires seekable stream)
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("read body: %v", err), http.StatusInternalServerError)
			return
		}

		hasher := md5.New()
		hasher.Write(bodyBytes)
		md5Sum := fmt.Sprintf("%x", hasher.Sum(nil))
		size := int64(len(bodyBytes))

		err = s.backend.PutObject(r.Context(), bucket, storageKey, bytes.NewReader(bodyBytes), size, contentType)
		if err != nil {
			http.Error(w, fmt.Sprintf("upload failed: %v", err), http.StatusInternalServerError)
			return
		}

		err = s.data.Queries.UpdateFileRecordReady(r.Context(), &dbsqlc.UpdateFileRecordReadyParams{
			ID:          fileID,
			Size:        &size,
			ContentType: &contentType,
			Md5:         &md5Sum,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("update record: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","file_id":"` + fileID + `","md5":"` + md5Sum + `"}`))
	})
}

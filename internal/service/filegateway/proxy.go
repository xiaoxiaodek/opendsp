package filegateway

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (s *Service) FileProxyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
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

		body, info, err := s.backend.GetObject(r.Context(), bucket, storageKey)
		if err != nil {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		defer body.Close()

		if info.ContentType != "" {
			w.Header().Set("Content-Type", info.ContentType)
		}
		if info.ContentLength > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(info.ContentLength, 10))
		}
		if info.ETag != "" {
			w.Header().Set("ETag", info.ETag)
		}
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")

		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}

		io.Copy(w, body)
	})
}

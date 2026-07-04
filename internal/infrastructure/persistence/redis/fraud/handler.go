package fraud

import (
	"encoding/json"
	"net/http"
	"strings"
)

// HTTPHandler provides REST endpoints for anti-fraud blacklist management.
type HTTPHandler struct {
	repo *BlacklistRepo
}

// NewHTTPHandler creates an anti-fraud HTTP handler.
func NewHTTPHandler(repo *BlacklistRepo) *HTTPHandler {
	return &HTTPHandler{repo: repo}
}

type blacklistEntry struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type blacklistResponse struct {
	Items []blacklistEntry `json:"items"`
}

// ServeHTTP routes anti-fraud API requests.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// /api/antifraud/blacklist -> list, add
	// /api/antifraud/blacklist/{type} -> list by type
	// /api/antifraud/blacklist/{type}/{value} -> delete

	path := strings.TrimPrefix(r.URL.Path, "/api/antifraud/blacklist")
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	switch {
	case r.Method == "GET" && len(path) == 0:
		h.listAll(w, r)
	case r.Method == "GET" && len(parts) >= 1 && parts[0] != "":
		h.listByType(w, r, parts[0])
	case r.Method == "POST" && len(path) == 0:
		h.add(w, r)
	case r.Method == "DELETE" && len(parts) >= 2:
		h.remove(w, r, parts[0], parts[1])
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *HTTPHandler) listAll(w http.ResponseWriter, r *http.Request) {
	var entries []blacklistEntry
	for _, t := range []string{"ip", "device", "ua", "geo"} {
		values, err := h.repo.ListBlacklist(r.Context(), t)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		for _, v := range values {
			entries = append(entries, blacklistEntry{Type: t, Value: v})
		}
	}
	writeJSON(w, http.StatusOK, blacklistResponse{Items: entries})
}

func (h *HTTPHandler) listByType(w http.ResponseWriter, r *http.Request, listType string) {
	values, err := h.repo.ListBlacklist(r.Context(), listType)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var entries []blacklistEntry
	for _, v := range values {
		entries = append(entries, blacklistEntry{Type: listType, Value: v})
	}
	writeJSON(w, http.StatusOK, blacklistResponse{Items: entries})
}

func (h *HTTPHandler) add(w http.ResponseWriter, r *http.Request) {
	var entry blacklistEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if entry.Type == "" || entry.Value == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type and value required"})
		return
	}
	if err := h.repo.AddToBlacklist(r.Context(), entry.Type, entry.Value); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (h *HTTPHandler) remove(w http.ResponseWriter, r *http.Request, listType, value string) {
	if err := h.repo.RemoveFromBlacklist(r.Context(), listType, value); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

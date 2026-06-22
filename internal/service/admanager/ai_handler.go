package admanager

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/opendsp/opendsp/internal/ai"
	"github.com/opendsp/opendsp/internal/middleware"
)

type AIHandler struct {
	chat    *ai.ChatService
	insight *ai.InsightService
}

func NewAIHandler(chat *ai.ChatService, insight *ai.InsightService) *AIHandler {
	return &AIHandler{chat: chat, insight: insight}
}

func (h *AIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/ai")

	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tokenStr == "" {
		writeJSON(w, 401, map[string]string{"error": "missing authorization"})
		return
	}
	claims, err := middleware.ParseToken(tokenStr)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "invalid token"})
		return
	}

	switch {
	case path == "/chat" && r.Method == "POST":
		h.handleNewChat(w, r, claims)
	case strings.HasPrefix(path, "/chat/") && strings.HasSuffix(path, "/confirm") && r.Method == "POST":
		h.handleConfirm(w, r, claims)
	case strings.HasPrefix(path, "/chat/") && r.Method == "POST":
		h.handleContinueChat(w, r, claims)
	case path == "/insights/dashboard" && r.Method == "GET":
		h.handleDashboardInsight(w, r, claims)
	case path == "/insights/report" && r.Method == "GET":
		h.handleReportInsight(w, r, claims)
	case path == "/insights/refresh" && r.Method == "POST":
		h.handleRefreshInsight(w, r, claims)
	default:
		http.Error(w, "not found", 404)
	}
}

func (h *AIHandler) handleNewChat(w http.ResponseWriter, r *http.Request, claims *middleware.Claims) {
	body, _ := io.ReadAll(r.Body)
	var req struct {
		Message string `json:"message"`
	}
	json.Unmarshal(body, &req)
	if req.Message == "" {
		writeJSON(w, 400, map[string]string{"error": "message is required"})
		return
	}

	sessionID, _ := h.chat.NewSession(claims.UserID, claims.AdvertiserID, claims.Role)
	h.streamChat(w, r, sessionID, req.Message)
}

func (h *AIHandler) handleContinueChat(w http.ResponseWriter, r *http.Request, claims *middleware.Claims) {
	sessionID := extractSessionID(r.URL.Path, "/api/v1/ai/chat/")
	if sessionID == "" {
		writeJSON(w, 400, map[string]string{"error": "invalid session id"})
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req struct {
		Message string `json:"message"`
	}
	json.Unmarshal(body, &req)
	if req.Message == "" {
		writeJSON(w, 400, map[string]string{"error": "message is required"})
		return
	}

	h.streamChat(w, r, sessionID, req.Message)
}

func (h *AIHandler) handleConfirm(w http.ResponseWriter, r *http.Request, claims *middleware.Claims) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/ai/chat/")
	sessionID := strings.TrimSuffix(path, "/confirm")
	if sessionID == "" {
		writeJSON(w, 400, map[string]string{"error": "invalid session id"})
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req struct {
		ToolCallID string `json:"tool_call_id"`
		Confirmed  bool   `json:"confirmed"`
	}
	json.Unmarshal(body, &req)

	eventCh, err := h.chat.ConfirmTool(r.Context(), sessionID, req.ToolCallID, req.Confirmed)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)

	for event := range eventCh {
		if event.Error != nil {
			fmt.Fprintf(w, "data: {\"error\":\"%s\"}\n\n", event.Error.Error())
			flusher.Flush()
			return
		}
		if event.Delta != "" {
			fmt.Fprintf(w, "data: %s\n\n", event.Delta)
			flusher.Flush()
		}
		if event.Done {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}
	}
}

func (h *AIHandler) streamChat(w http.ResponseWriter, r *http.Request, sessionID, message string) {
	eventCh, err := h.chat.Chat(r.Context(), sessionID, message)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)

	fmt.Fprintf(w, "data: {\"session_id\":\"%s\"}\n\n", sessionID)
	flusher.Flush()

	for event := range eventCh {
		if event.Error != nil {
			fmt.Fprintf(w, "data: {\"error\":\"%s\"}\n\n", event.Error.Error())
			flusher.Flush()
			return
		}
		if event.Delta != "" {
			fmt.Fprintf(w, "data: %s\n\n", event.Delta)
			flusher.Flush()
		}
		if event.Done {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}
	}
}

func (h *AIHandler) handleDashboardInsight(w http.ResponseWriter, r *http.Request, claims *middleware.Claims) {
	advertiserID := claims.AdvertiserID
	if idStr := r.URL.Query().Get("advertiser_id"); idStr != "" {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		if id > 0 {
			advertiserID = id
		}
	}
	insight, err := h.insight.GetDashboardInsight(r.Context(), advertiserID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, insight)
}

func (h *AIHandler) handleReportInsight(w http.ResponseWriter, r *http.Request, claims *middleware.Claims) {
	advertiserID := claims.AdvertiserID
	if idStr := r.URL.Query().Get("advertiser_id"); idStr != "" {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		if id > 0 {
			advertiserID = id
		}
	}
	startStr := r.URL.Query().Get("start_time")
	endStr := r.URL.Query().Get("end_time")

	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid start_time"})
		return
	}
	endTime, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid end_time"})
		return
	}

	anomalies, err := h.insight.GetReportAnomalies(r.Context(), advertiserID, startTime, endTime)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"anomalies": anomalies})
}

func (h *AIHandler) handleRefreshInsight(w http.ResponseWriter, r *http.Request, claims *middleware.Claims) {
	h.insight.Refresh(r.Context(), claims.AdvertiserID)
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func extractSessionID(path, prefix string) string {
	s := strings.TrimPrefix(path, prefix)
	s = strings.TrimRight(s, "/")
	return s
}

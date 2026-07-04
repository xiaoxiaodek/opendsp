package admanager

import (
	"net/http"
	"strconv"
	"time"

	"github.com/opendsp/opendsp/internal/infrastructure/persistence/clickhouse"
)

type SettlementHandler struct {
	chWriter *clickhouse.Writer
}

func NewSettlementHandler(writer *clickhouse.Writer) *SettlementHandler {
	return &SettlementHandler{chWriter: writer}
}

func (h *SettlementHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Path
	switch {
	case path == "/api/settlement/reconcile" && r.Method == "GET":
		h.handleReconcile(w, r)
	case path == "/api/settlement/upload" && r.Method == "POST":
		h.handleUpload(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *SettlementHandler) handleReconcile(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	advertiserID, _ := strconv.ParseInt(q.Get("advertiser_id"), 10, 64)

	startDate := q.Get("start_date")
	endDate := q.Get("end_date")
	if startDate == "" {
		startDate = time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	end = end.Add(24 * time.Hour)

	summary, err := h.chWriter.Reconcile(r.Context(), advertiserID, start, end)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (h *SettlementHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file"})
		return
	}
	defer file.Close()

	count, err := h.chWriter.ImportCSV(r.Context(), file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"imported": count,
	})
}

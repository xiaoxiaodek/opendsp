package fraud

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
)

// EventsHandler serves fraud_events query endpoints.
type EventsHandler struct {
	pool    *pgxpool.Pool
	queries *dbsqlc.Queries
}

// NewEventsHandler creates an events query handler.
func NewEventsHandler(pool *pgxpool.Pool) *EventsHandler {
	return &EventsHandler{pool: pool, queries: dbsqlc.New(pool)}
}

type fraudEventRow struct {
	ID        int64     `json:"id"`
	RequestID string    `json:"request_id"`
	RuleType  string    `json:"rule_type"`
	RuleValue string    `json:"rule_value"`
	RiskScore float64   `json:"risk_score"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

type eventsResponse struct {
	Items []fraudEventRow `json:"items"`
	Total int64           `json:"total"`
}

type statsResponse struct {
	TotalRequests int64   `json:"total_requests"`
	Blocked       int64   `json:"blocked"`
	Flagged       int64   `json:"flagged"`
	BlockRate     float64 `json:"block_rate"`
}

// ServeHTTP routes fraud_events API requests.
func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Path
	switch {
	case path == "/api/antifraud/events" && r.Method == "GET":
		h.listEvents(w, r)
	case path == "/api/antifraud/stats" && r.Method == "GET":
		h.getStats(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *EventsHandler) listEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := int32((page - 1) * pageSize)

	startDate, endDate := parseDateRange(q.Get("start_date"), q.Get("end_date"))

	total, err := h.queries.CountFraudEvents(r.Context(), &dbsqlc.CountFraudEventsParams{
		CreatedAt:   startDate,
		CreatedAt_2: endDate,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rows, err := h.queries.ListFraudEvents(r.Context(), &dbsqlc.ListFraudEventsParams{
		CreatedAt:   startDate,
		CreatedAt_2: endDate,
		Limit:       int32(pageSize),
		Offset:      offset,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var items []fraudEventRow
	for _, row := range rows {
		items = append(items, fraudEventRow{
			ID:        row.ID,
			RequestID: row.RequestID,
			RuleType:  row.RuleType,
			RuleValue: row.RuleValue,
			RiskScore: row.RiskScore,
			Action:    row.Action,
			CreatedAt: row.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, eventsResponse{Items: items, Total: total})
}

func (h *EventsHandler) getStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	startDate, endDate := parseDateRange(q.Get("start_date"), q.Get("end_date"))

	stats, err := h.queries.GetFraudEventStats(r.Context(), &dbsqlc.GetFraudEventStatsParams{
		CreatedAt:   startDate,
		CreatedAt_2: endDate,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	resp := statsResponse{
		TotalRequests: stats.Total,
		Blocked:       stats.Blocked,
		Flagged:       stats.Flagged,
	}
	if stats.Total > 0 {
		resp.BlockRate = float64(stats.Blocked) / float64(stats.Total) * 100
	}

	writeJSON(w, http.StatusOK, resp)
}

func parseDateRange(start, end string) (time.Time, time.Time) {
	now := time.Now()
	var startDate, endDate time.Time
	if start != "" {
		startDate, _ = time.Parse(time.RFC3339, start)
	}
	if startDate.IsZero() {
		startDate = now.Add(-24 * time.Hour)
	}
	if end != "" {
		endDate, _ = time.Parse(time.RFC3339, end)
	}
	if endDate.IsZero() {
		endDate = now
	}
	return startDate, endDate
}

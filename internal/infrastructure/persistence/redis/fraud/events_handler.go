package fraud

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EventsHandler serves fraud_events query endpoints.
type EventsHandler struct {
	pool *pgxpool.Pool
}

// NewEventsHandler creates an events query handler.
func NewEventsHandler(pool *pgxpool.Pool) *EventsHandler {
	return &EventsHandler{pool: pool}
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
	offset := (page - 1) * pageSize

	startDate := q.Get("start_date")
	endDate := q.Get("end_date")
	if startDate == "" {
		startDate = time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	}
	if endDate == "" {
		endDate = time.Now().Format(time.RFC3339)
	}

	var total int64
	err := h.pool.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM fraud_events WHERE created_at >= $1 AND created_at <= $2`,
		startDate, endDate,
	).Scan(&total)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, request_id, rule_type, rule_value, COALESCE(risk_score, 0), action, created_at
		 FROM fraud_events WHERE created_at >= $1 AND created_at <= $2
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		startDate, endDate, pageSize, offset,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var items []fraudEventRow
	for rows.Next() {
		var row fraudEventRow
		if err := rows.Scan(&row.ID, &row.RequestID, &row.RuleType, &row.RuleValue, &row.RiskScore, &row.Action, &row.CreatedAt); err != nil {
			continue
		}
		items = append(items, row)
	}

	writeJSON(w, http.StatusOK, eventsResponse{Items: items, Total: total})
}

func (h *EventsHandler) getStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	startDate := q.Get("start_date")
	endDate := q.Get("end_date")
	if startDate == "" {
		startDate = time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	}
	if endDate == "" {
		endDate = time.Now().Format(time.RFC3339)
	}

	var stats statsResponse
	err := h.pool.QueryRow(r.Context(),
		`SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE action = 'blocked') as blocked,
			COUNT(*) FILTER (WHERE action = 'flagged') as flagged
		 FROM fraud_events WHERE created_at >= $1 AND created_at <= $2`,
		startDate, endDate,
	).Scan(&stats.TotalRequests, &stats.Blocked, &stats.Flagged)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if stats.TotalRequests > 0 {
		stats.BlockRate = float64(stats.Blocked) / float64(stats.TotalRequests) * 100
	}

	writeJSON(w, http.StatusOK, stats)
}

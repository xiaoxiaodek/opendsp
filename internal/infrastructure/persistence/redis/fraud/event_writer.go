package fraud

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FraudEvent represents a fraud detection event to be persisted.
type FraudEvent struct {
	RequestID string
	RuleType  string
	RuleValue string
	RiskScore float64
	Action    string
}

// EventWriter writes fraud events to the PostgreSQL fraud_events table.
type EventWriter struct {
	pool *pgxpool.Pool
}

// NewEventWriter creates an EventWriter backed by a pgxpool.
func NewEventWriter(pool *pgxpool.Pool) *EventWriter {
	return &EventWriter{pool: pool}
}

// Write inserts a fraud event into fraud_events.
func (w *EventWriter) Write(ctx context.Context, event FraudEvent) {
	_, err := w.pool.Exec(ctx,
		`INSERT INTO fraud_events (request_id, rule_type, rule_value, risk_score, action, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		event.RequestID, event.RuleType, event.RuleValue, event.RiskScore, event.Action, time.Now(),
	)
	if err != nil {
		_ = err
	}
}

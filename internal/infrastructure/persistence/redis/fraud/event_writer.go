package fraud

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
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
	pool    *pgxpool.Pool
	queries *dbsqlc.Queries
}

// NewEventWriter creates an EventWriter backed by a pgxpool.
func NewEventWriter(pool *pgxpool.Pool) *EventWriter {
	return &EventWriter{pool: pool, queries: dbsqlc.New(pool)}
}

// Write inserts a fraud event into fraud_events.
func (w *EventWriter) Write(ctx context.Context, event FraudEvent) {
	var riskScore pgtype.Numeric
	_ = riskScore.Scan(event.RiskScore)
	_ = w.queries.InsertFraudEvent(ctx, &dbsqlc.InsertFraudEventParams{
		RequestID: event.RequestID,
		RuleType:  event.RuleType,
		RuleValue: event.RuleValue,
		RiskScore: riskScore,
		Action:    event.Action,
		CreatedAt: time.Now(),
	})
}

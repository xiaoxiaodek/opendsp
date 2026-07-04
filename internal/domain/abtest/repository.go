package abtest

import (
	"context"
	"time"
)

// ExperimentRepo manages A/B test experiment definitions.
type ExperimentRepo interface {
	// GetRunning returns all currently running experiments.
	GetRunning(ctx context.Context) ([]Experiment, error)

	// GetByID returns a single experiment by ID.
	GetByID(ctx context.Context, id int64) (*Experiment, error)
}

// MetricRepo stores and queries A/B test metrics.
type MetricRepo interface {
	// RecordMetric stores a metric data point for a variant.
	RecordMetric(ctx context.Context, metric Metric) error

	// QueryMetrics returns metrics for an experiment over a date range.
	QueryMetrics(ctx context.Context, experimentID int64, start, end time.Time) ([]Metric, error)
}

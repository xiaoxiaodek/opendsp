package admanager

import (
	"context"
	"log"
	"time"

	"github.com/opendsp/opendsp/internal/data"
)

const (
	rawEventRetention = 48 * time.Hour
	partitionTTL      = 90 * 24 * time.Hour
)

type ReportAggregator struct {
	data *data.Data
}

func NewReportAggregator(d *data.Data) *ReportAggregator {
	return &ReportAggregator{data: d}
}

func (a *ReportAggregator) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	a.aggregate(ctx)
	a.ensurePartitions(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.aggregate(ctx)
			a.ensurePartitions(ctx)
			a.cleanup(ctx)
		}
	}
}

func (a *ReportAggregator) aggregate(ctx context.Context) {
	now := time.Now()
	start := now.Truncate(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	repo := data.NewReportRepo(a.data)
	if err := repo.AggregateHourly(ctx, start, end); err != nil {
		log.Printf("report aggregation error: %v", err)
	} else {
		log.Printf("report aggregated: %s - %s", start.Format(time.RFC3339), end.Format(time.RFC3339))
	}
}

func (a *ReportAggregator) ensurePartitions(ctx context.Context) {
	now := time.Now()
	for i := 0; i < 3; i++ {
		month := now.AddDate(0, i, 0)
		_, err := a.data.Pool.Exec(ctx,
			"SELECT ensure_stat_event_partition($1)",
			month,
		)
		if err != nil {
			log.Printf("ensure partition for %s: %v", month.Format("2006-01"), err)
		}
	}
}

func (a *ReportAggregator) cleanup(ctx context.Context) {
	deleteBefore := time.Now().Add(-rawEventRetention)
	tag, err := a.data.Pool.Exec(ctx,
		"SELECT delete_aggregated_events($1)",
		deleteBefore,
	)
	if err != nil {
		log.Printf("delete raw events: %v", err)
	} else {
		log.Printf("deleted raw events before %s: %d rows", deleteBefore.Format(time.RFC3339), tag.RowsAffected())
	}

	_, err = a.data.Pool.Exec(ctx,
		"SELECT cleanup_stat_event_partitions($1)",
		int(partitionTTL.Hours()/24),
	)
	if err != nil {
		log.Printf("cleanup partitions: %v", err)
	}
}

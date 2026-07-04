package bidding

import (
	"context"
	"fmt"
	"time"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	stageDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dsp_pipeline_stage_duration_ms",
		Help:    "Pipeline stage duration in milliseconds",
		Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 10, 50},
	}, []string{"stage"})

	stageDropped = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dsp_pipeline_stage_dropped_total",
		Help: "Total candidates dropped by pipeline stage",
	}, []string{"stage", "reason"})

	stageErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dsp_pipeline_stage_errors_total",
		Help: "Total errors in pipeline stage",
	}, []string{"stage"})
)

// Pipeline orchestrates ordered stages for bid processing.
type Pipeline struct {
	preMatch  []Stage // stages that run before index matching (e.g., anti-fraud)
	postMatch []Stage // stages that run after index matching
}

// NewPipeline creates a Pipeline with pre-match and post-match stages.
func NewPipeline(preMatch, postMatch []Stage) *Pipeline {
	return &Pipeline{
		preMatch:  preMatch,
		postMatch: postMatch,
	}
}

// RunPreMatch executes pre-match stages. Returns false if the request should be aborted.
func (p *Pipeline) RunPreMatch(ctx context.Context, req *bidding.BidRequest) bool {
	for _, stage := range p.preMatch {
		start := time.Now()
		candidates, err := stage.Process(ctx, req, nil)
		elapsed := float64(time.Since(start).Microseconds()) / 1000.0
		stageDuration.WithLabelValues(stage.Name()).Observe(elapsed)

		if err != nil {
			stageErrors.WithLabelValues(stage.Name()).Inc()
			continue
		}
		if candidates == nil {
			stageDropped.WithLabelValues(stage.Name(), "abort").Inc()
			return false
		}
	}
	return true
}

// RunPostMatch executes post-match stages on candidates.
// Returns the filtered candidates and whether to abort.
func (p *Pipeline) RunPostMatch(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, bool) {
	if len(candidates) == 0 {
		return candidates, false
	}
	for _, stage := range p.postMatch {
		before := len(candidates)
		start := time.Now()

		var err error
		candidates, err = stage.Process(ctx, req, candidates)
		elapsed := float64(time.Since(start).Microseconds()) / 1000.0
		stageDuration.WithLabelValues(stage.Name()).Observe(elapsed)

		if err != nil {
			stageErrors.WithLabelValues(stage.Name()).Inc()
			continue
		}

		if len(candidates) == 0 {
			stageDropped.WithLabelValues(stage.Name(), fmt.Sprintf("all_%d_dropped", before)).Inc()
			return nil, true
		}

		dropped := before - len(candidates)
		if dropped > 0 {
			stageDropped.WithLabelValues(stage.Name(), "filtered").Add(float64(dropped))
		}
	}
	return candidates, false
}

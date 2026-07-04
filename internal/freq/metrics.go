package freq

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce     sync.Once
	globalMetrics   *BucketMetrics
)

// BucketMetrics holds all Prometheus metrics for the token bucket system.
type BucketMetrics struct {
	reserveTotal    *prometheus.CounterVec
	confirmTotal    *prometheus.CounterVec
	releaseTotal    *prometheus.CounterVec
	reservedTokens  *prometheus.GaugeVec
	budgetConsumed  *prometheus.GaugeVec
	availableTokens *prometheus.GaugeVec
	refillRate      *prometheus.GaugeVec
	reserveLatency  prometheus.Histogram
}

// NewBucketMetrics creates and registers all Prometheus metrics (singleton).
func NewBucketMetrics() *BucketMetrics {
	metricsOnce.Do(func() {
		globalMetrics = &BucketMetrics{
			reserveTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "dsp_budget_reserve_total",
				Help: "Total number of token reservation attempts.",
			}, []string{"campaign_id", "adgroup_id", "status"}),

			confirmTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "dsp_budget_confirm_total",
				Help: "Total number of confirmed reservations.",
			}, []string{"campaign_id", "adgroup_id", "status"}),

			releaseTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "dsp_budget_release_total",
				Help: "Total number of released reservations.",
			}, []string{"campaign_id", "adgroup_id", "reason"}),

			reservedTokens: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: "dsp_budget_reserved_tokens",
				Help: "Current number of reserved (unconfirmed) tokens.",
			}, []string{"campaign_id"}),

			budgetConsumed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: "dsp_budget_consumed_cents",
				Help: "Total confirmed budget consumption in cents.",
			}, []string{"campaign_id"}),

			availableTokens: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: "dsp_budget_available_tokens",
				Help: "Current available tokens per bucket.",
			}, []string{"campaign_id", "bucket_id"}),

			refillRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: "dsp_pacing_refill_rate",
				Help: "Current refill rate in cents per second.",
			}, []string{"campaign_id"}),

			reserveLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
				Name:    "dsp_budget_reserve_latency_seconds",
				Help:    "Latency of token reservation operations.",
				Buckets: []float64{.0001, .0005, .001, .002, .005, .01, .02, .05, .1},
			}),
		}

		// Register all metrics; panic on duplicate (should not happen with sync.Once)
		prometheus.MustRegister(
			globalMetrics.reserveTotal,
			globalMetrics.confirmTotal,
			globalMetrics.releaseTotal,
			globalMetrics.reservedTokens,
			globalMetrics.budgetConsumed,
			globalMetrics.availableTokens,
			globalMetrics.refillRate,
			globalMetrics.reserveLatency,
		)
	})
	return globalMetrics
}

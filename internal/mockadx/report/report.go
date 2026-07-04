package report

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/opendsp/opendsp/internal/mockadx/config"
	"github.com/opendsp/opendsp/internal/mockadx/funnel"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	bidRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mockadx_bid_requests_total",
		Help: "Total bid requests sent by mockadx",
	}, []string{"protocol", "profile", "status"})

	bidLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mockadx_bid_latency_seconds",
		Help:    "Bid request latency in seconds",
		Buckets: []float64{.001, .002, .005, .01, .02, .05, .1, .2, .5},
	}, []string{"protocol"})

	funnelEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mockadx_funnel_events_total",
		Help: "Funnel events by stage",
	}, []string{"stage"})
)

type Sample struct {
	TS    int64   `json:"ts"`
	QPS   float64 `json:"qps"`
	P50   float64 `json:"p50"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	Total int64   `json:"total"`
}

type Summary struct {
	TotalRequests      int64               `json:"total_requests"`
	TotalBids          int64               `json:"total_bids"`
	AvgQPS             float64             `json:"avg_qps"`
	LatencyP50         float64             `json:"latency_p50"`
	LatencyP95         float64             `json:"latency_p95"`
	LatencyP99         float64             `json:"latency_p99"`
	StatusDistribution map[string]int64    `json:"status_distribution"`
	Funnel             funnel.StoreSnapshot `json:"funnel"`
}

type Report struct {
	Config  config.Config `json:"config"`
	Summary Summary       `json:"summary"`
	Samples []Sample      `json:"samples"`
}

type Reporter struct {
	mu            sync.Mutex
	cfg           config.Config
	latencies     []float64
	statusCounts  map[string]int64
	totalRequests int64
	totalBids     int64
	startTime     time.Time
	samples       []Sample
	lastTotal     int64
	lastTime      time.Time
	store         *funnel.Store
}

func NewReporter(cfg config.Config, store *funnel.Store) *Reporter {
	return &Reporter{
		cfg:          cfg,
		statusCounts: make(map[string]int64),
		startTime:    time.Now(),
		store:        store,
		lastTime:     time.Now(),
	}
}

func (r *Reporter) RecordLatency(latency time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.latencies = append(r.latencies, latency.Seconds())
}

func (r *Reporter) RecordStatus(status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statusCounts[status]++
	r.totalRequests++
}

func (r *Reporter) RecordBid() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.totalBids++
}

func (r *Reporter) TotalRequests() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.totalRequests
}

func (r *Reporter) Run(stopCh <-chan struct{}, doneCh chan<- struct{}) {
	ticker := time.NewTicker(r.cfg.Report.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.printStatus()
		case <-stopCh:
			r.printStatus()
			r.writeJSON()
			close(doneCh)
			return
		}
	}
}

func (r *Reporter) printStatus() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastTime).Seconds()
	currentTotal := r.totalRequests
	qps := float64(currentTotal-r.lastTotal) / elapsed

	r.lastTotal = currentTotal
	r.lastTime = now

	p50, p95, p99 := r.percentiles()

	snap := r.store.Snapshot()

	fmt.Printf("\r[%s] QPS: %.0f | Latency p50/p95/p99: %.0f/%.0f/%.0fms | Status: 200:%d 204:%d errors:%d\n",
		now.Format("2006-01-02 15:04:05"),
		qps,
		p50*1000, p95*1000, p99*1000,
		r.statusCounts["200"],
		r.statusCounts["204"],
		r.statusCounts["error"],
	)
	fmt.Printf("                      Funnel: bid=%d win=%d imp=%d click=%d conv=%d\n",
		snap.BidCount, snap.WinCount, snap.ImpCount, snap.ClickCount, snap.ConvCount)

	r.samples = append(r.samples, Sample{
		TS:    now.Unix(),
		QPS:   qps,
		P50:   p50,
		P95:   p95,
		P99:   p99,
		Total: currentTotal,
	})
}

func (r *Reporter) percentiles() (p50, p95, p99 float64) {
	if len(r.latencies) == 0 {
		return 0, 0, 0
	}
	sorted := make([]float64, len(r.latencies))
	copy(sorted, r.latencies)
	sortFloat64s(sorted)

	p50 = sorted[len(sorted)/2]
	p95 = sorted[int(float64(len(sorted))*0.95)]
	p99 = sorted[int(float64(len(sorted))*0.99)]
	return
}

func (r *Reporter) writeJSON() {
	r.mu.Lock()
	defer r.mu.Unlock()

	p50, p95, p99 := r.percentiles()
	snap := r.store.Snapshot()

	report := Report{
		Config: r.cfg,
		Summary: Summary{
			TotalRequests:     r.totalRequests,
			TotalBids:         r.totalBids,
			AvgQPS:            float64(r.totalRequests) / time.Since(r.startTime).Seconds(),
			LatencyP50:        p50,
			LatencyP95:        p95,
			LatencyP99:        p99,
			StatusDistribution: r.statusCounts,
			Funnel:             snap,
		},
		Samples: r.samples,
	}

	data, _ := json.MarshalIndent(report, "", "  ")
	os.WriteFile(r.cfg.Report.Output, data, 0644)
	fmt.Printf("\nReport written to %s\n", r.cfg.Report.Output)
}

func sortFloat64s(a []float64) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}
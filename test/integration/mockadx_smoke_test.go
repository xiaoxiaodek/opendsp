package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opendsp/opendsp/internal/mockadx/config"
	"github.com/opendsp/opendsp/internal/mockadx/encoder"
	"github.com/opendsp/opendsp/internal/mockadx/funnel"
	"github.com/opendsp/opendsp/internal/mockadx/generator"
	"github.com/opendsp/opendsp/internal/mockadx/report"
	"github.com/opendsp/opendsp/internal/mockadx/sender"
)

func TestMockADX_Smoke(t *testing.T) {
	mockDSP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.Write([]byte{})
	}))
	defer mockDSP.Close()

	cfg := config.DefaultConfig()
	cfg.Target = mockDSP.URL
	cfg.Endpoint = "/rtb/iqiyi"
	cfg.Duration = 3 * time.Second
	cfg.QPS = 100
	cfg.Concurrency = 10
	cfg.Scenario.Profile = "hot-user"
	cfg.Scenario.HotUserPool = 5
	cfg.Funnel.WinRate = 0
	cfg.Funnel.ImpRate = 0
	cfg.Funnel.ClickRate = 0
	cfg.Funnel.ConvRate = 0

	rand := generator.NewRandomizer(42)
	gen := generator.NewScenarioMux(&cfg.Scenario, rand)
	enc := encoder.NewIqiyiEncoder(false)
	store := funnel.NewStore(5 * time.Minute)
	reporter := report.NewReporter(*cfg, store)

	pool := sender.NewPool(*cfg, gen, enc, store, reporter)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	pool.Start(ctx)
	<-ctx.Done()
	pool.Stop()

	total := reporter.TotalRequests()
	if total == 0 {
		t.Error("expected at least some requests to be sent")
	}
	fmt.Printf("smoke test: total_requests=%d\n", total)
}
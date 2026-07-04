package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/opendsp/opendsp/internal/mockadx/config"
	"github.com/opendsp/opendsp/internal/mockadx/encoder"
	"github.com/opendsp/opendsp/internal/mockadx/funnel"
	"github.com/opendsp/opendsp/internal/mockadx/generator"
	"github.com/opendsp/opendsp/internal/mockadx/report"
	"github.com/opendsp/opendsp/internal/mockadx/sender"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	configPath := flag.String("config", "config/mockadx.yaml", "path to config file")
	protocol := flag.String("protocol", "", "protocol: iqiyi or openrtb")
	target := flag.String("target", "", "DSP target URL")
	duration := flag.Duration("duration", 0, "test duration")
	qps := flag.Int("qps", 0, "target QPS")
	concurrency := flag.Int("concurrency", 0, "number of concurrent workers")
	profile := flag.String("profile", "", "scenario profile: hot-user, long-tail, peak, mixed")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("config load: %v, using defaults", err)
		cfg = config.DefaultConfig()
	}

	if *protocol != "" {
		cfg.Protocol = *protocol
	}
	if *target != "" {
		cfg.Target = *target
	}
	if *duration > 0 {
		cfg.Duration = *duration
	}
	if *qps > 0 {
		cfg.QPS = *qps
	}
	if *concurrency > 0 {
		cfg.Concurrency = *concurrency
	}
	if *profile != "" {
		cfg.Scenario.Profile = *profile
	}

	switch cfg.Protocol {
	case "iqiyi":
		cfg.Endpoint = "/rtb/iqiyi"
	case "openrtb":
		cfg.Endpoint = "/rtb/openrtb"
	default:
		log.Fatalf("unknown protocol: %s", cfg.Protocol)
	}

	log.Printf("mockadx starting: protocol=%s target=%s qps=%d concurrency=%d duration=%s profile=%s",
		cfg.Protocol, cfg.Target, cfg.QPS, cfg.Concurrency, cfg.Duration, cfg.Scenario.Profile)

	rand := generator.NewRandomizer(time.Now().UnixNano())
	gen := generator.NewScenarioMux(&cfg.Scenario, rand)

	var enc encoder.Encoder
	switch cfg.Protocol {
	case "iqiyi":
		enc = encoder.NewIqiyiEncoder(cfg.Gzip)
	case "openrtb":
		enc = encoder.NewOpenRTBEncoder(cfg.Gzip)
	}

	funnelStore := funnel.NewStore(5 * time.Minute)
	reporter := report.NewReporter(*cfg, funnelStore)

	senderPool := sender.NewPool(*cfg, gen, enc, funnelStore, reporter)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	senderPool.Start(ctx)

	mux := http.NewServeMux()
	mux.Handle(cfg.Receiver.MetricsPath, promhttp.Handler())
	httpServer := &http.Server{
		Addr:    cfg.Receiver.Listen,
		Handler: mux,
	}
	go func() {
		log.Printf("metrics server listening on %s", cfg.Receiver.Listen)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("metrics server: %v", err)
		}
	}()

	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	go reporter.Run(stopCh, doneCh)

	<-ctx.Done()
	log.Println("duration reached, stopping sender...")
	senderPool.Stop()

	close(stopCh)
	<-doneCh

	fmt.Println("mockadx finished")
	os.Exit(0)
}
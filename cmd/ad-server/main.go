package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data"
	"github.com/opendsp/opendsp/internal/dmp"
	"github.com/opendsp/opendsp/internal/freq"
	"github.com/opendsp/opendsp/internal/index"
	adserversvc "github.com/opendsp/opendsp/internal/service/adserver"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	ctx := context.Background()

	d, cleanup, err := data.NewData(ctx)
	if err != nil {
		log.Fatalf("data: %v", err)
	}
	defer cleanup()

	freqCtrl := freq.NewController(d.Rdb)

	dmpRepo := data.NewDmpRepo(d)
	tagStore := dmp.NewTagStore(d.Rdb)
	behaviorCollector := dmp.NewBehaviorCollector(d.Rdb, dmpRepo, tagStore)
	go behaviorCollector.RunAggregation(ctx)

	idx := index.New()
	if err := idx.BuildFromDB(ctx, d); err != nil {
		log.Fatalf("build index: %v", err)
	}
	log.Printf("index built: %d ad groups, version %d", idx.AdCount(), idx.Version())

	engine := adserversvc.NewEngine(idx, freqCtrl)
	tracker := adserversvc.NewTracker(freqCtrl, d)
	attrTracker := adserversvc.NewAttributionTracker(d)
	srv := adserversvc.NewServer(engine, tracker)

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := idx.BuildFromDB(ctx, d); err != nil {
				log.Printf("index refresh error: %v", err)
			} else {
				log.Printf("index refreshed: %d ad groups, version %d", idx.AdCount(), idx.Version())
			}
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/", srv)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/postback", attrTracker.HandlePostback)
	mux.HandleFunc("/postback/batch", attrTracker.HandlePostbackBatch)
	mux.HandleFunc("/postback/mmp", attrTracker.HandleMMPPostback)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 50 * time.Millisecond,
		ReadTimeout:       200 * time.Millisecond,
		WriteTimeout:      500 * time.Millisecond,
		IdleTimeout:       30 * time.Second,
	}
	go func() {
		log.Printf("ad-server listening on :%s", port)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpServer.Shutdown(shutdownCtx)

	_ = biz.CampaignStatusActive
}

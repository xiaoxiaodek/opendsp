package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	appbidding "github.com/opendsp/opendsp/internal/application/bidding"
	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/config"
	"github.com/opendsp/opendsp/internal/data"
	"github.com/opendsp/opendsp/internal/dmp"
	"github.com/opendsp/opendsp/internal/domain/bidding"
	domainFraud "github.com/opendsp/opendsp/internal/domain/fraud"
	"github.com/opendsp/opendsp/internal/freq"
	"github.com/opendsp/opendsp/internal/index"
	rtaClient "github.com/opendsp/opendsp/internal/infrastructure/external/rta"
	onnxPredictor "github.com/opendsp/opendsp/internal/infrastructure/ml/onnx"
	abtestInfra "github.com/opendsp/opendsp/internal/infrastructure/persistence/redis/abtest"
	budgetGuard "github.com/opendsp/opendsp/internal/infrastructure/persistence/redis/budget"
	dpaInfra "github.com/opendsp/opendsp/internal/infrastructure/persistence/redis/dpa"
	featureRepo "github.com/opendsp/opendsp/internal/infrastructure/persistence/redis/feature"
	fraudRepo "github.com/opendsp/opendsp/internal/infrastructure/persistence/redis/fraud"
	adserversvc "github.com/opendsp/opendsp/internal/service/adserver"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	ctx := context.Background()

	cfg, dyn, err := config.Load("config/app.yaml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	_ = dyn // for future dynamic config use

	d, cleanup, err := data.NewData(ctx, cfg.Database, cfg.Redis)
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

	// Sync advertiser balances from DB to Redis so the budget guard can read them
	if err := syncBalancesToRedis(ctx, d); err != nil {
		log.Printf("WARNING: sync balances to redis: %v", err)
	} else {
		log.Printf("advertiser balances synced to redis")
	}

	// --- Pipeline setup ---
	pCfg := cfg.Pipeline

	fraudSvc := fraudRepo.NewBlacklistRepo(d.Rdb)
	featRepo := featureRepo.NewFeatureRepo(d.Rdb)
	fallbackPredictor := onnxPredictor.NewFallbackPredictor(
		pCfg.Scoring.FallbackCTR,
		pCfg.Scoring.FallbackCVR,
	)
	scoringSvc := onnxPredictor.NewPredictor(
		nil, // ONNX session not yet created
		fallbackPredictor,
		pCfg.Scoring.FeatureOrder,
	)
	budgetSvc := budgetGuard.NewBudgetGuard(freqCtrl, d.Rdb)

	// Sliding window
	swCfg := pCfg.AntiFraud.SlidingWindow
	if swCfg.Enabled {
		sw := fraudRepo.NewSlidingWindow(d.Rdb, domainFraud.SlidingWindowConfig{
			Enabled: swCfg.Enabled,
			RequestRate: domainFraud.RequestRateConfig{
				WindowMs:       swCfg.RequestRate.WindowMs,
				MaxIPCount:     swCfg.RequestRate.MaxIPCount,
				MaxDeviceCount: swCfg.RequestRate.MaxDeviceCount,
			},
			CTRAnomaly: domainFraud.CTRAnomalyConfig{
				WindowMs:  swCfg.CTRAnomaly.WindowMs,
				MaxCTRPct: swCfg.CTRAnomaly.MaxCTRPct,
			},
			DeviceDiversity: domainFraud.DeviceDiversityConfig{
				WindowMs:     swCfg.DeviceDiversity.WindowMs,
				MaxIPChanges: swCfg.DeviceDiversity.MaxIPChanges,
				MaxUAChanges: swCfg.DeviceDiversity.MaxUAChanges,
			},
			DynamicBlacklistTTLMs: swCfg.DynamicBlacklistTTLMs,
		})
		fraudSvc = fraudRepo.NewBlacklistRepoWithSlidingWindow(d.Rdb, sw)
	}

	abtestSvc := abtestInfra.NewAssignmentService(d.Rdb)
	dpaSvc := dpaInfra.NewRetargetingService(d.Rdb)
	multiplierStore := budgetGuard.NewOXBIMultiplierStore(d.Rdb)

	// Pre-match stages
	preMatch := []appbidding.Stage{
		appbidding.NewABTestStage(abtestSvc),
		appbidding.NewAntiFraudStage(fraudSvc, pCfg.AntiFraud.Threshold),
	}

	// RTA
	if pCfg.RTA.Enabled && len(cfg.RTA.Advertisers) > 0 {
		registry := rtaClient.NewRegistryFromConfig(cfg.RTA)
		rtaSvc := rtaClient.NewClient(registry)
		preMatch = append(preMatch, appbidding.NewRTAStage(rtaSvc, pCfg.RTA.TimeoutMs))
	}

	// Post-match stages
	postMatch := []appbidding.Stage{
		appbidding.NewCoarseRankingStage(pCfg.CoarseRanking.MaxCandidates, bidding.LRModel{
			Intercept: pCfg.CoarseRanking.Model.Intercept,
			Weights:   pCfg.CoarseRanking.Model.Weights,
		}),
		appbidding.NewFeatureAssemblyStage(featRepo),
		appbidding.NewScoringStage(scoringSvc),
		appbidding.NewDPAStage(dpaSvc),
		appbidding.NewPricingStage(bidding.PricingStrategy(pCfg.Pricing.Strategy), pCfg.Pricing.OXBITargetROAS, multiplierStore),
		appbidding.NewPacingStage(budgetGuard.NewPacingService(d.Rdb)),
		appbidding.NewBudgetGuardStage(budgetSvc),
	}

	pipeline := appbidding.NewPipeline(preMatch, postMatch)

	engine := adserversvc.NewEngine(idx, freqCtrl, pipeline)
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
			if err := syncBalancesToRedis(ctx, d); err != nil {
				log.Printf("balance sync error: %v", err)
			}
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/", srv)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/postback", attrTracker.HandlePostback)
	mux.HandleFunc("/postback/batch", attrTracker.HandlePostbackBatch)
	mux.HandleFunc("/postback/mmp", attrTracker.HandleMMPPostback)

	port := strconv.Itoa(cfg.Server.Port)

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

// syncBalancesToRedis loads all advertiser balances from the DB and writes them
// to Redis (in cents) so the budget guard can atomically check/decrement.
func syncBalancesToRedis(ctx context.Context, d *data.Data) error {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, COALESCE(balance, 0) AS balance FROM advertiser WHERE status = 1`)
	if err != nil {
		return fmt.Errorf("query advertisers: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var balance float64
		if err := rows.Scan(&id, &balance); err != nil {
			continue
		}
		cents := int64(balance * 100)
		key := fmt.Sprintf("balance:%d", id)
		if err := d.Rdb.Set(ctx, key, cents, 0).Err(); err != nil {
			return fmt.Errorf("set balance %d: %w", id, err)
		}
		count++
	}
	return rows.Err()
}

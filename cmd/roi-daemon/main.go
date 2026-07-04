package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opendsp/opendsp/internal/config"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
	"github.com/opendsp/opendsp/internal/infrastructure/pid"
	"github.com/redis/go-redis/v9"
)

type ROIMetrics struct {
	AdvertiserID  int64
	CampaignID    int64
	CostMicros    int64
	RevenueMicros int64
}

func main() {
	cfg, _, err := config.Load("config/app.yaml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pgURL := cfg.Database.URL
	redisAddr := cfg.Redis.Addr
	targetROAS := cfg.ROI.OXBITargetROAS
	kp := cfg.ROI.PID.Kp
	ki := cfg.ROI.PID.Ki
	kd := cfg.ROI.PID.Kd
	intervalSec := cfg.ROI.IntervalSec

	pool, err := pgxpool.New(context.Background(), pgURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runLoop(ctx, pool, rdb, targetROAS, kp, ki, kd, time.Duration(intervalSec)*time.Second)

	log.Println("roi-daemon: running")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("roi-daemon: shutting down...")
}

func runLoop(ctx context.Context, pool *pgxpool.Pool, rdb *redis.Client, targetROAS, kp, ki, kd float64, interval time.Duration) {
	controllers := make(map[string]*pid.Controller)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			adjustBids(ctx, pool, rdb, controllers, targetROAS, kp, ki, kd)
		}
	}
}

func adjustBids(ctx context.Context, pool *pgxpool.Pool, rdb *redis.Client, controllers map[string]*pid.Controller, targetROAS, kp, ki, kd float64) {
	queries := dbsqlc.New(pool)

	var pgDate pgtype.Date
	_ = pgDate.Scan(time.Now().Add(-24 * time.Hour).Format("2006-01-02"))

	rows, err := queries.ListROIMetricsByAdvertiser(ctx, pgDate)
	if err != nil {
		log.Printf("roi-daemon: query roi_metrics: %v", err)
		return
	}

	for _, row := range rows {
		actualROAS := float64(row.RevenueMicros) / float64(row.CostMicros)
		key := fmt.Sprintf("%d:%d", row.AdvertiserID, row.CampaignID)

		ctrl, ok := controllers[key]
		if !ok {
			ctrl = pid.NewController(kp, ki, kd, targetROAS)
			controllers[key] = ctrl
		}

		adjustment := ctrl.Adjust(actualROAS)
		if adjustment == 0 {
			continue
		}

		redisKey := fmt.Sprintf("oxbi:multiplier:%s", key)
		currentMultiplier, err := rdb.Get(ctx, redisKey).Float64()
		if err != nil || currentMultiplier == 0 {
			currentMultiplier = 1.0
		}

		newMultiplier := currentMultiplier * (1 + adjustment)
		if newMultiplier > 2.0 {
			newMultiplier = 2.0
		}
		if newMultiplier < 0.5 {
			newMultiplier = 0.5
		}

		if err := rdb.Set(ctx, redisKey, newMultiplier, 0).Err(); err != nil {
			log.Printf("roi-daemon: write multiplier: %v", err)
		}
	}
}

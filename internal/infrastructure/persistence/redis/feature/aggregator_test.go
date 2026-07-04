package feature

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestAggregator_UserFeatures(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	buf := make(chan RawEvent, 10)
	cfg := DefaultAggregatorConfig()
	cfg.AggregateInterval = 10 * time.Millisecond
	agg := NewAggregator(rdb, buf, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agg.Run(ctx)

	buf <- RawEvent{EventType: "impression", UserID: "u1", Timestamp: time.Now().UnixMilli()}
	buf <- RawEvent{EventType: "impression", UserID: "u1", Timestamp: time.Now().UnixMilli()}
	buf <- RawEvent{EventType: "click", UserID: "u1", Timestamp: time.Now().UnixMilli()}
	buf <- RawEvent{EventType: "bid", UserID: "u1", CTR: 0.03, CVR: 0.01, Timestamp: time.Now().UnixMilli()}

	time.Sleep(50 * time.Millisecond)
	cancel()

	features, err := rdb.HGetAll(context.Background(), "feature:user:u1").Result()
	if err != nil {
		t.Fatalf("read features: %v", err)
	}
	if features["impressions_1h"] != "2" {
		t.Errorf("expected impressions_1h=2, got %s", features["impressions_1h"])
	}
	if features["clicks_1h"] != "1" {
		t.Errorf("expected clicks_1h=1, got %s", features["clicks_1h"])
	}
	if features["ctr_24h"] != "0.030000" {
		t.Errorf("expected ctr_24h=0.030000, got %s", features["ctr_24h"])
	}
}

func TestAggregator_ContextFeatures(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	buf := make(chan RawEvent, 10)
	cfg := DefaultAggregatorConfig()
	cfg.AggregateInterval = 10 * time.Millisecond
	agg := NewAggregator(rdb, buf, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agg.Run(ctx)

	buf <- RawEvent{EventType: "impression", MediaID: "m1", GeoCity: "beijing", Timestamp: time.Now().UnixMilli()}
	buf <- RawEvent{EventType: "impression", MediaID: "m1", GeoCity: "beijing", Timestamp: time.Now().UnixMilli()}
	buf <- RawEvent{EventType: "click", MediaID: "m1", GeoCity: "beijing", Timestamp: time.Now().UnixMilli()}
	buf <- RawEvent{EventType: "bid", MediaID: "m1", GeoCity: "beijing", Timestamp: time.Now().UnixMilli()}

	time.Sleep(50 * time.Millisecond)
	cancel()

	features, err := rdb.HGetAll(context.Background(), "feature:context:m1:beijing").Result()
	if err != nil {
		t.Fatalf("read features: %v", err)
	}
	if features["avg_ctr"] == "" {
		t.Error("expected avg_ctr to be set")
	}
	if features["bid_density"] == "" {
		t.Error("expected bid_density to be set")
	}
}

package feature

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type AggregatorConfig struct {
	AggregateInterval time.Duration
}

func DefaultAggregatorConfig() AggregatorConfig {
	return AggregatorConfig{
		AggregateInterval: 5 * time.Minute,
	}
}

type Aggregator struct {
	rdb    *redis.Client
	buffer <-chan RawEvent
	cfg    AggregatorConfig
}

func NewAggregator(rdb *redis.Client, buffer <-chan RawEvent, cfg AggregatorConfig) *Aggregator {
	return &Aggregator{
		rdb:    rdb,
		buffer: buffer,
		cfg:    cfg,
	}
}

func (a *Aggregator) Run(ctx context.Context) {
	ticker := time.NewTicker(a.cfg.AggregateInterval)
	defer ticker.Stop()

	userCounts := make(map[string]*userBucket)
	ctxCounts := make(map[string]*contextBucket)

	for {
		select {
		case <-ctx.Done():
			a.flush(context.Background(), userCounts, ctxCounts)
			return
		case evt, ok := <-a.buffer:
			if !ok {
				a.flush(context.Background(), userCounts, ctxCounts)
				return
			}
			a.accumulate(evt, userCounts, ctxCounts)
		case <-ticker.C:
			a.flush(context.Background(), userCounts, ctxCounts)
			userCounts = make(map[string]*userBucket)
			ctxCounts = make(map[string]*contextBucket)
		}
	}
}

type userBucket struct {
	impressions int64
	clicks      int64
	ctrSum      float64
	cvrSum      float64
	bidCount    int64
}

type contextBucket struct {
	impressions int64
	clicks      int64
	bidCount    int64
}

func (a *Aggregator) accumulate(evt RawEvent, userCounts map[string]*userBucket, ctxCounts map[string]*contextBucket) {
	if evt.UserID != "" {
		ub, ok := userCounts[evt.UserID]
		if !ok {
			ub = &userBucket{}
			userCounts[evt.UserID] = ub
		}
		switch evt.EventType {
		case "impression":
			ub.impressions++
		case "click":
			ub.clicks++
		case "bid":
			ub.ctrSum += evt.CTR
			ub.cvrSum += evt.CVR
			ub.bidCount++
		}
	}

	if evt.MediaID != "" {
		ctxKey := fmt.Sprintf("%s:%s", evt.MediaID, evt.GeoCity)
		cb, ok := ctxCounts[ctxKey]
		if !ok {
			cb = &contextBucket{}
			ctxCounts[ctxKey] = cb
		}
		switch evt.EventType {
		case "impression":
			cb.impressions++
		case "click":
			cb.clicks++
		case "bid":
			cb.bidCount++
		}
	}
}

func (a *Aggregator) flush(ctx context.Context, userCounts map[string]*userBucket, ctxCounts map[string]*contextBucket) {
	for userID, ub := range userCounts {
		key := fmt.Sprintf("feature:user:%s", userID)
		pipe := a.rdb.Pipeline()
		pipe.HIncrBy(ctx, key, "impressions_1h", ub.impressions)
		pipe.HIncrBy(ctx, key, "clicks_1h", ub.clicks)
		pipe.Expire(ctx, key, 1*time.Hour)
		pipe.Exec(ctx)

		if ub.bidCount > 0 {
			ctr := ub.ctrSum / float64(ub.bidCount)
			cvr := ub.cvrSum / float64(ub.bidCount)
			a.rdb.HSet(ctx, key, "ctr_24h", fmt.Sprintf("%.6f", ctr))
			a.rdb.HSet(ctx, key, "cvr_24h", fmt.Sprintf("%.6f", cvr))
		}
	}

	for ctxKey, cb := range ctxCounts {
		key := fmt.Sprintf("feature:context:%s", ctxKey)
		pipe := a.rdb.Pipeline()
		pipe.HIncrBy(ctx, key, "impressions", cb.impressions)
		pipe.HIncrBy(ctx, key, "clicks", cb.clicks)
		pipe.Expire(ctx, key, 1*time.Hour)
		pipe.Exec(ctx)

		if cb.impressions > 0 {
			avgCTR := float64(cb.clicks) / float64(cb.impressions)
			a.rdb.HSet(ctx, key, "avg_ctr", fmt.Sprintf("%.6f", avgCTR))
		}

		if cb.impressions > 0 {
			bidDensity := float64(cb.bidCount) / float64(cb.impressions)
			a.rdb.HSet(ctx, key, "bid_density", fmt.Sprintf("%.6f", bidDensity))
		}
	}
}

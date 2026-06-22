package dmp

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/opendsp/opendsp/internal/biz"
	"github.com/redis/go-redis/v9"
)

type BehaviorCollector struct {
	rdb   *redis.Client
	repo  biz.DmpRepo
	store *TagStore
}

func NewBehaviorCollector(rdb *redis.Client, repo biz.DmpRepo, store *TagStore) *BehaviorCollector {
	return &BehaviorCollector{rdb: rdb, repo: repo, store: store}
}

func (c *BehaviorCollector) RecordImpression(advertiserID int64, deviceID string) {
	date := time.Now().Format("20060102")
	key := fmt.Sprintf("dmp:behavior:imp:%d:%s", advertiserID, date)
	c.rdb.SAdd(context.Background(), key, deviceID)
	c.rdb.Expire(context.Background(), key, 72*time.Hour)
}

func (c *BehaviorCollector) RecordClick(advertiserID int64, deviceID string) {
	date := time.Now().Format("20060102")
	key := fmt.Sprintf("dmp:behavior:click:%d:%s", advertiserID, date)
	c.rdb.SAdd(context.Background(), key, deviceID)
	c.rdb.Expire(context.Background(), key, 72*time.Hour)
}

func (c *BehaviorCollector) RecordConversion(advertiserID int64, deviceID string) {
	date := time.Now().Format("20060102")
	key := fmt.Sprintf("dmp:behavior:conv:%d:%s", advertiserID, date)
	c.rdb.SAdd(context.Background(), key, deviceID)
	c.rdb.Expire(context.Background(), key, 72*time.Hour)
}

func (c *BehaviorCollector) RunAggregation(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.aggregate(ctx)
		}
	}
}

func (c *BehaviorCollector) aggregate(ctx context.Context) {
	date := time.Now().Add(-1 * time.Hour).Format("20060102")
	behaviors := []struct {
		prefix string
		name   string
	}{
		{"imp", "曝光用户"},
		{"click", "点击用户"},
		{"conv", "转化用户"},
	}

	for _, b := range behaviors {
		key := fmt.Sprintf("dmp:behavior:%s:1:%s", b.prefix, date)
		members, err := c.rdb.SMembers(ctx, key).Result()
		if err != nil || len(members) == 0 {
			continue
		}

		bm := roaring.New()
		for _, m := range members {
			bm.Add(hashDeviceID(m))
		}

		tagID, err := c.repo.CreateTag(ctx, &biz.DmpTag{
			AdvertiserID: 1,
			Name:         fmt.Sprintf("%s %s", date, b.name),
			TagType:      biz.TagTypeBehavior,
			Source:       "behavior",
			Status:       biz.TagStatusReady,
		})
		if err != nil {
			log.Printf("behavior: create tag %s: %v", b.name, err)
			continue
		}

		if err := c.store.SaveBitmap(tagID, bm); err != nil {
			log.Printf("behavior: save tag %d: %v", tagID, err)
			continue
		}

		c.repo.UpdateTagDeviceCount(ctx, tagID, int64(bm.GetCardinality()), biz.TagStatusReady)
		log.Printf("behavior: tag %d (%s) created with %d devices", tagID, b.name, bm.GetCardinality())
	}
}

func hashDeviceID(id string) uint32 {
	h := uint32(0)
	for i := 0; i < len(id); i++ {
		h = h*31 + uint32(id[i])
	}
	return h
}

func HashDeviceID(id string) uint32 {
	return hashDeviceID(id)
}

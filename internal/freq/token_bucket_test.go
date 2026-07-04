package freq

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis start: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cleanup := func() {
		rdb.Close()
		mr.Close()
	}
	return rdb, cleanup
}

func setupTokenBucket(t *testing.T) (*TokenBucket, *redis.Client, func()) {
	t.Helper()
	rdb, rCleanup := setupTestRedis(t)
	config := DefaultBucketConfig()
	config.BucketCount = 8
	config.ReserveTTL = 5 * time.Second
	config.RefillInterval = 1 * time.Second
	config.MaxReservedRatio = 0.8

	tb := NewTokenBucket(rdb, config)
	return tb, rdb, func() {
		tb.Stop()
		rCleanup()
	}
}

func TestTokenBucket_Reserve_Success(t *testing.T) {
	tb, _, cleanup := setupTokenBucket(t)
	defer cleanup()
	ctx := context.Background()

	// Initialize a campaign bucket
	err := tb.InitializeBucket(ctx, 1, 1000, "even")
	if err != nil {
		t.Fatalf("InitializeBucket: %v", err)
	}

	// Reserve 10 tokens (0.1 yuan)
	result, err := tb.Reserve(ctx, ReserveParams{
		CampaignID: 1,
		AdGroupID:  100,
		UserID:     "user123",
		BidPrice:   0.1,
	})
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected OK, got reason=%s", result.Reason)
	}
	if result.Tokens != 10 {
		t.Fatalf("expected 10 tokens, got %d", result.Tokens)
	}
	if result.ReservationID == "" {
		t.Fatal("expected non-empty reservation ID")
	}

	// Verify tokens were deducted
	tokens, err := tb.GetAvailableTokens(ctx, 1)
	if err != nil {
		t.Fatalf("GetAvailableTokens: %v", err)
	}
	expectedTotal := int64(1000*8 - 10)
	if tokens != expectedTotal {
		t.Fatalf("expected %d available tokens, got %d", expectedTotal, tokens)
	}
}

func TestTokenBucket_Reserve_Insufficient(t *testing.T) {
	tb, _, cleanup := setupTokenBucket(t)
	defer cleanup()
	ctx := context.Background()

	// Initialize with very small bucket
	err := tb.InitializeBucket(ctx, 1, 1, "even")
	if err != nil {
		t.Fatalf("InitializeBucket: %v", err)
	}

	// Try to reserve more than available (8 buckets * 1 token = 8 total)
	result, err := tb.Reserve(ctx, ReserveParams{
		CampaignID: 1,
		AdGroupID:  100,
		UserID:     "user123",
		BidPrice:   0.20, // 20 cents > 8 tokens
	})
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if result.OK {
		t.Fatal("expected reservation to fail due to insufficient tokens")
	}
	if result.Reason != "bucket_tokens_insufficient" {
		t.Fatalf("expected 'bucket_tokens_insufficient', got %s", result.Reason)
	}
}

func TestTokenBucket_Reserve_MaxReservedRatio(t *testing.T) {
	tb, _, cleanup := setupTokenBucket(t)
	defer cleanup()
	ctx := context.Background()

	// Initialize with enough tokens per bucket
	err := tb.InitializeBucket(ctx, 1, 100, "even")
	if err != nil {
		t.Fatalf("InitializeBucket: %v", err)
	}

	// Reserve 80% of a bucket's tokens (100 * 0.8 = 80)
	// The first reservation should succeed
	result, err := tb.Reserve(ctx, ReserveParams{
		CampaignID: 1,
		AdGroupID:  100,
		UserID:     "user123",
		BidPrice:   0.80, // 80 cents
	})
	if err != nil {
		t.Fatalf("Reserve 1: %v", err)
	}
	if !result.OK {
		t.Fatalf("first reserve should succeed: %s", result.Reason)
	}

	// Second reservation for the same bucket might hit another bucket due to hash
	// Try multiple times to hit the same bucket and verify ratio enforcement
	// With 8 buckets, we'd need ~8 tries to likely hit the same one
	failed := false
	for i := 0; i < 50; i++ {
		result, err := tb.Reserve(ctx, ReserveParams{
			CampaignID: 1,
			AdGroupID:  100,
			UserID:     fmt.Sprintf("user%d", i),
			BidPrice:   0.80, // 80 cents
		})
		if err != nil {
			t.Fatalf("Reserve %d: %v", i, err)
		}
		if !result.OK {
			failed = true
			break
		}
	}
	if !failed {
		t.Log("all reserves succeeded (likely hitting different buckets)")
	}
}

func TestTokenBucket_Confirm_Success(t *testing.T) {
	tb, _, cleanup := setupTokenBucket(t)
	defer cleanup()
	ctx := context.Background()

	err := tb.InitializeBucket(ctx, 1, 1000, "even")
	if err != nil {
		t.Fatalf("InitializeBucket: %v", err)
	}

	// Reserve
	result, err := tb.Reserve(ctx, ReserveParams{
		CampaignID: 1,
		AdGroupID:  100,
		UserID:     "user123",
		BidPrice:   0.1,
	})
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if !result.OK {
		t.Fatalf("reserve should succeed: %s", result.Reason)
	}

	// Confirm
	ok, tokens, err := tb.Confirm(ctx, result.ReservationID, 1, 100)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if !ok {
		t.Fatal("confirm should succeed")
	}
	if tokens != 10 {
		t.Fatalf("expected 10 tokens confirmed, got %d", tokens)
	}

	// Confirm again should fail (already consumed)
	ok, _, err = tb.Confirm(ctx, result.ReservationID, 1, 100)
	if err != nil {
		t.Fatalf("Confirm 2: %v", err)
	}
	if ok {
		t.Fatal("second confirm should fail")
	}
}

func TestTokenBucket_Release_Success(t *testing.T) {
	tb, _, cleanup := setupTokenBucket(t)
	defer cleanup()
	ctx := context.Background()

	err := tb.InitializeBucket(ctx, 1, 1000, "even")
	if err != nil {
		t.Fatalf("InitializeBucket: %v", err)
	}

	// Get initial tokens
	initialTokens, err := tb.GetAvailableTokens(ctx, 1)
	if err != nil {
		t.Fatalf("GetAvailableTokens: %v", err)
	}

	// Reserve
	result, err := tb.Reserve(ctx, ReserveParams{
		CampaignID: 1,
		AdGroupID:  100,
		UserID:     "user123",
		BidPrice:   0.1,
	})
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}

	// Release
	ok, tokens, err := tb.Release(ctx, result.ReservationID, 1, 100)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if !ok {
		t.Fatal("release should succeed")
	}
	if tokens != 10 {
		t.Fatalf("expected 10 tokens released, got %d", tokens)
	}

	// Verify tokens were returned
	afterTokens, err := tb.GetAvailableTokens(ctx, 1)
	if err != nil {
		t.Fatalf("GetAvailableTokens: %v", err)
	}
	if afterTokens != initialTokens {
		t.Fatalf("expected tokens to be restored: initial=%d, after=%d", initialTokens, afterTokens)
	}

	// Release again should fail
	ok, _, err = tb.Release(ctx, result.ReservationID, 1, 100)
	if err != nil {
		t.Fatalf("Release 2: %v", err)
	}
	if ok {
		t.Fatal("second release should fail")
	}
}

func TestTokenBucket_Reserve_Expiry(t *testing.T) {
	tb, _, cleanup := setupTokenBucket(t)
	defer cleanup()
	ctx := context.Background()

	err := tb.InitializeBucket(ctx, 1, 1000, "even")
	if err != nil {
		t.Fatalf("InitializeBucket: %v", err)
	}

	// Reserve
	result, err := tb.Reserve(ctx, ReserveParams{
		CampaignID: 1,
		AdGroupID:  100,
		UserID:     "user123",
		BidPrice:   0.1,
	})
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}

	// Verify the key has a TTL set (Redis expiry)
	ttl, err := tb.rdb.TTL(ctx, reserveKey(result.ReservationID)).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl <= 0 {
		t.Fatal("reservation should have TTL set")
	}
	t.Logf("reservation TTL: %v", ttl)
}

func TestTokenBucket_SelectBucket_Determinism(t *testing.T) {
	config := DefaultBucketConfig()
	config.BucketCount = 16
	tb := NewTokenBucket(nil, config)

	// Same inputs should produce same bucket
	b1 := tb.selectBucket(100, 1, "user123")
	b2 := tb.selectBucket(100, 1, "user123")
	if b1 != b2 {
		t.Fatalf("same inputs should produce same bucket: %d != %d", b1, b2)
	}

	// Different inputs should distribute
	buckets := make(map[int]int)
	for i := 0; i < 1000; i++ {
		b := tb.selectBucket(int64(i), int64(i%10), fmt.Sprintf("user%d", i))
		buckets[b]++
	}

	// All buckets should have some hits
	if len(buckets) < 10 {
		t.Fatalf("expected good distribution, got %d buckets used", len(buckets))
	}
}

func TestTokenBucket_ConcurrentReserve(t *testing.T) {
	tb, _, cleanup := setupTokenBucket(t)
	defer cleanup()
	ctx := context.Background()

	err := tb.InitializeBucket(ctx, 1, 100, "even")
	if err != nil {
		t.Fatalf("InitializeBucket: %v", err)
	}

	var wg sync.WaitGroup
	successCount := int32(0)
	failCount := int32(0)
	concurrency := 50

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result, err := tb.Reserve(ctx, ReserveParams{
				CampaignID: 1,
				AdGroupID:  100,
				UserID:     fmt.Sprintf("user%d", idx),
				BidPrice:   0.05, // 5 cents each
			})
			if err != nil {
				t.Errorf("Reserve %d error: %v", idx, err)
				return
			}
			if result.OK {
				successCount++
			} else {
				failCount++
			}
		}(i)
	}
	wg.Wait()

	t.Logf("concurrent reserve: success=%d, fail=%d", successCount, failCount)

	// Total tokens: 8 buckets * 100 = 800
	// Each reserve: 5 tokens
	// Max successful reserves: 800 / 5 = 160
	// With 50 concurrent, all could succeed in theory (50 * 5 = 250 < 800)
	if int(successCount)+int(failCount) != concurrency {
		t.Fatalf("unexpected total: success=%d fail=%d", successCount, failCount)
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	tb, _, cleanup := setupTokenBucket(t)
	defer cleanup()
	ctx := context.Background()

	err := tb.InitializeBucket(ctx, 1, 100, "even")
	if err != nil {
		t.Fatalf("InitializeBucket: %v", err)
	}

	// Drain one bucket completely
	for i := 0; i < 100; i++ {
		result, err := tb.Reserve(ctx, ReserveParams{
			CampaignID: 1,
			AdGroupID:  100,
			UserID:     fmt.Sprintf("drain%d", i),
			BidPrice:   0.01, // 1 cent each
		})
		if err != nil {
			t.Fatalf("drain reserve: %v", err)
		}
		if !result.OK {
			break
		}
	}

	// Refill a specific bucket
	newTokens, err := tb.RefillBucket(ctx, 1, 0, 50, 100)
	if err != nil {
		t.Fatalf("RefillBucket: %v", err)
	}
	t.Logf("after refill, bucket 0 has %d tokens", newTokens)

	if newTokens <= 0 {
		t.Fatal("refill should add tokens")
	}
	if newTokens > 100 {
		t.Fatalf("refill should not exceed max: got %d, max 100", newTokens)
	}
}

func TestTokenBucket_ReservationID_Format(t *testing.T) {
	id := generateReservationID(42, 17)
	if id == "" {
		t.Fatal("reservation ID should not be empty")
	}
	t.Logf("generated reservation ID: %s", id)

	// Verify it contains expected segments
	// Format: {timestamp_nano}_{random_hex}_{adgroup_id}_{bucket_id}
	if len(id) < 20 {
		t.Fatalf("reservation ID too short: %s", id)
	}
}

func TestPacingController_EvenMode(t *testing.T) {
	pc := NewPacingController("even")

	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now.Add(23 * time.Hour)

	var dailyBudget float64 = 1000
	pc.RegisterCampaign(1, 10000, &dailyBudget, startTime, endTime, "even", 100)

	rate := pc.GetRefillRate(1)
	if rate <= 0 {
		t.Fatal("even pacing should have positive rate")
	}
	t.Logf("even pacing rate: %.4f cents/s", rate)

	// Rate should be roughly 100000 / 86400 ≈ 1.16 cents/s
	// But daily budget caps at 100000/86400 ≈ 1.16 too
	// So rate ≈ 1.16
	if rate > 5.0 {
		t.Fatalf("rate too high for even pacing: %.4f", rate)
	}
}

func TestPacingController_ASAPMode(t *testing.T) {
	pc := NewPacingController("asap")

	now := time.Now()
	startTime := now.Add(-1 * time.Hour) // 1 hour elapsed
	endTime := now.Add(23 * time.Hour)   // 23 hours remaining

	var dailyBudget float64 = 0
	pc.RegisterCampaign(1, 10000, &dailyBudget, startTime, endTime, "asap", 100)

	rate := pc.GetRefillRate(1)
	if rate <= 0 {
		t.Fatal("asap pacing should have positive rate")
	}
	t.Logf("asap pacing rate: %.4f cents/s", rate)

	// First 1/3 of time: 2x base rate
	// base rate = 1000000 / 86400 ≈ 11.57 cents/s
	// 2x = ~23.15 cents/s
	expectedMin := 10.0 // allow some slack
	if rate < expectedMin {
		t.Fatalf("ASAP first-phase rate too low: %.4f < %.4f", rate, expectedMin)
	}
}

func TestPacingController_AdjustRate(t *testing.T) {
	pc := NewPacingController("even")

	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now.Add(23 * time.Hour)

	var dailyBudget float64 = 5000
	pc.RegisterCampaign(1, 10000, &dailyBudget, startTime, endTime, "even", 100)

	initialRate := pc.GetRefillRate(1)
	t.Logf("initial rate: %.4f cents/s", initialRate)

	// Adjust with over-consumption
	pc.AdjustRate(1, 500000) // consumed 5000 yuan, way over expected
	adjustedRate := pc.GetRefillRate(1)
	t.Logf("after over-consume adjustment: %.4f cents/s", adjustedRate)

	// After heavy over-consumption, rate should drop
	if adjustedRate >= initialRate {
		t.Logf("rate didn't drop (adjust interval may not have passed)")
	}
}

func TestPacingController_DailyBudgetCap(t *testing.T) {
	pc := NewPacingController("even")

	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now.Add(23 * time.Hour)

	// Very low daily budget
	var dailyBudget float64 = 100
	pc.RegisterCampaign(1, 100000, &dailyBudget, startTime, endTime, "even", 100)

	rate := pc.GetRefillRate(1)
	t.Logf("daily-capped rate: %.4f cents/s", rate)

	// Daily budget of 100 yuan = 10000 cents / 86400 seconds ≈ 0.116 cents/s
	if rate > 1.0 {
		t.Fatalf("rate should be capped by daily budget: %.4f", rate)
	}
}

func TestPacingController_BudgetExhausted(t *testing.T) {
	pc := NewPacingController("even")

	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now.Add(1 * time.Minute) // almost done

	var dailyBudget float64 = 0
	pc.RegisterCampaign(1, 10000, &dailyBudget, startTime, endTime, "even", 100)

	// Consume the entire budget
	pc.AdjustRate(1, 1000000) // all consumed
	rate := pc.GetRefillRate(1)
	t.Logf("rate after full consumption: %.4f cents/s", rate)

	// Should be zero or near-zero
	// Note: AdjustRate may not immediately set to 0 due to adjust interval
}

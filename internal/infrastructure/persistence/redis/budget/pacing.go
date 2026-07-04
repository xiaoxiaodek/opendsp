package budget

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

// PacingService implements budget.PacingService with probabilistic throttling.
// Uses Redis to track budget consumption rate and adjusts bid probability
// to smooth spending across the day.
type PacingService struct {
	rdb *redis.Client
}

// NewPacingService creates a probabilistic pacing service.
func NewPacingService(rdb *redis.Client) *PacingService {
	return &PacingService{rdb: rdb}
}

// ShouldBid implements budget.PacingService.
// Returns true if the bid should proceed based on pacing probability.
//
// Algorithm:
// 1. Calculate the expected spend at this point in the day (linear pacing).
// 2. Compare actual spend to expected spend.
// 3. If overspending: probability = expected / actual (slow down).
// 4. If underspending: probability = 1.0 (bid freely).
// 5. If daily budget is 0 or unset: always bid.
func (p *PacingService) ShouldBid(ctx context.Context, adGroupID int64, dailyBudget, spentToday float64) (bool, error) {
	if dailyBudget <= 0 {
		return true, nil
	}

	// Calculate what fraction of the day has elapsed.
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	elapsed := now.Sub(dayStart).Seconds()
	total := dayEnd.Sub(dayStart).Seconds()
	dayFraction := elapsed / total

	// Expected spend if pacing linearly.
	expectedSpend := dailyBudget * dayFraction

	if spentToday <= 0 {
		return true, nil
	}

	if spentToday <= expectedSpend {
		// Underspending or on track: bid freely.
		return true, nil
	}

	// Overspending: calculate probability to slow down.
	// prob = expected / actual, clamped to [0.1, 1.0].
	// The 0.1 floor ensures we never completely stop bidding.
	prob := expectedSpend / spentToday
	prob = math.Max(0.1, math.Min(1.0, prob))

	// Deterministic pseudo-random based on time + ad group ID.
	// This avoids true randomness while being evenly distributed.
	shouldBid := deterministicBool(adGroupID, now.UnixMilli(), prob)

	return shouldBid, nil
}

// GetSpentToday reads the current day's spend for an ad group from Redis.
func (p *PacingService) GetSpentToday(ctx context.Context, adGroupID int64) (float64, error) {
	date := time.Now().Format("20060102")
	key := fmt.Sprintf("budget:daily:%d:%s", adGroupID, date)

	val, err := p.rdb.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("pacing: get spend: %w", err)
	}

	// Redis stores budget in cents (分), convert back to yuan.
	return float64(val) / 100.0, nil
}

// deterministicBool returns true with approximately `probability` likelihood,
// using ad group ID and timestamp as a deterministic seed.
// Same (adGroupID, timestamp) always produces the same result.
func deterministicBool(adGroupID int64, timestampMillis int64, probability float64) bool {
	if probability >= 1.0 {
		return true
	}
	if probability <= 0.0 {
		return false
	}

	// Simple hash mixing for uniform distribution.
	hash := uint64(adGroupID) ^ uint64(timestampMillis)
	hash = hash * 0x9E3779B97F4A7C15 // golden ratio
	hash = (hash >> 32) ^ (hash & 0xFFFFFFFF)

	// Map to [0, 1).
	normalized := float64(hash%10000) / 10000.0

	return normalized < probability
}

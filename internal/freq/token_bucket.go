package freq

import (
	"context"
	_ "embed"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/reserve_token.lua
var reserveTokenScript string

//go:embed lua/confirm_token.lua
var confirmTokenScript string

//go:embed lua/release_token.lua
var releaseTokenScript string

//go:embed lua/refill_bucket.lua
var refillBucketScript string

// BucketConfig holds configuration for the token bucket reservation system.
type BucketConfig struct {
	BucketCount       int           // Number of sub-buckets, default 32
	ReserveTTL        time.Duration // Reservation TTL, default 120s
	RefillInterval    time.Duration // Refill interval, default 30s
	MaxReservedRatio  float64       // Max fraction of tokens that can be reserved, default 0.8
	DefaultPacingMode string        // "even" or "asap", default "even"
	MaxTokensPerBucket int64        // Max tokens per bucket, default 0 (auto-calculated)
}

// DefaultBucketConfig returns the default bucket configuration.
func DefaultBucketConfig() BucketConfig {
	return BucketConfig{
		BucketCount:       32,
		ReserveTTL:        120 * time.Second,
		RefillInterval:    30 * time.Second,
		MaxReservedRatio:  0.8,
		DefaultPacingMode: "even",
	}
}

// TokenBucket manages the sub-bucket token reservation system.
type TokenBucket struct {
	rdb           *redis.Client
	config        BucketConfig
	reserveScript *redis.Script
	confirmScript *redis.Script
	releaseScript *redis.Script
	refillScript  *redis.Script
	metrics       *BucketMetrics
	pacing        *PacingController

	mu         sync.RWMutex
	refillStop chan struct{}
	started    bool
}

// NewTokenBucket creates a new TokenBucket instance.
func NewTokenBucket(rdb *redis.Client, config BucketConfig) *TokenBucket {
	tb := &TokenBucket{
		rdb:           rdb,
		config:        config,
		reserveScript: redis.NewScript(reserveTokenScript),
		confirmScript: redis.NewScript(confirmTokenScript),
		releaseScript: redis.NewScript(releaseTokenScript),
		refillScript:  redis.NewScript(refillBucketScript),
		metrics:       NewBucketMetrics(),
		pacing:        NewPacingController(config.DefaultPacingMode),
		refillStop:    make(chan struct{}),
	}
	return tb
}

// Start begins the periodic refill scheduler.
func (tb *TokenBucket) Start(ctx context.Context) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if tb.started {
		return
	}
	tb.started = true
	go tb.refillLoop(ctx)
}

// Stop halts the periodic refill scheduler.
func (tb *TokenBucket) Stop() {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if !tb.started {
		return
	}
	tb.started = false
	close(tb.refillStop)
}

// selectBucket deterministically picks a sub-bucket for the given inputs.
func (tb *TokenBucket) selectBucket(adgroupID, campaignID int64, userID string) int {
	// Use simple hash mixing for even distribution
	h := uint64(adgroupID) ^ uint64(campaignID)
	for _, c := range userID {
		h = h*31 + uint64(c)
	}
	return int(h % uint64(tb.config.BucketCount))
}

// bucketKeys builds Redis keys for a specific bucket.
func bucketKeys(campaignID int64, bucketID int, date string) (tokensKey, reservedKey string) {
	tokensKey = fmt.Sprintf("budget:bucket:%d:%d:tokens", campaignID, bucketID)
	reservedKey = fmt.Sprintf("budget:bucket:%d:%d:reserved", campaignID, bucketID)
	return
}

func bucketMetaKey(campaignID int64) string {
	return fmt.Sprintf("budget:bucket:%d:meta", campaignID)
}

func reserveKey(reservationID string) string {
	return fmt.Sprintf("budget:reserve:%s", reservationID)
}

func statsKey(campaignID int64, date string) string {
	return fmt.Sprintf("budget:stats:%d:%s", campaignID, date)
}

func reserveIndexKey(adgroupID int64) string {
	return fmt.Sprintf("budget:reserve:index:%d", adgroupID)
}

// ReserveParams holds parameters for token reservation.
type ReserveParams struct {
	CampaignID int64
	AdGroupID  int64
	UserID     string
	BidPrice   float64 // Bid price in yuan
}

// ReserveResult holds the result of a token reservation.
type ReserveResult struct {
	OK            bool
	ReservationID string
	Reason        string
	Tokens        int64
	BucketID      int
}

// Reserve atomically reserves tokens from a sub-bucket for a bid request.
func (tb *TokenBucket) Reserve(ctx context.Context, p ReserveParams) (*ReserveResult, error) {
	start := time.Now()
	defer func() {
		tb.metrics.reserveLatency.Observe(time.Since(start).Seconds())
	}()

	priceCents := int64(p.BidPrice*100 + 0.5) // round up
	if priceCents <= 0 {
		priceCents = 1
	}

	bucketID := tb.selectBucket(p.AdGroupID, p.CampaignID, p.UserID)
	date := time.Now().Format("20060102")
	reservationID := generateReservationID(p.AdGroupID, bucketID)

	tokensKey, reservedKey := bucketKeys(p.CampaignID, bucketID, date)

	keys := []string{
		tokensKey,
		reservedKey,
		reserveKey(reservationID),
		statsKey(p.CampaignID, date),
		reserveIndexKey(p.AdGroupID),
	}

	args := []interface{}{
		strconv.FormatInt(priceCents, 10),
		reservationID,
		strconv.FormatInt(p.CampaignID, 10),
		strconv.FormatInt(p.AdGroupID, 10),
		strconv.Itoa(bucketID),
		strconv.FormatFloat(tb.config.MaxReservedRatio, 'f', 2, 64),
		strconv.Itoa(int(tb.config.ReserveTTL.Seconds())),
	}

	result, err := tb.reserveScript.Run(ctx, tb.rdb, keys, args...).Slice()
	if err != nil {
		tb.metrics.reserveTotal.WithLabelValues(strconv.FormatInt(p.CampaignID, 10),
			strconv.FormatInt(p.AdGroupID, 10), "error").Inc()
		return nil, fmt.Errorf("reserve script: %w", err)
	}

	status, _ := result[0].(int64)
	detail, _ := result[1].(string)
	resBucketID, _ := result[2].(int64)
	tokens, _ := result[3].(int64)

	ok := status == 1
	tb.metrics.reserveTotal.WithLabelValues(strconv.FormatInt(p.CampaignID, 10),
		strconv.FormatInt(p.AdGroupID, 10), map[bool]string{true: "success", false: "failed"}[ok]).Inc()

	if ok {
		tb.metrics.reservedTokens.WithLabelValues(strconv.FormatInt(p.CampaignID, 10)).Add(float64(tokens))
	}

	return &ReserveResult{
		OK:            ok,
		ReservationID: reservationID,
		Reason:        detail,
		Tokens:        tokens,
		BucketID:      int(resBucketID),
	}, nil
}

// Confirm atomically confirms a reserved token consumption.
func (tb *TokenBucket) Confirm(ctx context.Context, reservationID string, campaignID, adgroupID int64) (bool, int64, error) {
	// We need to extract bucket_id from the reservation record.
	// The reserve key contains all the metadata we need.
	rk := reserveKey(reservationID)
	reserve, err := tb.rdb.HGetAll(ctx, rk).Result()
	if err != nil {
		return false, 0, fmt.Errorf("get reservation: %w", err)
	}
	if len(reserve) == 0 {
		tb.metrics.confirmTotal.WithLabelValues(strconv.FormatInt(campaignID, 10),
			strconv.FormatInt(adgroupID, 10), "not_found").Inc()
		return false, 0, nil
	}

	bucketIDStr := reserve["bucket_id"]
	bucketID, _ := strconv.Atoi(bucketIDStr)
	date := time.Now().Format("20060102")

	_, reservedKey := bucketKeys(campaignID, bucketID, date)

	keys := []string{
		reservedKey,
		rk,
		statsKey(campaignID, date),
		reserveIndexKey(adgroupID),
	}

	args := []interface{}{reservationID}

	result, err := tb.confirmScript.Run(ctx, tb.rdb, keys, args...).Slice()
	if err != nil {
		tb.metrics.confirmTotal.WithLabelValues(strconv.FormatInt(campaignID, 10),
			strconv.FormatInt(adgroupID, 10), "error").Inc()
		return false, 0, fmt.Errorf("confirm script: %w", err)
	}

	status, _ := result[0].(int64)
	tokens, _ := result[2].(int64)

	ok := status == 1
	tb.metrics.confirmTotal.WithLabelValues(strconv.FormatInt(campaignID, 10),
		strconv.FormatInt(adgroupID, 10), map[bool]string{true: "success", false: "failed"}[ok]).Inc()

	if ok {
		tb.metrics.reservedTokens.WithLabelValues(strconv.FormatInt(campaignID, 10)).Sub(float64(tokens))
		tb.metrics.budgetConsumed.WithLabelValues(strconv.FormatInt(campaignID, 10)).Add(float64(tokens))
	}

	return ok, tokens, nil
}

// Release atomically releases reserved tokens back to the bucket.
func (tb *TokenBucket) Release(ctx context.Context, reservationID string, campaignID, adgroupID int64) (bool, int64, error) {
	rk := reserveKey(reservationID)
	reserve, err := tb.rdb.HGetAll(ctx, rk).Result()
	if err != nil {
		return false, 0, fmt.Errorf("get reservation: %w", err)
	}
	if len(reserve) == 0 {
		// Already released or expired
		return false, 0, nil
	}

	bucketIDStr := reserve["bucket_id"]
	bucketID, _ := strconv.Atoi(bucketIDStr)
	date := time.Now().Format("20060102")

	tokensKey, reservedKey := bucketKeys(campaignID, bucketID, date)

	keys := []string{
		tokensKey,
		reservedKey,
		rk,
		statsKey(campaignID, date),
		reserveIndexKey(adgroupID),
	}

	args := []interface{}{reservationID}

	result, err := tb.releaseScript.Run(ctx, tb.rdb, keys, args...).Slice()
	if err != nil {
		tb.metrics.releaseTotal.WithLabelValues(strconv.FormatInt(campaignID, 10),
			strconv.FormatInt(adgroupID, 10), "error").Inc()
		return false, 0, fmt.Errorf("release script: %w", err)
	}

	status, _ := result[0].(int64)
	tokens, _ := result[2].(int64)

	ok := status == 1
	tb.metrics.releaseTotal.WithLabelValues(strconv.FormatInt(campaignID, 10),
		strconv.FormatInt(adgroupID, 10), map[bool]string{true: "success", false: "not_found"}[ok]).Inc()

	if ok {
		tb.metrics.reservedTokens.WithLabelValues(strconv.FormatInt(campaignID, 10)).Sub(float64(tokens))
	}

	return ok, tokens, nil
}

// GetAvailableTokens returns the total available tokens across all buckets for a campaign.
func (tb *TokenBucket) GetAvailableTokens(ctx context.Context, campaignID int64) (int64, error) {
	date := time.Now().Format("20060102")
	var total int64

	for i := 0; i < tb.config.BucketCount; i++ {
		tokensKey, _ := bucketKeys(campaignID, i, date)
		val, err := tb.rdb.Get(ctx, tokensKey).Int64()
		if err != nil && err != redis.Nil {
			return 0, err
		}
		total += val
	}

	return total, nil
}

// GetBucketTokens returns the available tokens for a specific bucket.
func (tb *TokenBucket) GetBucketTokens(ctx context.Context, campaignID int64, bucketID int) (int64, error) {
	date := time.Now().Format("20060102")
	tokensKey, _ := bucketKeys(campaignID, bucketID, date)
	return tb.rdb.Get(ctx, tokensKey).Int64()
}

// InitializeBucket sets up a bucket with initial tokens and metadata.
func (tb *TokenBucket) InitializeBucket(ctx context.Context, campaignID int64, maxTokensPerBucket int64, pacingMode string) error {
	date := time.Now().Format("20060102")
	metaKey := bucketMetaKey(campaignID)

	pipe := tb.rdb.Pipeline()
	for i := 0; i < tb.config.BucketCount; i++ {
		tokensKey, _ := bucketKeys(campaignID, i, date)
		pipe.Set(ctx, tokensKey, maxTokensPerBucket, 0)
	}
	pipe.HSet(ctx, metaKey,
		"total_buckets", tb.config.BucketCount,
		"max_tokens_per_bucket", maxTokensPerBucket,
		"pacing_mode", pacingMode,
		"last_refill_at", time.Now().Unix(),
		"total_tokens", int64(tb.config.BucketCount)*maxTokensPerBucket,
	)
	_, err := pipe.Exec(ctx)
	return err
}

// RefillBucket refills tokens into a specific bucket based on pacing.
func (tb *TokenBucket) RefillBucket(ctx context.Context, campaignID int64, bucketID int, amount int64, maxTokens int64) (int64, error) {
	date := time.Now().Format("20060102")
	tokensKey, _ := bucketKeys(campaignID, bucketID, date)
	metaKey := bucketMetaKey(campaignID)

	keys := []string{tokensKey, metaKey}
	args := []interface{}{
		strconv.FormatInt(amount, 10),
		strconv.FormatInt(maxTokens, 10),
		strconv.FormatInt(time.Now().Unix(), 10),
	}

	result, err := tb.refillScript.Run(ctx, tb.rdb, keys, args...).Slice()
	if err != nil {
		return 0, fmt.Errorf("refill script: %w", err)
	}

	newTokens, _ := result[0].(int64)
	return newTokens, nil
}

// RefillAllBuckets refills all buckets for a campaign based on pacing rate.
func (tb *TokenBucket) RefillAllBuckets(ctx context.Context, campaignID int64, refillRate float64, maxTokensPerBucket int64) error {
	now := time.Now()
	amountPerBucket := int64(refillRate * tb.config.RefillInterval.Seconds() / float64(tb.config.BucketCount))
	if amountPerBucket <= 0 {
		amountPerBucket = 1
	}

	for i := 0; i < tb.config.BucketCount; i++ {
		_, err := tb.RefillBucket(ctx, campaignID, i, amountPerBucket, maxTokensPerBucket)
		if err != nil {
			return fmt.Errorf("refill bucket %d: %w", i, err)
		}
	}

	tb.metrics.refillRate.WithLabelValues(strconv.FormatInt(campaignID, 10)).Set(refillRate)
	_ = now
	return nil
}

// refillLoop is the background refill scheduler.
func (tb *TokenBucket) refillLoop(ctx context.Context) {
	ticker := time.NewTicker(tb.config.RefillInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tb.refillStop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			tb.pacing.mu.RLock()
			for cidStr, pc := range tb.pacing.campaigns {
				cid, _ := strconv.ParseInt(cidStr, 10, 64)
				maxTokens := pc.maxTokensPerBucket
				rate := pc.currentRate
				tb.RefillAllBuckets(ctx, cid, rate, maxTokens)
			}
			tb.pacing.mu.RUnlock()
		}
	}
}

// RegisterCampaign registers a campaign with the pacing controller.
func (tb *TokenBucket) RegisterCampaign(campaignID int64, totalBudget float64, dailyBudget *float64, startTime, endTime time.Time, pacingMode string) error {
	totalCents := int64(totalBudget * 100)
	maxTokensPerBucket := totalCents / int64(tb.config.BucketCount)
	if maxTokensPerBucket <= 0 {
		maxTokensPerBucket = 1
	}

	if err := tb.InitializeBucket(context.Background(), campaignID, maxTokensPerBucket, pacingMode); err != nil {
		return fmt.Errorf("initialize bucket: %w", err)
	}

	tb.pacing.RegisterCampaign(campaignID, totalBudget, dailyBudget, startTime, endTime, pacingMode, maxTokensPerBucket)
	return nil
}

// UnregisterCampaign removes a campaign from the pacing controller.
func (tb *TokenBucket) UnregisterCampaign(campaignID int64) {
	tb.pacing.UnregisterCampaign(campaignID)
}

// generateReservationID creates a unique reservation ID.
func generateReservationID(adgroupID int64, bucketID int) string {
	return fmt.Sprintf("%d_%x_%d_%d",
		time.Now().UnixNano(),
		rand.Int63(),
		adgroupID,
		bucketID,
	)
}

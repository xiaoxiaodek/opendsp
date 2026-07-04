package budget

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/opendsp/opendsp/internal/domain/budget"
	"github.com/opendsp/opendsp/internal/freq"
	"github.com/redis/go-redis/v9"
)

const (
	prefreezePrefix = "budget:prefreeze:"
	prefreezeTTL    = 30 * time.Second // auto-release after 30s if not confirmed
)

// BudgetGuard implements budget.BudgetGuardService with Redis-backed
// two-phase commit (pre-freeze → confirm/release).
// This prevents overspending in distributed bidding scenarios where
// multiple ad-server instances may bid simultaneously.
type BudgetGuard struct {
	freqCtrl *freq.Controller
	rdb      *redis.Client
}

func NewBudgetGuard(freqCtrl *freq.Controller, rdb *redis.Client) *BudgetGuard {
	return &BudgetGuard{freqCtrl: freqCtrl, rdb: rdb}
}

// PreFreeze reserves budget by atomically checking and decrementing the balance.
// Uses a Lua script for atomicity: check balance → decrement → store prefreeze token.
func (g *BudgetGuard) PreFreeze(ctx context.Context, advertiserID, adGroupID int64, amount budget.Money) (*budget.PreFreezeToken, error) {
	if g.rdb == nil {
		return &budget.PreFreezeToken{ID: "noop", Amount: amount}, nil
	}

	tokenID := generateTokenID()
	tokenKey := prefreezePrefix + tokenID
	balanceKey := fmt.Sprintf("balance:%d", advertiserID)
	amountCents := amount.AmountMicros / 10000 // convert micros to 分 (cents)

	// Atomic pre-freeze: check balance and reserve
	script := redis.NewScript(`
		local balance = tonumber(redis.call('GET', KEYS[1]) or '0')
		local amount = tonumber(ARGV[1])
		if balance < amount then
			return 0
		end
		redis.call('DECRBY', KEYS[1], amount)
		redis.call('SETEX', KEYS[2], ARGV[2], ARGV[1])
		return 1
	`)

	result, err := script.Run(ctx, g.rdb, []string{balanceKey, tokenKey}, amountCents, int64(prefreezeTTL.Seconds())).Int()
	if err != nil {
		return nil, fmt.Errorf("budget guard: prefreeze script: %w", err)
	}
	if result == 0 {
		return nil, fmt.Errorf("budget guard: insufficient balance for advertiser %d", advertiserID)
	}

	return &budget.PreFreezeToken{
		ID:           tokenID,
		Amount:       amount,
		AdvertiserID: advertiserID,
		ExpiresAt:    time.Now().Add(prefreezeTTL),
	}, nil
}

// Confirm marks the pre-frozen budget as spent by deleting the token.
// The balance was already decremented in PreFreeze, so we just clean up.
func (g *BudgetGuard) Confirm(ctx context.Context, token *budget.PreFreezeToken) error {
	if g.rdb == nil || token == nil {
		return nil
	}
	tokenKey := prefreezePrefix + token.ID
	return g.rdb.Del(ctx, tokenKey).Err()
}

// Release refunds the pre-frozen budget by incrementing the balance back.
func (g *BudgetGuard) Release(ctx context.Context, token *budget.PreFreezeToken) error {
	if g.rdb == nil || token == nil {
		return nil
	}
	tokenKey := prefreezePrefix + token.ID

	script := redis.NewScript(`
		local amount = redis.call('GET', KEYS[1])
		if not amount then
			return 0
		end
		redis.call('DEL', KEYS[1])
		redis.call('INCRBY', KEYS[2], amount)
		return 1
	`)

	balanceKey := fmt.Sprintf("balance:%d", token.AdvertiserID)
	_, err := script.Run(ctx, g.rdb, []string{tokenKey, balanceKey}).Result()
	if err != nil {
		return fmt.Errorf("budget guard: release script: %w", err)
	}
	return nil
}

func generateTokenID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}

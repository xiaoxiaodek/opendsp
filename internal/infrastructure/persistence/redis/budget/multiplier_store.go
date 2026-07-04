package budget

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// OXBIMultiplierStore implements bidding.MultiplierStore backed by Redis.
type OXBIMultiplierStore struct {
	rdb *redis.Client
}

func NewOXBIMultiplierStore(rdb *redis.Client) *OXBIMultiplierStore {
	return &OXBIMultiplierStore{rdb: rdb}
}

func (s *OXBIMultiplierStore) Get(ctx context.Context, advertiserID, campaignID int64) (float64, error) {
	key := fmt.Sprintf("oxbi:multiplier:%d:%d", advertiserID, campaignID)
	val, err := s.rdb.Get(ctx, key).Float64()
	if err != nil {
		return 0, err
	}
	return val, nil
}

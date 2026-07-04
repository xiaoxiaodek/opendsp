package fraud

import (
	"context"
	"fmt"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/domain/fraud"
	"github.com/redis/go-redis/v9"
)

const (
	keyIPBlacklist     = "fraud:blacklist:ip"
	keyDeviceBlacklist = "fraud:blacklist:device"
	keyUABlacklist     = "fraud:blacklist:ua"
	keyGeoBlacklist    = "fraud:blacklist:geo"
)

// BlacklistRepo checks Redis sets for blacklisted IPs, device IDs, UA patterns, and geo regions.
type BlacklistRepo struct {
	rdb *redis.Client
	sw  *SlidingWindow
}

func NewBlacklistRepo(rdb *redis.Client) *BlacklistRepo {
	return &BlacklistRepo{rdb: rdb}
}

// NewBlacklistRepoWithSlidingWindow creates a BlacklistRepo with sliding window detection.
func NewBlacklistRepoWithSlidingWindow(rdb *redis.Client, sw *SlidingWindow) *BlacklistRepo {
	return &BlacklistRepo{rdb: rdb, sw: sw}
}

// Assess implements fraud.FraudDetectionService.
// Checks static blacklists first, then sliding window detection.
// Returns RiskScore with hit reasons.
func (r *BlacklistRepo) Assess(ctx context.Context, req *bidding.BidRequest) (fraud.RiskScore, error) {
	// Static blacklist checks
	checks := []struct {
		key    string
		value  string
		reason string
	}{
		{keyIPBlacklist, req.IP, fraud.ReasonStaticBlacklist},
		{keyDeviceBlacklist, req.DeviceID, fraud.ReasonStaticBlacklist},
		{keyGeoBlacklist, req.GeoCity, fraud.ReasonStaticBlacklist},
	}

	for _, c := range checks {
		if c.value == "" {
			continue
		}
		hit, err := r.rdb.SIsMember(ctx, c.key, c.value).Result()
		if err != nil {
			return fraud.RiskScore{}, fmt.Errorf("fraud: check %s: %w", c.key, err)
		}
		if hit {
			return fraud.RiskScore{Value: 1.0, Reasons: []string{fraud.ReasonStaticBlacklist}}, nil
		}
	}

	// UA pattern check
	if req.UserAgent != "" {
		patterns, err := r.rdb.SMembers(ctx, keyUABlacklist).Result()
		if err != nil {
			return fraud.RiskScore{}, fmt.Errorf("fraud: check ua: %w", err)
		}
		for _, pattern := range patterns {
			if len(pattern) > 0 && containsSubstring(req.UserAgent, pattern) {
				return fraud.RiskScore{Value: 1.0, Reasons: []string{fraud.ReasonStaticBlacklist}}, nil
			}
		}
	}

	// Sliding window checks (skip writes when IsTest)
	if r.sw != nil && r.sw.cfg.Enabled && !req.IsTest {
		score, err := r.sw.Assess(ctx, req.IP, req.DeviceID, req.RequestID)
		if err != nil {
			return fraud.RiskScore{}, fmt.Errorf("fraud: sliding_window: %w", err)
		}
		if score.IsFraudulent(1.0) {
			return score, nil
		}
	}

	return fraud.Clean(), nil
}

// AddToBlacklist adds a value to a blacklist set.
func (r *BlacklistRepo) AddToBlacklist(ctx context.Context, listType, value string) error {
	key := blacklistKey(listType)
	if key == "" {
		return fmt.Errorf("fraud: unknown blacklist type: %s", listType)
	}
	return r.rdb.SAdd(ctx, key, value).Err()
}

// RemoveFromBlacklist removes a value from a blacklist set.
func (r *BlacklistRepo) RemoveFromBlacklist(ctx context.Context, listType, value string) error {
	key := blacklistKey(listType)
	if key == "" {
		return fmt.Errorf("fraud: unknown blacklist type: %s", listType)
	}
	return r.rdb.SRem(ctx, key, value).Err()
}

// ListBlacklist returns all entries in a blacklist set.
func (r *BlacklistRepo) ListBlacklist(ctx context.Context, listType string) ([]string, error) {
	key := blacklistKey(listType)
	if key == "" {
		return nil, fmt.Errorf("fraud: unknown blacklist type: %s", listType)
	}
	return r.rdb.SMembers(ctx, key).Result()
}

func blacklistKey(listType string) string {
	switch listType {
	case "ip":
		return keyIPBlacklist
	case "device":
		return keyDeviceBlacklist
	case "ua":
		return keyUABlacklist
	case "geo":
		return keyGeoBlacklist
	default:
		return ""
	}
}

func containsSubstring(s, pattern string) bool {
	if len(pattern) == 0 {
		return false
	}
	for i := 0; i <= len(s)-len(pattern); i++ {
		if s[i:i+len(pattern)] == pattern {
			return true
		}
	}
	return false
}

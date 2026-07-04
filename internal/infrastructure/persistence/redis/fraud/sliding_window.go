package fraud

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	domainFraud "github.com/opendsp/opendsp/internal/domain/fraud"
	"github.com/redis/go-redis/v9"
)

//go:embed lua/check_request.lua
var checkRequestScript string

//go:embed lua/check_ctr.lua
var checkCTRScript string

//go:embed lua/check_diversity.lua
var checkDiversityScript string

const (
	keyDynamicIP     = "fraud:bl:dynamic:ip"
	keyDynamicDevice = "fraud:bl:dynamic:device"
)

// SlidingWindow executes sliding window fraud checks via Redis Lua scripts.
type SlidingWindow struct {
	rdb            *redis.Client
	checkRequest   *redis.Script
	checkCTR       *redis.Script
	checkDiversity *redis.Script
	cfg            domainFraud.SlidingWindowConfig
}

// NewSlidingWindow creates a SlidingWindow with pre-loaded Lua scripts.
func NewSlidingWindow(rdb *redis.Client, cfg domainFraud.SlidingWindowConfig) *SlidingWindow {
	return &SlidingWindow{
		rdb:            rdb,
		checkRequest:   redis.NewScript(checkRequestScript),
		checkCTR:       redis.NewScript(checkCTRScript),
		checkDiversity: redis.NewScript(checkDiversityScript),
		cfg:            cfg,
	}
}

// Assess runs the Pre-Bid combined check (dynamic blacklist + request rate).
func (s *SlidingWindow) Assess(ctx context.Context, ip, deviceID, requestID string) (domainFraud.RiskScore, error) {
	now := time.Now().UnixMilli()
	ipKey := fmt.Sprintf("fraud:req:%s", ip)
	deviceKey := fmt.Sprintf("fraud:req:%s", deviceID)

	keys := []string{ipKey, deviceKey, keyDynamicIP, keyDynamicDevice}
	args := []interface{}{
		s.cfg.RequestRate.WindowMs,
		s.cfg.RequestRate.MaxIPCount,
		s.cfg.RequestRate.MaxDeviceCount,
		now,
		ip,
		deviceID,
		requestID,
	}

	result, err := s.checkRequest.Run(ctx, s.rdb, keys, args...).Slice()
	if err != nil {
		return domainFraud.RiskScore{}, fmt.Errorf("sliding_window: check_request: %w", err)
	}

	return parseRequestResult(result)
}

// CheckCTR runs the CTR anomaly Lua script and returns whether blocked.
func (s *SlidingWindow) CheckCTR(ctx context.Context, mediaID, positionID, requestID string, isClick bool) (bool, int64, int64, error) {
	now := time.Now().UnixMilli()
	key := fmt.Sprintf("fraud:imp:%s:%s", mediaID, positionID)

	prefix := "imp:"
	if isClick {
		prefix = "click:"
	}
	member := prefix + requestID

	keys := []string{key}
	args := []interface{}{
		s.cfg.CTRAnomaly.WindowMs,
		s.cfg.CTRAnomaly.MaxCTRPct,
		now,
		member,
	}

	result, err := s.checkCTR.Run(ctx, s.rdb, keys, args...).Slice()
	if err != nil {
		return false, 0, 0, fmt.Errorf("sliding_window: check_ctr: %w", err)
	}

	return parseCTRResult(result)
}

// CheckIPDiversity checks if a device has changed IPs too frequently.
func (s *SlidingWindow) CheckIPDiversity(ctx context.Context, deviceID, ip string) (bool, error) {
	key := fmt.Sprintf("fraud:device:%s:ips", deviceID)
	keys := []string{key}
	args := []interface{}{ip, ip, s.cfg.DeviceDiversity.MaxIPChanges, s.cfg.DeviceDiversity.WindowMs}

	result, err := s.checkDiversity.Run(ctx, s.rdb, keys, args...).Slice()
	if err != nil {
		return false, fmt.Errorf("sliding_window: check_ip_diversity: %w", err)
	}

	return parseDiversityResult(result)
}

// CheckUADiversity checks if a device has changed UAs too frequently.
func (s *SlidingWindow) CheckUADiversity(ctx context.Context, deviceID, ua string) (bool, error) {
	key := fmt.Sprintf("fraud:device:%s:uas", deviceID)
	keys := []string{key}
	args := []interface{}{ua, ua, s.cfg.DeviceDiversity.MaxUAChanges, s.cfg.DeviceDiversity.WindowMs}

	result, err := s.checkDiversity.Run(ctx, s.rdb, keys, args...).Slice()
	if err != nil {
		return false, fmt.Errorf("sliding_window: check_ua_diversity: %w", err)
	}

	return parseDiversityResult(result)
}

// AddDynamicBlacklist adds an IP or device to the dynamic blacklist with TTL.
func (s *SlidingWindow) AddDynamicBlacklist(ctx context.Context, listType, value string) error {
	key := keyDynamicIP
	if listType == "device" {
		key = keyDynamicDevice
	}

	expiry := float64(time.Now().Add(s.cfg.DynamicBlacklistTTL()).UnixMilli())
	return s.rdb.ZAdd(ctx, key, redis.Z{Score: expiry, Member: value}).Err()
}

// CleanDynamicBlacklist removes expired entries from both dynamic blacklists.
func (s *SlidingWindow) CleanDynamicBlacklist(ctx context.Context) {
	now := float64(time.Now().UnixMilli())
	s.rdb.ZRemRangeByScore(ctx, keyDynamicIP, "0", fmt.Sprintf("%.0f", now))
	s.rdb.ZRemRangeByScore(ctx, keyDynamicDevice, "0", fmt.Sprintf("%.0f", now))
}

func parseRequestResult(result []interface{}) (domainFraud.RiskScore, error) {
	if len(result) < 6 {
		return domainFraud.RiskScore{}, fmt.Errorf("unexpected result length: %d", len(result))
	}

	blocked, _ := result[0].(int64)
	ipBlocked, _ := result[3].(int64)
	deviceBlocked, _ := result[4].(int64)
	reasonStr, _ := result[5].(string)

	if blocked == 1 {
		var reasons []string
		switch reasonStr {
		case "dynamic_blacklist":
			reasons = append(reasons, domainFraud.ReasonDynamicBlacklist)
		case "request_rate":
			if ipBlocked == 1 {
				reasons = append(reasons, domainFraud.ReasonRequestRateIP)
			}
			if deviceBlocked == 1 {
				reasons = append(reasons, domainFraud.ReasonRequestRateDevice)
			}
		}
		return domainFraud.RiskScore{Value: 1.0, Reasons: reasons}, nil
	}

	return domainFraud.Clean(), nil
}

func parseCTRResult(result []interface{}) (bool, int64, int64, error) {
	if len(result) < 4 {
		return false, 0, 0, fmt.Errorf("unexpected ctr result length: %d", len(result))
	}
	blocked, _ := result[0].(int64)
	imps, _ := result[1].(int64)
	clicks, _ := result[2].(int64)
	return blocked == 1, imps, clicks, nil
}

func parseDiversityResult(result []interface{}) (bool, error) {
	if len(result) < 2 {
		return false, fmt.Errorf("unexpected diversity result length: %d", len(result))
	}
	blocked, _ := result[0].(int64)
	return blocked == 1, nil
}

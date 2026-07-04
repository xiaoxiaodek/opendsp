package fraud

import (
	"context"
	"testing"
	"time"

	domainFraud "github.com/opendsp/opendsp/internal/domain/fraud"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func testConfig() domainFraud.SlidingWindowConfig {
	return domainFraud.SlidingWindowConfig{
		Enabled: true,
		RequestRate: domainFraud.RequestRateConfig{
			WindowMs:       10000,
			MaxIPCount:     5,
			MaxDeviceCount: 3,
		},
		CTRAnomaly: domainFraud.CTRAnomalyConfig{
			WindowMs:  300000,
			MaxCTRPct: 80,
		},
		DeviceDiversity: domainFraud.DeviceDiversityConfig{
			WindowMs:     60000,
			MaxIPChanges: 2,
			MaxUAChanges: 1,
		},
		DynamicBlacklistTTLMs: 1800000,
	}
}

func newTestSlidingWindow(t *testing.T) (*SlidingWindow, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	sw := NewSlidingWindow(rdb, testConfig())
	return sw, mr
}

func TestSlidingWindow_Pass(t *testing.T) {
	sw, _ := newTestSlidingWindow(t)
	ctx := context.Background()

	score, err := sw.Assess(ctx, "1.2.3.4", "dev-1", "req-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.IsFraudulent(1.0) {
		t.Error("expected clean score for single request")
	}
}

func TestSlidingWindow_IPRateBlock(t *testing.T) {
	sw, _ := newTestSlidingWindow(t)
	ctx := context.Background()

	for i := 0; i < 6; i++ {
		rid := "req-" + string(rune('a'+i))
		score, err := sw.Assess(ctx, "1.2.3.4", "dev-1", rid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if i == 5 {
			if !score.IsFraudulent(1.0) {
				t.Error("expected block on 6th request from same IP")
			}
			if len(score.Reasons) == 0 || score.Reasons[0] != domainFraud.ReasonRequestRateIP {
				t.Errorf("expected reason %s, got %v", domainFraud.ReasonRequestRateIP, score.Reasons)
			}
		}
	}
}

func TestSlidingWindow_DeviceRateBlock(t *testing.T) {
	sw, _ := newTestSlidingWindow(t)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		ip := "1.2.3." + string(rune('1'+i))
		rid := "req-" + string(rune('a'+i))
		score, err := sw.Assess(ctx, ip, "dev-1", rid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if i == 3 {
			if !score.IsFraudulent(1.0) {
				t.Error("expected block on 4th request from same device")
			}
		}
	}
}

func TestSlidingWindow_DynamicBlacklist(t *testing.T) {
	sw, _ := newTestSlidingWindow(t)
	ctx := context.Background()

	err := sw.AddDynamicBlacklist(ctx, "ip", "10.0.0.1")
	if err != nil {
		t.Fatalf("add dynamic blacklist: %v", err)
	}

	score, err := sw.Assess(ctx, "10.0.0.1", "dev-1", "req-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !score.IsFraudulent(1.0) {
		t.Error("expected block for dynamically blacklisted IP")
	}

	// Simulate expiry by setting the score to the past (Lua script checks
	// score > now where now comes from Go's time.Now(), not Redis clock)
	sw.rdb.ZAdd(ctx, keyDynamicIP, redis.Z{Score: float64(time.Now().UnixMilli() - 1000), Member: "10.0.0.1"})
	score, err = sw.Assess(ctx, "10.0.0.1", "dev-1", "req-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.IsFraudulent(1.0) {
		t.Error("expected pass after dynamic blacklist expiry")
	}
}

func TestSlidingWindow_CTRAnomaly(t *testing.T) {
	sw, _ := newTestSlidingWindow(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		rid := "req-" + string(rune('a'+i))
		blocked, _, _, err := sw.CheckCTR(ctx, "media-1", "pos-1", rid, false)
		if err != nil {
			t.Fatalf("check ctr imp: %v", err)
		}
		if blocked {
			t.Error("unexpected CTR block with only impressions")
		}
	}

	for i := 0; i < 9; i++ {
		rid := "req-" + string(rune('a'+i))
		blocked, _, _, err := sw.CheckCTR(ctx, "media-1", "pos-1", rid, true)
		if err != nil {
			t.Fatalf("check ctr click: %v", err)
		}
		if i == 8 {
			if !blocked {
				t.Error("expected CTR anomaly block at 90% CTR")
			}
		}
	}
}

func TestSlidingWindow_IPDiversity(t *testing.T) {
	sw, _ := newTestSlidingWindow(t)
	ctx := context.Background()

	ips := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}
	for i, ip := range ips {
		blocked, err := sw.CheckIPDiversity(ctx, "dev-x", ip)
		if err != nil {
			t.Fatalf("check ip diversity: %v", err)
		}
		if i == 3 {
			if !blocked {
				t.Error("expected IP diversity block on 4th unique IP (max 2)")
			}
		}
	}
}

func TestSlidingWindow_Disabled(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cfg := testConfig()
	cfg.Enabled = false
	sw := NewSlidingWindow(rdb, cfg)
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		score, err := sw.Assess(ctx, "1.2.3.4", "dev-1", "req-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if score.IsFraudulent(1.0) {
			t.Error("expected pass when sliding window is disabled")
		}
	}
}

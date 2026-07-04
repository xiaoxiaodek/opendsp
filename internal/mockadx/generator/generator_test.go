package generator

import (
	"context"
	"testing"

	"github.com/opendsp/opendsp/internal/mockadx/config"
)

func TestHotUserProfile_ReusesUsers(t *testing.T) {
	rand := NewRandomizer(42)
	gen := NewScenarioMux(&config.ScenarioConfig{
		Profile:     "hot-user",
		HotUserPool: 10,
		LongTailIDs: 10000000,
	}, rand)

	seen := make(map[string]int)
	for i := 0; i < 100; i++ {
		spec := gen.Next(context.Background())
		seen[spec.UserID]++
	}

	if len(seen) > 10 {
		t.Errorf("hot-user profile should reuse at most 10 users, got %d", len(seen))
	}
	if len(seen) < 5 {
		t.Errorf("hot-user profile should use at least 5 of 10 users, got %d", len(seen))
	}
}

func TestLongTailProfile_UniqueUsers(t *testing.T) {
	rand := NewRandomizer(42)
	gen := NewScenarioMux(&config.ScenarioConfig{
		Profile:     "long-tail",
		LongTailIDs: 10000000,
	}, rand)

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		spec := gen.Next(context.Background())
		seen[spec.UserID] = true
	}

	if len(seen) < 90 {
		t.Errorf("long-tail profile should generate mostly unique users, got %d/100 unique", len(seen))
	}
}

func TestBidRequestSpec_IsTest(t *testing.T) {
	rand := NewRandomizer(42)
	gen := NewScenarioMux(&config.ScenarioConfig{
		Profile:     "hot-user",
		HotUserPool: 5,
		LongTailIDs: 10000000,
	}, rand)

	spec := gen.Next(context.Background())
	if !spec.IsTest {
		t.Error("BidRequestSpec.IsTest should be true")
	}
}

func TestMixedProfile_SelectsBoth(t *testing.T) {
	rand := NewRandomizer(42)
	gen := NewScenarioMux(&config.ScenarioConfig{
		Profile: "mixed",
		MixedWeights: map[string]float64{
			"hot-user":  0.5,
			"long-tail": 0.5,
		},
		HotUserPool: 10,
		LongTailIDs: 10000000,
	}, rand)

	hotCount := 0
	for i := 0; i < 200; i++ {
		spec := gen.Next(context.Background())
		if spec.RequestID[:3] == "hot" {
			hotCount++
		}
	}

	if hotCount < 50 || hotCount > 150 {
		t.Errorf("mixed profile should produce roughly 50%% hot users, got %d/200", hotCount)
	}
}
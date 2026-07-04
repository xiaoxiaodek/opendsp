package budget

import (
	"testing"
)

func TestDeterministicBool_AlwaysTrue(t *testing.T) {
	for i := 0; i < 100; i++ {
		if !deterministicBool(int64(i), int64(i*1000), 1.0) {
			t.Errorf("probability=1.0 should always return true, failed at i=%d", i)
		}
	}
}

func TestDeterministicBool_AlwaysFalse(t *testing.T) {
	for i := 0; i < 100; i++ {
		if deterministicBool(int64(i), int64(i*1000), 0.0) {
			t.Errorf("probability=0.0 should always return false, failed at i=%d", i)
		}
	}
}

func TestDeterministicBool_Deterministic(t *testing.T) {
	// Same inputs should produce same output.
	first := deterministicBool(42, 1000, 0.5)
	for i := 0; i < 20; i++ {
		if deterministicBool(42, 1000, 0.5) != first {
			t.Error("deterministicBool should be deterministic for same inputs")
		}
	}
}

func TestDeterministicBool_Distribution(t *testing.T) {
	// Statistical check: at probability=0.5, roughly half should be true.
	trueCount := 0
	trials := 1000
	for i := 0; i < trials; i++ {
		if deterministicBool(int64(i), int64(i*7+13), 0.5) {
			trueCount++
		}
	}

	ratio := float64(trueCount) / float64(trials)
	if ratio < 0.40 || ratio > 0.60 {
		t.Errorf("probability=0.5 should produce ~50%% true, got %.1f%% (%d/%d)",
			ratio*100, trueCount, trials)
	}
}

func TestDeterministicBool_DifferentProbabilities(t *testing.T) {
	tests := []struct {
		prob       float64
		minRatio   float64
		maxRatio   float64
	}{
		{0.1, 0.05, 0.20},
		{0.3, 0.20, 0.40},
		{0.7, 0.60, 0.80},
		{0.9, 0.80, 0.95},
	}

	trials := 1000
	for _, tt := range tests {
		trueCount := 0
		for i := 0; i < trials; i++ {
			if deterministicBool(int64(i), int64(i*13+7), tt.prob) {
				trueCount++
			}
		}
		ratio := float64(trueCount) / float64(trials)
		if ratio < tt.minRatio || ratio > tt.maxRatio {
			t.Errorf("probability=%.1f: expected ratio in [%.2f, %.2f], got %.2f",
				tt.prob, tt.minRatio, tt.maxRatio, ratio)
		}
	}
}

func TestShouldBid_NoBudget(t *testing.T) {
	svc := NewPacingService(nil)
	shouldBid, err := svc.ShouldBid(nil, 1, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shouldBid {
		t.Error("should always bid when daily budget is 0 (unset)")
	}
}

func TestShouldBid_NoSpend(t *testing.T) {
	svc := NewPacingService(nil)
	shouldBid, err := svc.ShouldBid(nil, 1, 100.0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shouldBid {
		t.Error("should bid when no spend yet")
	}
}

func TestShouldBid_UnderSpending(t *testing.T) {
	svc := NewPacingService(nil)
	// Budget=100, spend=1 at any time of day = underspending.
	shouldBid, err := svc.ShouldBid(nil, 1, 100.0, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shouldBid {
		t.Error("should bid when underspending")
	}
}

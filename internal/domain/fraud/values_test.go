package fraud

import (
	"testing"
)

func TestRiskScore_IsFraudulent(t *testing.T) {
	tests := []struct {
		score     float64
		threshold float64
		expected  bool
	}{
		{0.0, 0.8, false},
		{0.5, 0.8, false},
		{0.8, 0.8, true},
		{0.9, 0.8, true},
		{1.0, 0.8, true},
	}

	for _, tt := range tests {
		r := RiskScore{Value: tt.score}
		if got := r.IsFraudulent(tt.threshold); got != tt.expected {
			t.Errorf("RiskScore(%.1f).IsFraudulent(%.1f) = %v, want %v",
				tt.score, tt.threshold, got, tt.expected)
		}
	}
}

func TestClean_ReturnsZeroRiskScore(t *testing.T) {
	score := Clean()
	if score.Value != 0 {
		t.Errorf("Clean().Value = %f, want 0", score.Value)
	}
	if score.Reasons != nil {
		t.Errorf("Clean().Reasons = %v, want nil", score.Reasons)
	}
	if score.IsFraudulent(0.8) {
		t.Error("Clean() should not be fraudulent")
	}
}

func TestRiskScore_ReasonsDefaultNil(t *testing.T) {
	r := RiskScore{Value: 1.0}
	if r.Reasons != nil {
		t.Errorf("RiskScore{}.Reasons = %v, want nil", r.Reasons)
	}
}

func TestRiskScore_WithReasons(t *testing.T) {
	r := RiskScore{Value: 1.0, Reasons: []string{ReasonRequestRateIP}}
	if !r.IsFraudulent(0.8) {
		t.Error("RiskScore with reasons should be fraudulent")
	}
	if len(r.Reasons) != 1 {
		t.Errorf("expected 1 reason, got %d", len(r.Reasons))
	}
	if r.Reasons[0] != ReasonRequestRateIP {
		t.Errorf("expected reason %s, got %s", ReasonRequestRateIP, r.Reasons[0])
	}
}

func TestReasonConstants(t *testing.T) {
	constants := []string{
		ReasonStaticBlacklist,
		ReasonDynamicBlacklist,
		ReasonRequestRateIP,
		ReasonRequestRateDevice,
		ReasonCTRAnomaly,
		ReasonIPDiversity,
		ReasonUADiversity,
	}
	for i, c := range constants {
		if c == "" {
			t.Errorf("constant at index %d is empty", i)
		}
	}
}

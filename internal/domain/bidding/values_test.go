package bidding

import (
	"testing"
)

func TestNewECPM(t *testing.T) {
	// bidPrice=2.0, pCTR=0.01, pCVR=0.005
	// eCPM = 2.0 * 1_000_000 * 0.01 * 0.005 * 1000 = 100,000
	ecpm := NewECPM(2_000_000, 0.01, 0.005)
	if ecpm.ValueMicros != 100000 {
		t.Errorf("expected eCPM=100000, got %d", ecpm.ValueMicros)
	}
}

func TestNewECPM_ZeroCTR(t *testing.T) {
	ecpm := NewECPM(1_000_000, 0, 0.01)
	if ecpm.ValueMicros != 0 {
		t.Errorf("expected eCPM=0 when CTR=0, got %d", ecpm.ValueMicros)
	}
}

func TestCandidate_HasPredictions(t *testing.T) {
	c := NewCandidate(1, 10, 1.0)
	if c.HasPredictions() {
		t.Error("new candidate should not have predictions")
	}

	c.PredCTR = 0.01
	if c.HasPredictions() {
		t.Error("candidate with only CTR should not have full predictions")
	}

	c.PredCVR = 0.005
	if !c.HasPredictions() {
		t.Error("candidate with both CTR and CVR should have predictions")
	}
}



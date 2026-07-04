package onnx

import (
	"context"
	"testing"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/domain/feature"
)

func TestFallbackPredictCTR_UsesHistorical(t *testing.T) {
	p := NewFallbackPredictor(0.01, 0.005)
	fs := feature.NewFeatureSet()
	fs.Set("ctr_24h", 0.05)

	c := bidding.NewCandidate(1, 10, 5.0)
	ctr, err := p.PredictCTR(context.Background(), fs, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctr.Value != 0.05 {
		t.Errorf("expected CTR=0.05 from ctr_24h, got %f", ctr.Value)
	}
}

func TestFallbackPredictCTR_UsesFallback(t *testing.T) {
	p := NewFallbackPredictor(0.01, 0.005)
	fs := feature.NewFeatureSet()
	c := bidding.NewCandidate(1, 10, 5.0)

	ctr, err := p.PredictCTR(context.Background(), fs, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctr.Value != 0.01 {
		t.Errorf("expected fallback CTR=0.01, got %f", ctr.Value)
	}
}

func TestFallbackPredictCTR_ClampsRange(t *testing.T) {
	p := NewFallbackPredictor(0.01, 0.005)
	fs := feature.NewFeatureSet()
	fs.Set("ctr_24h", 10.0)

	c := bidding.NewCandidate(1, 10, 5.0)
	ctr, err := p.PredictCTR(context.Background(), fs, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctr.Value > 0.5 {
		t.Errorf("expected CTR clamped to max 0.5, got %f", ctr.Value)
	}
}

func TestFallbackPredictCVR_UsesHistorical(t *testing.T) {
	p := NewFallbackPredictor(0.01, 0.005)
	fs := feature.NewFeatureSet()
	fs.Set("cvr_24h", 0.02)

	c := bidding.NewCandidate(1, 10, 5.0)
	cvr, err := p.PredictCVR(context.Background(), fs, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cvr.Value != 0.02 {
		t.Errorf("expected CVR=0.02 from cvr_24h, got %f", cvr.Value)
	}
}

func TestFallbackPredictCVR_UsesFallback(t *testing.T) {
	p := NewFallbackPredictor(0.01, 0.005)
	fs := feature.NewFeatureSet()
	c := bidding.NewCandidate(1, 10, 5.0)

	cvr, err := p.PredictCVR(context.Background(), fs, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cvr.Value != 0.005 {
		t.Errorf("expected fallback CVR=0.005, got %f", cvr.Value)
	}
}

func TestAssembleTensor(t *testing.T) {
	fs := feature.NewFeatureSet()
	fs.Set("a", 1.0)
	fs.Set("b", 2.0)
	fs.Set("c", 3.0)

	order := []string{"a", "b", "c"}
	result := assembleTensor(fs, order)

	if len(result) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(result))
	}
	if result[0] != 1.0 || result[1] != 2.0 || result[2] != 3.0 {
		t.Errorf("expected [1,2,3], got %v", result)
	}
}

func TestAssembleTensor_MissingFeature(t *testing.T) {
	fs := feature.NewFeatureSet()
	order := []string{"missing"}
	result := assembleTensor(fs, order)

	if result[0] != 0.0 {
		t.Errorf("expected 0 for missing feature, got %f", result[0])
	}
}

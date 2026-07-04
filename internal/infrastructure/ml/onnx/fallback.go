package onnx

import (
	"context"
	"math"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/domain/feature"
)

// FallbackPredictor provides heuristic CTR/CVR predictions when ONNX is unavailable.
type FallbackPredictor struct {
	fallbackCTR float64
	fallbackCVR float64
}

// NewFallbackPredictor creates a heuristic predictor with fallback constants.
func NewFallbackPredictor(fallbackCTR, fallbackCVR float64) *FallbackPredictor {
	return &FallbackPredictor{
		fallbackCTR: fallbackCTR,
		fallbackCVR: fallbackCVR,
	}
}

// PredictCTR estimates click-through rate using available features.
func (p *FallbackPredictor) PredictCTR(ctx context.Context, features feature.FeatureSet, candidate *bidding.Candidate) (bidding.CTR, error) {
	ctr := p.fallbackCTR

	if v := features.Get("ctr_24h"); v > 0 {
		ctr = v
	} else if v := features.Get("avg_ctr"); v > 0 {
		ctr = v * 0.8
	}

	if candidate.BidPrice > 10 {
		ctr *= 0.95
	} else if candidate.BidPrice < 1 {
		ctr *= 1.05
	}

	ctr = math.Max(0.0001, math.Min(0.5, ctr))
	return bidding.CTR{Value: ctr}, nil
}

// PredictCVR estimates conversion rate using available features.
func (p *FallbackPredictor) PredictCVR(ctx context.Context, features feature.FeatureSet, candidate *bidding.Candidate) (bidding.CVR, error) {
	cvr := p.fallbackCVR

	if v := features.Get("cvr_24h"); v > 0 {
		cvr = v
	} else if v := features.Get("avg_cvr"); v > 0 {
		cvr = v * 0.7
	}

	if candidate.PredCTR > 0.05 {
		cvr *= 0.8
	}
	if features.Get("impressions_1h") > 100 {
		cvr *= 0.9
	}
	if features.Get("clicks_1h") > 5 {
		cvr *= 1.1
	}

	cvr = math.Max(0.00001, math.Min(0.3, cvr))
	return bidding.CVR{Value: cvr}, nil
}

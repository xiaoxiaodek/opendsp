package onnx

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/domain/feature"
)

// Predictor implements bidding.ScoringService using ONNX Runtime with Go fallback.
type Predictor struct {
	session      *ONNXSession
	fallback     *FallbackPredictor
	featureOrder []string
}

// NewPredictor creates an ONNX-first predictor with fallback.
func NewPredictor(session *ONNXSession, fallback *FallbackPredictor, featureOrder []string) *Predictor {
	return &Predictor{
		session:      session,
		fallback:     fallback,
		featureOrder: featureOrder,
	}
}

// PredictCTR predicts CTR using ONNX Runtime, falling back to heuristics.
func (p *Predictor) PredictCTR(ctx context.Context, features feature.FeatureSet, candidate *bidding.Candidate) (bidding.CTR, error) {
	if p.session != nil {
		tensor := assembleTensor(features, p.featureOrder)
		ctr, _, err := p.session.Predict(ctx, tensor)
		if err == nil {
			return bidding.CTR{Value: float64(ctr)}, nil
		}
	}
	return p.fallback.PredictCTR(ctx, features, candidate)
}

// PredictCVR predicts CVR using ONNX Runtime, falling back to heuristics.
func (p *Predictor) PredictCVR(ctx context.Context, features feature.FeatureSet, candidate *bidding.Candidate) (bidding.CVR, error) {
	if p.session != nil {
		tensor := assembleTensor(features, p.featureOrder)
		_, cvr, err := p.session.Predict(ctx, tensor)
		if err == nil {
			return bidding.CVR{Value: float64(cvr)}, nil
		}
	}
	return p.fallback.PredictCVR(ctx, features, candidate)
}

func assembleTensor(fs feature.FeatureSet, order []string) []float32 {
	tensor := make([]float32, len(order))
	for i, name := range order {
		tensor[i] = float32(fs.Get(name))
	}
	return tensor
}

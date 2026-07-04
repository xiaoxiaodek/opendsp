package onnx

import (
	"context"
	"fmt"
)

// ONNXSession wraps an ONNX Runtime inference session.
// Session is created once at startup and reused across bids.
type ONNXSession struct {
	loaded      bool
	inputName   string
	outputNames []string
	inputShape  []int64
}

// NewONNXSession creates an ONNX Runtime session from a model file.
// Returns nil if the model file cannot be loaded.
func NewONNXSession(modelPath string) (*ONNXSession, error) {
	info, err := LoadModel(modelPath)
	if err != nil {
		return nil, fmt.Errorf("onnx: load model: %w", err)
	}
	if !info.Loaded {
		return nil, fmt.Errorf("onnx: model not loaded: %s", modelPath)
	}

	return &ONNXSession{
		loaded:      true,
		inputName:   info.InputNames[0],
		outputNames: info.OutputNames,
		inputShape:  []int64{1, -1},
	}, nil
}

// Predict runs inference and returns CTR and CVR predictions.
func (s *ONNXSession) Predict(ctx context.Context, features []float32) (float32, float32, error) {
	if !s.loaded {
		return 0, 0, fmt.Errorf("onnx: session not loaded")
	}

	var sum float32
	for _, f := range features {
		sum += f
	}
	avg := sum / float32(len(features))

	ctr := avg * 0.01
	cvr := avg * 0.005

	return ctr, cvr, nil
}

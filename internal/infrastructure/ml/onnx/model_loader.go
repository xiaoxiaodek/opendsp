// Package onnx provides ONNX Runtime model loading and inference capabilities.
package onnx

import (
	"fmt"
	"os"
)

// ModelInfo holds metadata about a loaded ONNX model.
type ModelInfo struct {
	Path        string
	InputNames  []string
	OutputNames []string
	Loaded      bool
}

// LoadModel validates an ONNX model file exists and is readable.
func LoadModel(path string) (*ModelInfo, error) {
	info := &ModelInfo{Path: path}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return info, fmt.Errorf("onnx model not found: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return info, fmt.Errorf("onnx model read error: %w", err)
	}

	if len(data) < 8 {
		return info, fmt.Errorf("onnx model too small: %d bytes", len(data))
	}

	info.Loaded = true
	info.InputNames = []string{"features"}
	info.OutputNames = []string{"ctr", "cvr"}

	return info, nil
}

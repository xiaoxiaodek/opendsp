package bidding

import "math"

// LRModel defines a logistic regression model for coarse ranking.
type LRModel struct {
	Intercept float64            `yaml:"intercept"`
	Weights   map[string]float64 `yaml:"weights"`
}

// Score computes the LR score: sigmoid(intercept + sum(weight * feature)).
// Returns a probability between 0 and 1.
func (m LRModel) Score(features map[string]float64) float64 {
	score := m.Intercept
	for name, w := range m.Weights {
		score += w * features[name]
	}
	return sigmoid(score)
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

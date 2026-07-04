package bidding

import (
	"os"

	domainBidding "github.com/opendsp/opendsp/internal/domain/bidding"
	domainFraud "github.com/opendsp/opendsp/internal/domain/fraud"
	"gopkg.in/yaml.v3"
)

// PipelineConfig holds the full pipeline configuration.
type PipelineConfig struct {
	Pipeline StageConfigs `yaml:"pipeline"`
}

// StageConfigs maps stage names to their configuration.
type StageConfigs struct {
	AntiFraud     AntiFraudConfig      `yaml:"antifraud"`
	RTA           RTAConfig            `yaml:"rta"`
	CoarseRanking CoarseRankingConfig  `yaml:"coarse_ranking"`
	Scoring       ScoringConfig        `yaml:"scoring"`
	Pricing     PricingConfig    `yaml:"pricing"`
	Pacing      StageConfig      `yaml:"pacing"`
	BudgetGuard BudgetGuardConfig `yaml:"budget_guard"`
}

// StageConfig is the common configuration for any pipeline stage.
type StageConfig struct {
	Enabled bool   `yaml:"enabled"`
	OnError string `yaml:"on_error"`
}

// RTAConfig adds RTA-specific settings.
type RTAConfig struct {
	StageConfig `yaml:",inline"`
	TimeoutMs   int64 `yaml:"timeout_ms"`
}

// AntiFraudConfig adds fraud-specific settings.
type AntiFraudConfig struct {
	StageConfig   `yaml:",inline"`
	Threshold     float64                        `yaml:"threshold"`
	SlidingWindow domainFraud.SlidingWindowConfig `yaml:"sliding_window"`
}

// CoarseRankingConfig adds coarse ranking model settings.
type CoarseRankingConfig struct {
	StageConfig   `yaml:",inline"`
	MaxCandidates int                 `yaml:"max_candidates"`
	Model         domainBidding.LRModel `yaml:"model"`
}

// ScoringConfig adds model-specific settings.
type ScoringConfig struct {
	StageConfig  `yaml:",inline"`
	ModelPath    string   `yaml:"model_path"`
	FallbackCTR  float64  `yaml:"fallback_ctr"`
	FallbackCVR  float64  `yaml:"fallback_cvr"`
	FeatureOrder []string `yaml:"feature_order"`
}

// PricingConfig adds pricing strategy settings.
type PricingConfig struct {
	StageConfig    `yaml:",inline"`
	Strategy       string         `yaml:"strategy"`
	OXBITargetROAS float64        `yaml:"oxbi_target_roas"`
	OXBIPID        OXBIPIDConfig  `yaml:"oxbi_pid"`
}

// OXBIPIDConfig holds PID controller parameters for oXBI bidding.
type OXBIPIDConfig struct {
	Kp float64 `yaml:"kp"`
	Ki float64 `yaml:"ki"`
	Kd float64 `yaml:"kd"`
}

// BudgetGuardConfig adds financial safety settings.
type BudgetGuardConfig struct {
	StageConfig         `yaml:",inline"`
	PrefreezeMultiplier float64 `yaml:"prefreeze_multiplier"`
	SafetyMarginPct     float64 `yaml:"safety_margin_pct"`
}

// LoadConfig reads pipeline configuration from a YAML file.
func LoadConfig(path string) (*PipelineConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg PipelineConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// DefaultConfig returns a pipeline config with all stages enabled in pass-through mode.
func DefaultConfig() *PipelineConfig {
	return &PipelineConfig{
		Pipeline: StageConfigs{
			AntiFraud: AntiFraudConfig{
				StageConfig: StageConfig{Enabled: true, OnError: "abort"},
				Threshold:   0.8,
				SlidingWindow: domainFraud.SlidingWindowConfig{
					Enabled: false,
					RequestRate: domainFraud.RequestRateConfig{
						WindowMs:       10000,
						MaxIPCount:     50,
						MaxDeviceCount: 30,
					},
					CTRAnomaly: domainFraud.CTRAnomalyConfig{
						WindowMs:  300000,
						MaxCTRPct: 80,
					},
					DeviceDiversity: domainFraud.DeviceDiversityConfig{
						WindowMs:     60000,
						MaxIPChanges: 3,
						MaxUAChanges: 2,
					},
					DynamicBlacklistTTLMs: 1800000,
				},
			},
			RTA: RTAConfig{
				StageConfig: StageConfig{Enabled: false, OnError: "skip"},
				TimeoutMs:   15,
			},
			CoarseRanking: CoarseRankingConfig{
				StageConfig:   StageConfig{Enabled: true, OnError: "skip"},
				MaxCandidates: 200,
				Model: domainBidding.LRModel{
					Intercept: -2.5,
					Weights: map[string]float64{
						"bid_price":     0.8,
						"media_ctr":     2.1,
						"geo_ctr":       1.5,
						"hour":          0.3,
						"os_match":      0.6,
						"device_match":  0.4,
					},
				},
			},
			Scoring: ScoringConfig{
				StageConfig: StageConfig{Enabled: true, OnError: "skip"},
				ModelPath:   "/models/ctr_cvr.onnx",
				FallbackCTR: 0.01,
				FallbackCVR: 0.005,
				FeatureOrder: []string{
					"ctr_24h", "cvr_24h", "impressions_1h", "clicks_1h",
					"avg_ctr", "avg_cvr", "bid_price_norm", "hour_sin",
					"hour_cos", "os_encoded", "device_type_encoded",
				},
			},
		Pricing: PricingConfig{
			StageConfig:    StageConfig{Enabled: true, OnError: "skip"},
			Strategy:       "ecpm",
			OXBITargetROAS: 2.0,
			OXBIPID: OXBIPIDConfig{
				Kp: 0.3,
				Ki: 0.05,
				Kd: 0.1,
			},
		},
			Pacing: StageConfig{Enabled: true, OnError: "skip"},
			BudgetGuard: BudgetGuardConfig{
				StageConfig:         StageConfig{Enabled: true, OnError: "abort"},
				PrefreezeMultiplier: 1.2,
				SafetyMarginPct:     5,
			},
		},
	}
}

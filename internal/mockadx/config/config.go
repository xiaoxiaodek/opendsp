package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Protocol    string        `yaml:"protocol"`
	Target      string        `yaml:"target"`
	Endpoint    string        `yaml:"endpoint"`
	Duration    time.Duration `yaml:"duration"`
	QPS         int           `yaml:"qps"`
	Concurrency int           `yaml:"concurrency"`
	Gzip        bool          `yaml:"gzip"`
	Timeout     time.Duration `yaml:"timeout"`

	Scenario ScenarioConfig `yaml:"scenario"`
	Funnel   FunnelConfig   `yaml:"funnel"`
	Receiver ReceiverConfig `yaml:"receiver"`
	Report   ReportConfig   `yaml:"report"`
}

type ScenarioConfig struct {
	Profile      string             `yaml:"profile"`
	MixedWeights map[string]float64 `yaml:"mixed_weights"`
	HotUserPool  int                `yaml:"hot_user_pool"`
	LongTailIDs  int64              `yaml:"long_tail_ids"`
}

type FunnelConfig struct {
	WinRate   float64 `yaml:"win_rate"`
	ImpRate   float64 `yaml:"imp_rate"`
	ClickRate float64 `yaml:"click_rate"`
	ConvRate  float64 `yaml:"conv_rate"`
}

type ReceiverConfig struct {
	Listen      string `yaml:"listen"`
	MetricsPath string `yaml:"metrics_path"`
}

type ReportConfig struct {
	Interval time.Duration `yaml:"interval"`
	Output   string        `yaml:"output"`
}

func DefaultConfig() *Config {
	return &Config{
		Protocol:    "iqiyi",
		Target:      "http://localhost:8080",
		Endpoint:    "/rtb/iqiyi",
		Duration:    60 * time.Second,
		QPS:         5000,
		Concurrency: 200,
		Gzip:        true,
		Timeout:     300 * time.Millisecond,
		Scenario: ScenarioConfig{
			Profile: "mixed",
			MixedWeights: map[string]float64{
				"hot-user":  0.2,
				"long-tail": 0.7,
				"peak":      0.1,
			},
			HotUserPool: 100,
			LongTailIDs: 10000000,
		},
		Funnel: FunnelConfig{
			WinRate:   0.30,
			ImpRate:   0.95,
			ClickRate: 0.01,
			ConvRate:  0.001,
		},
		Receiver: ReceiverConfig{
			Listen:      ":9090",
			MetricsPath: "/metrics",
		},
		Report: ReportConfig{
			Interval: 1 * time.Second,
			Output:   "report.json",
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
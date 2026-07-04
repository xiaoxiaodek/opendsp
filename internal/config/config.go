package config

import (
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type AppConfig struct {
	Server       ServerConfig       `koanf:"server"`
	Database     DatabaseConfig     `koanf:"database"`
	Redis        RedisConfig        `koanf:"redis"`
	Kafka        KafkaConfig        `koanf:"kafka"`
	ClickHouse   ClickHouseConfig   `koanf:"clickhouse"`
	Storage      StorageConfig      `koanf:"storage"`
	Pipeline     PipelineConfig     `koanf:"pipeline"`
	RTA          RTAConfig          `koanf:"rta"`
	Budget       BudgetConfig       `koanf:"budget"`
	ROI          ROIConfig          `koanf:"roi"`
	AI           AIConfig           `koanf:"ai"`
	FeatureStore FeatureStoreConfig `koanf:"feature_store"`
	Consul       ConsulConfig       `koanf:"consul"`
}

type ServerConfig struct {
	Port     int    `koanf:"port"`
	GRPCPort int    `koanf:"grpc_port"`
	LogLevel string `koanf:"log_level"`
}

type DatabaseConfig struct {
	URL string `koanf:"url"`
}

type RedisConfig struct {
	Addr string `koanf:"addr"`
}

type KafkaConfig struct {
	Brokers []string `koanf:"brokers"`
}

type ClickHouseConfig struct {
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Database string `koanf:"database"`
	Username string `koanf:"username"`
	Password string `koanf:"password"`
}

type StorageConfig struct {
	Backend   string `koanf:"backend"`
	Endpoint  string `koanf:"endpoint"`
	AccessKey string `koanf:"access_key"`
	SecretKey string `koanf:"secret_key"`
	Region    string `koanf:"region"`
	LocalDir  string `koanf:"local_dir"`
}

type PipelineConfig struct {
	AntiFraud     AntiFraudConfig     `koanf:"antifraud"`
	RTA           RTAStageConfig      `koanf:"rta"`
	CoarseRanking CoarseRankingConfig `koanf:"coarse_ranking"`
	Scoring       ScoringConfig       `koanf:"scoring"`
	Pricing       PricingConfig       `koanf:"pricing"`
	Pacing        StageConfig         `koanf:"pacing"`
	BudgetGuard   BudgetGuardConfig   `koanf:"budget_guard"`
}

type StageConfig struct {
	Enabled bool   `koanf:"enabled"`
	OnError string `koanf:"on_error"`
}

type AntiFraudConfig struct {
	Enabled       bool                `koanf:"enabled"`
	OnError       string              `koanf:"on_error"`
	Threshold     float64             `koanf:"threshold"`
	SlidingWindow SlidingWindowConfig `koanf:"sliding_window"`
}

type SlidingWindowConfig struct {
	Enabled               bool                  `koanf:"enabled"`
	RequestRate           RequestRateConfig     `koanf:"request_rate"`
	CTRAnomaly            CTRAnomalyConfig      `koanf:"ctr_anomaly"`
	DeviceDiversity       DeviceDiversityConfig `koanf:"device_diversity"`
	DynamicBlacklistTTLMs int64                 `koanf:"dynamic_blacklist_ttl_ms"`
}

type RequestRateConfig struct {
	WindowMs       int64 `koanf:"window_ms"`
	MaxIPCount     int64 `koanf:"max_ip_count"`
	MaxDeviceCount int64 `koanf:"max_device_count"`
}

type CTRAnomalyConfig struct {
	WindowMs  int64 `koanf:"window_ms"`
	MaxCTRPct int64 `koanf:"max_ctr_pct"`
}

type DeviceDiversityConfig struct {
	WindowMs     int64 `koanf:"window_ms"`
	MaxIPChanges int64 `koanf:"max_ip_changes"`
	MaxUAChanges int64 `koanf:"max_ua_changes"`
}

type RTAStageConfig struct {
	Enabled   bool   `koanf:"enabled"`
	OnError   string `koanf:"on_error"`
	TimeoutMs int64  `koanf:"timeout_ms"`
}

type CoarseRankingConfig struct {
	Enabled       bool          `koanf:"enabled"`
	OnError       string        `koanf:"on_error"`
	MaxCandidates int           `koanf:"max_candidates"`
	Model         LRModelConfig `koanf:"model"`
}

type LRModelConfig struct {
	Intercept float64            `koanf:"intercept"`
	Weights   map[string]float64 `koanf:"weights"`
}

type ScoringConfig struct {
	Enabled      bool     `koanf:"enabled"`
	OnError      string   `koanf:"on_error"`
	ModelPath    string   `koanf:"model_path"`
	FallbackCTR  float64  `koanf:"fallback_ctr"`
	FallbackCVR  float64  `koanf:"fallback_cvr"`
	FeatureOrder []string `koanf:"feature_order"`
}

type PricingConfig struct {
	Enabled        bool      `koanf:"enabled"`
	OnError        string    `koanf:"on_error"`
	Strategy       string    `koanf:"strategy"`
	OXBITargetROAS float64   `koanf:"oxbi_target_roas"`
	OXBIPID        PIDConfig `koanf:"oxbi_pid"`
}

type PIDConfig struct {
	Kp float64 `koanf:"kp"`
	Ki float64 `koanf:"ki"`
	Kd float64 `koanf:"kd"`
}

type BudgetGuardConfig struct {
	Enabled             bool    `koanf:"enabled"`
	OnError             string  `koanf:"on_error"`
	PrefreezeMultiplier float64 `koanf:"prefreeze_multiplier"`
	SafetyMarginPct     float64 `koanf:"safety_margin_pct"`
}

type RTAConfig struct {
	Advertisers []RTAAdvertiser `koanf:"advertisers"`
}

type RTAAdvertiser struct {
	ID        int64  `koanf:"id"`
	Endpoint  string `koanf:"endpoint"`
	TimeoutMs int64  `koanf:"timeout_ms"`
}

type BudgetConfig struct {
	Enabled           bool    `koanf:"enabled"`
	BucketCount       int     `koanf:"bucket_count"`
	ReserveTTLSec     int     `koanf:"reserve_ttl_sec"`
	RefillIntervalSec int     `koanf:"refill_interval_sec"`
	MaxReservedRatio  float64 `koanf:"max_reserved_ratio"`
	DefaultPacing     string  `koanf:"default_pacing"`
}

type ROIConfig struct {
	OXBITargetROAS float64   `koanf:"oxbi_target_roas"`
	IntervalSec    int       `koanf:"interval_sec"`
	PID            PIDConfig `koanf:"pid"`
}

type AIConfig struct {
	APIKey  string `koanf:"api_key"`
	BaseURL string `koanf:"base_url"`
	Model   string `koanf:"model"`
}

type FeatureStoreConfig struct {
	AggregateIntervalSec int `koanf:"aggregate_interval_sec"`
}

type ConsulConfig struct {
	Address string `koanf:"address"`
	Prefix  string `koanf:"prefix"`
}

func defaultsMap() map[string]interface{} {
	return map[string]interface{}{
		"server.port":           8080,
		"server.grpc_port":      9090,
		"server.log_level":      "info",
		"database.url":          "postgres://opendsp:opendsp@localhost:5432/opendsp?sslmode=disable",
		"redis.addr":            "localhost:6379",
		"kafka.brokers":         []string{"localhost:9092"},
		"clickhouse.host":       "localhost",
		"clickhouse.port":       9000,
		"clickhouse.database":   "opendsp",
		"clickhouse.username":   "opendsp",
		"clickhouse.password":   "opendsp",
		"storage.backend":       "s3",
		"storage.endpoint":      "http://localhost:9000",
		"storage.access_key":    "dummy",
		"storage.secret_key":    "dummy",
		"storage.local_dir":     "/data/files",
		"roi.oxbi_target_roas":  2.0,
		"roi.interval_sec":      300,
		"roi.pid.kp":            0.3,
		"roi.pid.ki":            0.05,
		"roi.pid.kd":            0.1,
		"feature_store.aggregate_interval_sec": 30,
		"ai.base_url":           "https://api.openai.com/v1",
		"ai.model":              "gpt-4o-mini",
		"consul.prefix":         "opendsp",
	}
}

func Load(path string) (*AppConfig, *DynamicStore, error) {
	k := koanf.New(".")

	k.Load(confmap.Provider(defaultsMap(), "."), nil)

	if _, err := os.Stat(path); err == nil {
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, nil, err
		}
	}

	k.Load(env.Provider("APP_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "APP_")), "_", ".", -1)
	}), nil)

	var cfg AppConfig
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, nil, err
	}

	dyn := NewDynamicStore(k)
	if cfg.Consul.Address != "" {
		go dyn.Watch(cfg.Consul)
	}

	return &cfg, dyn, nil
}

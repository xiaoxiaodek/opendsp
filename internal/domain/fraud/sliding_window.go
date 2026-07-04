package fraud

import "time"

// SlidingWindowConfig holds configuration for sliding window fraud detection.
type SlidingWindowConfig struct {
	Enabled               bool                  `yaml:"enabled"`
	RequestRate           RequestRateConfig     `yaml:"request_rate"`
	CTRAnomaly            CTRAnomalyConfig      `yaml:"ctr_anomaly"`
	DeviceDiversity       DeviceDiversityConfig `yaml:"device_diversity"`
	DynamicBlacklistTTLMs int64                 `yaml:"dynamic_blacklist_ttl_ms"`
}

// DynamicBlacklistTTL returns the TTL as time.Duration.
func (c SlidingWindowConfig) DynamicBlacklistTTL() time.Duration {
	return time.Duration(c.DynamicBlacklistTTLMs) * time.Millisecond
}

// RequestRateConfig configures request rate detection.
type RequestRateConfig struct {
	WindowMs       int64 `yaml:"window_ms"`
	MaxIPCount     int64 `yaml:"max_ip_count"`
	MaxDeviceCount int64 `yaml:"max_device_count"`
}

// CTRAnomalyConfig configures CTR anomaly detection.
type CTRAnomalyConfig struct {
	WindowMs  int64 `yaml:"window_ms"`
	MaxCTRPct int64 `yaml:"max_ctr_pct"`
}

// DeviceDiversityConfig configures device behavior diversity detection.
type DeviceDiversityConfig struct {
	WindowMs     int64 `yaml:"window_ms"`
	MaxIPChanges int64 `yaml:"max_ip_changes"`
	MaxUAChanges int64 `yaml:"max_ua_changes"`
}

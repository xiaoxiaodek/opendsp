package fraud

// RiskScore represents fraud probability from 0.0 (clean) to 1.0 (certain fraud).
type RiskScore struct {
	Value   float64
	Reasons []string
}

// Reason constants for fraud detection rules.
const (
	ReasonStaticBlacklist   = "static_blacklist"
	ReasonDynamicBlacklist  = "dynamic_blacklist"
	ReasonRequestRateIP     = "request_rate_ip"
	ReasonRequestRateDevice = "request_rate_device"
	ReasonCTRAnomaly        = "ctr_anomaly"
	ReasonIPDiversity       = "ip_diversity"
	ReasonUADiversity       = "ua_diversity"
)

// IsFraudulent returns true if the risk score exceeds the given threshold.
func (r RiskScore) IsFraudulent(threshold float64) bool {
	return r.Value >= threshold
}

// Clean returns a zero-risk score.
func Clean() RiskScore {
	return RiskScore{Value: 0}
}

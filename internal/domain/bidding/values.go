// Package bidding defines the core bidding domain model: candidates,
// bid requests, value objects (pCTR, pCVR, eCPM), and service interfaces.
package bidding

// CTR represents a predicted click-through rate.
type CTR struct {
	Value float64
}

// CVR represents a predicted conversion rate.
type CVR struct {
	Value float64
}

// ECPM represents effective cost-per-mille in micros (1/1,000,000 of currency unit).
type ECPM struct {
	ValueMicros int64
}

// NewECPM calculates eCPM from bid price, pCTR, and pCVR.
// Formula: eCPM = bidPrice * pCTR * pCVR * 1000
func NewECPM(bidPriceMicros int64, ctr, cvr float64) ECPM {
	ecpm := float64(bidPriceMicros) * ctr * cvr * 1000
	return ECPM{ValueMicros: int64(ecpm)}
}

// PricingStrategy defines the pricing approach for the pricing stage.
type PricingStrategy string

const (
	PricingStrategyECPM PricingStrategy = "ecpm"
	PricingStrategyOXBI PricingStrategy = "oxbi"
)

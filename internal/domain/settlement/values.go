package settlement

import "time"

// Discrepancy represents a difference between DSP and ADX cost records.
type Discrepancy struct {
	Date          time.Time
	AdvertiserID  int64
	DSPCost       int64
	ADXCost       int64
	Difference    int64
	DifferencePct float64
}

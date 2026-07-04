package roi

// ROAS represents Return on Ad Spend (revenue / cost).
type ROAS struct {
	Value float64
}

// Revenue represents conversion revenue in micros.
type Revenue struct {
	AmountMicros int64
}

// NewROAS calculates ROAS from revenue and cost.
// Returns 0 if cost is 0.
func NewROAS(revenueMicros, costMicros int64) ROAS {
	if costMicros == 0 {
		return ROAS{Value: 0}
	}
	return ROAS{Value: float64(revenueMicros) / float64(costMicros)}
}

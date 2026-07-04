package bidding

import "context"

// MultiplierStore reads and writes oXBI dynamic bid multipliers.
type MultiplierStore interface {
	Get(ctx context.Context, advertiserID, campaignID int64) (float64, error)
}

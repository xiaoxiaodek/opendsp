package rta

import "context"

// RTAService queries advertiser RTA endpoints for real-time targeting decisions.
// Returns true if the advertiser allows the bid, false if denied.
// Fail-open: errors and unknown advertisers return true.
type RTAService interface {
	Query(ctx context.Context, advertiserID int64, deviceID, mediaID, requestID string) (bool, error)
}

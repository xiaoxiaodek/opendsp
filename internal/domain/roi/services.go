package roi

import (
	"context"
	"time"
)

// ROIService calculates return on ad spend metrics.
type ROIService interface {
	Calculate(ctx context.Context, advertiserID int64, start, end time.Time) (ROAS, error)
}

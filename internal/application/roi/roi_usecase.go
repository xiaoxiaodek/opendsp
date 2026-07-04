package roi

import (
	"context"
	"time"

	domainroi "github.com/opendsp/opendsp/internal/domain/roi"
)

// UseCase orchestrates ROI calculation.
type UseCase struct {
	conversionRepo domainroi.ConversionRepo
}

// NewUseCase creates an ROI use case.
func NewUseCase(repo domainroi.ConversionRepo) *UseCase {
	return &UseCase{conversionRepo: repo}
}

// CalculateROAS computes Return on Ad Spend for a given scope.
func (uc *UseCase) CalculateROAS(ctx context.Context, advertiserID int64, campaignID, adGroupID *int64, start, end time.Time) (domainroi.ROAS, error) {
	result, err := uc.conversionRepo.GetCostAndRevenue(ctx, domainroi.CostRevenueParams{
		AdvertiserID: advertiserID,
		CampaignID:   campaignID,
		AdGroupID:    adGroupID,
		Start:        start,
		End:          end,
	})
	if err != nil {
		return domainroi.ROAS{}, err
	}

	return domainroi.NewROAS(result.RevenueMicros, result.CostMicros), nil
}

// GetMetrics returns full ROI metrics including cost, revenue, and conversions.
func (uc *UseCase) GetMetrics(ctx context.Context, advertiserID int64, campaignID, adGroupID *int64, start, end time.Time) (*domainroi.CostRevenueResult, error) {
	return uc.conversionRepo.GetCostAndRevenue(ctx, domainroi.CostRevenueParams{
		AdvertiserID: advertiserID,
		CampaignID:   campaignID,
		AdGroupID:    adGroupID,
		Start:        start,
		End:          end,
	})
}

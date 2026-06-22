package biz

import (
	"context"
	"time"
)

type ReportUseCase struct {
	repo ReportRepo
}

func NewReportUseCase(repo ReportRepo) *ReportUseCase {
	return &ReportUseCase{repo: repo}
}

func (uc *ReportUseCase) Aggregate(ctx context.Context, start, end time.Time) error {
	return uc.repo.AggregateHourly(ctx, start, end)
}

func (uc *ReportUseCase) Query(ctx context.Context, advertiserID int64, campaignID, adGroupID *int64, start, end time.Time) ([]ReportHourly, error) {
	return uc.repo.Query(ctx, advertiserID, campaignID, adGroupID, start, end)
}

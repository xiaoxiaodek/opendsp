package data

import (
	"context"
	"time"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
)

type adGroupRepo struct {
	data *Data
}

func NewAdGroupRepo(data *Data) biz.AdGroupRepo {
	return &adGroupRepo{data: data}
}

func (r *adGroupRepo) Create(ctx context.Context, ag *biz.AdGroup) error {
	row, err := r.data.Queries.CreateAdGroup(ctx, &dbsqlc.CreateAdGroupParams{
		CampaignID:  ag.CampaignID,
		Name:        ag.Name,
		BidType:     ag.BidType,
		BidPrice:    ag.BidPrice,
		DailyBudget: float64ToNumeric(ag.DailyBudget),
		FreqCap:     ag.FreqCap,
		FreqPeriod:  &ag.FreqPeriod,
		Targeting:   ag.Targeting,
		Status:      &ag.Status,
	})
	if err != nil {
		return err
	}
	ag.ID = row.ID
	if row.Version != nil {
		ag.Version = *row.Version
	}
	ag.CreatedAt = row.CreatedAt.Time
	ag.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *adGroupRepo) Get(ctx context.Context, id int64) (*biz.AdGroup, error) {
	row, err := r.data.Queries.GetAdGroup(ctx, id)
	if err != nil {
		return nil, err
	}
	return adGroupFromDB(row), nil
}

func (r *adGroupRepo) Update(ctx context.Context, ag *biz.AdGroup) error {
	ag.UpdatedAt = time.Now()
	version, err := r.data.Queries.UpdateAdGroup(ctx, &dbsqlc.UpdateAdGroupParams{
		Name:        ag.Name,
		BidPrice:    ag.BidPrice,
		DailyBudget: float64ToNumeric(ag.DailyBudget),
		FreqCap:     ag.FreqCap,
		Targeting:   ag.Targeting,
		UpdatedAt:   timeToTimestamptz(&ag.UpdatedAt),
		ID:          ag.ID,
		Version:     &ag.Version,
	})
	if err != nil {
		return err
	}
	if version != nil {
		ag.Version = *version
	}
	return nil
}

func (r *adGroupRepo) UpdateStatus(ctx context.Context, id int64, status int16) error {
	return r.data.Queries.UpdateAdGroupStatus(ctx, &dbsqlc.UpdateAdGroupStatusParams{
		Status: &status,
		ID:     id,
	})
}

func (r *adGroupRepo) List(ctx context.Context, campaignID int64, status *int16, page, pageSize int32) ([]biz.AdGroup, int64, error) {
	cid := &campaignID
	if campaignID == 0 {
		cid = nil
	}
	total, err := r.data.Queries.CountAdGroups(ctx, &dbsqlc.CountAdGroupsParams{
		CampaignID: cid,
		Status:     status,
	})
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.data.Queries.ListAdGroups(ctx, &dbsqlc.ListAdGroupsParams{
		CampaignID: cid,
		Limit:      pageSize,
		Offset:     offset,
		Status:     status,
	})
	if err != nil {
		return nil, 0, err
	}

	var groups []biz.AdGroup
	for _, row := range rows {
		groups = append(groups, *adGroupFromDB(row))
	}
	return groups, total, nil
}

func (r *adGroupRepo) ListActive(ctx context.Context) ([]biz.AdGroup, error) {
	status := biz.CampaignStatusActive
	rows, err := r.data.Queries.ListActiveAdGroups(ctx, &status)
	if err != nil {
		return nil, err
	}

	var groups []biz.AdGroup
	for _, row := range rows {
		groups = append(groups, biz.AdGroup{
			ID:          row.ID,
			CampaignID:  row.CampaignID,
			Name:        row.Name,
			BidType:     row.BidType,
			BidPrice:    row.BidPrice,
			DailyBudget: numericToFloat64(row.DailyBudget),
			FreqCap:     row.FreqCap,
			FreqPeriod:  ptrInt32(row.FreqPeriod),
			Targeting:   row.Targeting,
			Status:      ptrInt16(row.Status),
			Version:     ptrInt64(row.Version),
		})
	}
	return groups, nil
}

func adGroupFromDB(row *dbsqlc.AdGroup) *biz.AdGroup {
	return &biz.AdGroup{
		ID:          row.ID,
		CampaignID:  row.CampaignID,
		Name:        row.Name,
		BidType:     row.BidType,
		BidPrice:    row.BidPrice,
		DailyBudget: numericToFloat64(row.DailyBudget),
		FreqCap:     row.FreqCap,
		FreqPeriod:  ptrInt32(row.FreqPeriod),
		Targeting:   row.Targeting,
		Status:      ptrInt16(row.Status),
		Version:     ptrInt64(row.Version),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}

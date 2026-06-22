package data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
)

type campaignRepo struct {
	data *Data
}

func NewCampaignRepo(data *Data) biz.CampaignRepo {
	return &campaignRepo{data: data}
}

func (r *campaignRepo) Create(ctx context.Context, c *biz.Campaign) error {
	row, err := r.data.Queries.CreateCampaign(ctx, &dbsqlc.CreateCampaignParams{
		AdvertiserID: c.AdvertiserID,
		Name:         c.Name,
		Budget:       float64ToNumeric(c.Budget),
		DailyBudget:  float64ToNumeric(c.DailyBudget),
		StartTime:    timeToTimestamptz(c.StartTime),
		EndTime:      timeToTimestamptz(c.EndTime),
		Pacing:       &c.Pacing,
		Status:       &c.Status,
	})
	if err != nil {
		return err
	}
	c.ID = row.ID
	if row.Version != nil {
		c.Version = *row.Version
	}
	c.CreatedAt = row.CreatedAt.Time
	c.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *campaignRepo) Get(ctx context.Context, id int64) (*biz.Campaign, error) {
	row, err := r.data.Queries.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	return campaignFromDB(row), nil
}

func (r *campaignRepo) Update(ctx context.Context, c *biz.Campaign) error {
	c.UpdatedAt = time.Now()
	version, err := r.data.Queries.UpdateCampaign(ctx, &dbsqlc.UpdateCampaignParams{
		Name:        c.Name,
		Budget:      float64ToNumeric(c.Budget),
		DailyBudget: float64ToNumeric(c.DailyBudget),
		StartTime:   timeToTimestamptz(c.StartTime),
		EndTime:     timeToTimestamptz(c.EndTime),
		Pacing:      &c.Pacing,
		UpdatedAt:   pgtype.Timestamptz{Time: c.UpdatedAt, Valid: true},
		ID:          c.ID,
		Version:     &c.Version,
	})
	if err != nil {
		return err
	}
	if version != nil {
		c.Version = *version
	}
	return nil
}

func (r *campaignRepo) UpdateStatus(ctx context.Context, id int64, status int16) error {
	return r.data.Queries.UpdateCampaignStatus(ctx, &dbsqlc.UpdateCampaignStatusParams{
		Status: &status,
		ID:     id,
	})
}

func (r *campaignRepo) List(ctx context.Context, advertiserID int64, status *int16, page, pageSize int32) ([]biz.Campaign, int64, error) {
	total, err := r.data.Queries.CountCampaigns(ctx, &dbsqlc.CountCampaignsParams{
		AdvertiserID: advertiserID,
		Status:       status,
	})
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.data.Queries.ListCampaigns(ctx, &dbsqlc.ListCampaignsParams{
		AdvertiserID: advertiserID,
		Limit:        pageSize,
		Offset:       offset,
		Status:       status,
	})
	if err != nil {
		return nil, 0, err
	}

	var campaigns []biz.Campaign
	for _, row := range rows {
		campaigns = append(campaigns, *campaignFromDB(row))
	}
	return campaigns, total, nil
}

func campaignFromDB(row *dbsqlc.Campaign) *biz.Campaign {
	return &biz.Campaign{
		ID:           row.ID,
		AdvertiserID: row.AdvertiserID,
		Name:         row.Name,
		Budget:       numericToFloat64(row.Budget),
		DailyBudget:  numericToFloat64(row.DailyBudget),
		StartTime:    timestamptzToTime(row.StartTime),
		EndTime:      timestamptzToTime(row.EndTime),
		Pacing:       ptrInt16(row.Pacing),
		Status:       ptrInt16(row.Status),
		Version:      ptrInt64(row.Version),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}

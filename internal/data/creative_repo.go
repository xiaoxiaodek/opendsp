package data

import (
	"context"
	"encoding/json"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
)

type creativeRepo struct {
	data *Data
}

func NewCreativeRepo(data *Data) biz.CreativeRepo {
	return &creativeRepo{data: data}
}

func (r *creativeRepo) Create(ctx context.Context, c *biz.Creative) error {
	row, err := r.data.Queries.CreateCreative(ctx, &dbsqlc.CreateCreativeParams{
		AdGroupID:          c.AdGroupID,
		Name:               c.Name,
		CreativeType:       c.CreativeType,
		AssetUrl:           c.AssetURL,
		AssetSize:          c.AssetSize,
		AssetDuration:      &c.AssetDuration,
		AssetWidth:         &c.AssetWidth,
		AssetHeight:        &c.AssetHeight,
		AssetMime:          &c.AssetMime,
		Title:              &c.Title,
		Description:        &c.Description,
		CtaText:            &c.CTAText,
		BrandName:          &c.BrandName,
		BrandLogo:          &c.BrandLogo,
		LandingUrl:         c.LandingURL,
		DeeplinkUrl:        &c.DeeplinkURL,
		ImpTracker:         &c.ImpTracker,
		ClickTracker:       &c.ClickTracker,
		ThirdPartyTrackers: c.ThirdPartyTrackers,
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

	if c.AuditStatus != biz.AuditStatusPending {
		if err := r.UpdateAuditStatus(ctx, c.ID, c.AuditStatus, c.AuditReason); err != nil {
			return err
		}
	}
	return nil
}

func (r *creativeRepo) ListByAdGroup(ctx context.Context, adGroupID int64, page, pageSize int32) ([]biz.Creative, int64, error) {
	agID := &adGroupID
	if adGroupID == 0 {
		agID = nil
	}
	total, err := r.data.Queries.CountCreativesByAdGroup(ctx, agID)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.data.Queries.ListCreativesByAdGroup(ctx, &dbsqlc.ListCreativesByAdGroupParams{
		AdGroupID: agID,
		Limit:     pageSize,
		Offset:    offset,
	})
	if err != nil {
		return nil, 0, err
	}

	var creatives []biz.Creative
	for _, row := range rows {
		creatives = append(creatives, creativeFromListRow(row))
	}
	return creatives, total, nil
}

func (r *creativeRepo) ListApprovedByAdGroup(ctx context.Context, adGroupID int64) ([]biz.Creative, error) {
	status := biz.AuditStatusApproved
	rows, err := r.data.Queries.ListApprovedCreativesByAdGroup(ctx, &dbsqlc.ListApprovedCreativesByAdGroupParams{
		AdGroupID:   adGroupID,
		AuditStatus: &status,
	})
	if err != nil {
		return nil, err
	}

	var creatives []biz.Creative
	for _, row := range rows {
		creatives = append(creatives, creativeFromApprovedRow(row))
	}
	return creatives, nil
}

func (r *creativeRepo) Update(ctx context.Context, c *biz.Creative) error {
	_, err := r.data.Pool.Exec(ctx,
		`UPDATE creative SET name=$1, asset_url=$2, asset_width=$3, asset_height=$4, asset_duration=$5,
		 title=$6, description=$7, landing_url=$8, imp_tracker=$9, click_tracker=$10,
		 version=version+1, updated_at=NOW()
		 WHERE id=$11 AND version=$12`,
		c.Name, c.AssetURL, c.AssetWidth, c.AssetHeight, c.AssetDuration,
		c.Title, c.Description, c.LandingURL, c.ImpTracker, c.ClickTracker,
		c.ID, c.Version)
	return err
}

func (r *creativeRepo) SubmitAudit(ctx context.Context, id int64) error {
	status := biz.AuditStatusPending
	return r.data.Queries.SubmitCreativeAudit(ctx, &dbsqlc.SubmitCreativeAuditParams{
		AuditStatus: &status,
		ID:          id,
	})
}

func (r *creativeRepo) UpdateAuditStatus(ctx context.Context, id int64, status int16, reason string) error {
	return r.data.Queries.UpdateCreativeAuditStatus(ctx, &dbsqlc.UpdateCreativeAuditStatusParams{
		AuditStatus: &status,
		AuditReason: &reason,
		ID:          id,
	})
}

func creativeFromListRow(r *dbsqlc.ListCreativesByAdGroupRow) biz.Creative {
	return biz.Creative{
		ID:                 r.ID,
		AdGroupID:          r.AdGroupID,
		Name:               r.Name,
		CreativeType:       r.CreativeType,
		AssetURL:           r.AssetUrl,
		AssetSize:          r.AssetSize,
		AssetDuration:      ptrInt32(r.AssetDuration),
		AssetWidth:         ptrInt32(r.AssetWidth),
		AssetHeight:        ptrInt32(r.AssetHeight),
		AssetMime:          ptrStr(r.AssetMime),
		Title:              ptrStr(r.Title),
		Description:        ptrStr(r.Description),
		CTAText:            ptrStr(r.CtaText),
		BrandName:          ptrStr(r.BrandName),
		BrandLogo:          ptrStr(r.BrandLogo),
		LandingURL:         r.LandingUrl,
		DeeplinkURL:        ptrStr(r.DeeplinkUrl),
		ImpTracker:         ptrStr(r.ImpTracker),
		ClickTracker:       ptrStr(r.ClickTracker),
		ThirdPartyTrackers: json.RawMessage(r.ThirdPartyTrackers),
		AuditStatus:        ptrInt16(r.AuditStatus),
		AuditReason:        ptrStr(r.AuditReason),
		Version:            ptrInt64(r.Version),
		CreatedAt:          r.CreatedAt.Time,
		UpdatedAt:          r.UpdatedAt.Time,
	}
}

func creativeFromApprovedRow(r *dbsqlc.ListApprovedCreativesByAdGroupRow) biz.Creative {
	return biz.Creative{
		ID:                 r.ID,
		AdGroupID:          r.AdGroupID,
		Name:               r.Name,
		CreativeType:       r.CreativeType,
		AssetURL:           r.AssetUrl,
		AssetSize:          r.AssetSize,
		AssetDuration:      ptrInt32(r.AssetDuration),
		AssetWidth:         ptrInt32(r.AssetWidth),
		AssetHeight:        ptrInt32(r.AssetHeight),
		AssetMime:          ptrStr(r.AssetMime),
		Title:              ptrStr(r.Title),
		Description:        ptrStr(r.Description),
		CTAText:            ptrStr(r.CtaText),
		BrandName:          ptrStr(r.BrandName),
		BrandLogo:          ptrStr(r.BrandLogo),
		LandingURL:         r.LandingUrl,
		DeeplinkURL:        ptrStr(r.DeeplinkUrl),
		ImpTracker:         ptrStr(r.ImpTracker),
		ClickTracker:       ptrStr(r.ClickTracker),
		ThirdPartyTrackers: json.RawMessage(r.ThirdPartyTrackers),
		AuditStatus:        ptrInt16(r.AuditStatus),
	}
}

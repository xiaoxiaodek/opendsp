package data

import (
	"context"
	"encoding/json"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
)

type dmpRepo struct {
	data *Data
}

func NewDmpRepo(data *Data) biz.DmpRepo {
	return &dmpRepo{data: data}
}

func (r *dmpRepo) CreateTag(ctx context.Context, tag *biz.DmpTag) (int64, error) {
	sourceConfig, _ := json.Marshal(tag.SourceConfig)
	id, err := r.data.Queries.CreateTag(ctx, &dbsqlc.CreateTagParams{
		AdvertiserID: tag.AdvertiserID,
		Name:         tag.Name,
		TagType:      tag.TagType,
		Source:       &tag.Source,
		SourceConfig: sourceConfig,
		Status:       &tag.Status,
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *dmpRepo) UpdateTagDeviceCount(ctx context.Context, id int64, count int64, status int16) error {
	return r.data.Queries.UpdateTagDeviceCount(ctx, &dbsqlc.UpdateTagDeviceCountParams{
		ID:          id,
		DeviceCount: &count,
		Status:      &status,
	})
}

func (r *dmpRepo) GetTag(ctx context.Context, id int64) (*biz.DmpTag, error) {
	row, err := r.data.Queries.GetTag(ctx, id)
	if err != nil {
		return nil, err
	}
	return &biz.DmpTag{
		ID:           row.ID,
		AdvertiserID: row.AdvertiserID,
		Name:         row.Name,
		TagType:      row.TagType,
		DeviceCount:  ptrInt64(row.DeviceCount),
		Source:       ptrStr(row.Source),
		SourceConfig: row.SourceConfig,
		Status:       ptrInt16(row.Status),
		CreatedAt:    row.CreatedAt.Time,
	}, nil
}

func (r *dmpRepo) ListTags(ctx context.Context, advertiserID int64, tagType *int16) ([]biz.DmpTag, error) {
	var col2 int16
	if tagType != nil {
		col2 = *tagType
	}
	rows, err := r.data.Queries.ListTags(ctx, &dbsqlc.ListTagsParams{
		AdvertiserID: advertiserID,
		Column2:      col2,
	})
	if err != nil {
		return nil, err
	}
	var tags []biz.DmpTag
	for _, row := range rows {
		tags = append(tags, biz.DmpTag{
			ID:           row.ID,
			AdvertiserID: row.AdvertiserID,
			Name:         row.Name,
			TagType:      row.TagType,
			DeviceCount:  ptrInt64(row.DeviceCount),
			Source:       ptrStr(row.Source),
			Status:       ptrInt16(row.Status),
			CreatedAt:    row.CreatedAt.Time,
		})
	}
	return tags, nil
}

func (r *dmpRepo) DeleteTag(ctx context.Context, id int64) error {
	return r.data.Queries.DeleteTag(ctx, id)
}

func (r *dmpRepo) CreateAudience(ctx context.Context, audience *biz.DmpAudience) (int64, error) {
	id, err := r.data.Queries.CreateAudience(ctx, &dbsqlc.CreateAudienceParams{
		AdvertiserID: audience.AdvertiserID,
		Name:         audience.Name,
		AudienceType: &audience.AudienceType,
		Rules:        audience.Rules,
		Status:       &audience.Status,
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *dmpRepo) UpdateAudienceDeviceCount(ctx context.Context, id int64, count int64, status int16) error {
	return r.data.Queries.UpdateAudienceDeviceCount(ctx, &dbsqlc.UpdateAudienceDeviceCountParams{
		ID:          id,
		DeviceCount: &count,
		Status:      &status,
	})
}

func (r *dmpRepo) GetAudience(ctx context.Context, id int64) (*biz.DmpAudience, error) {
	row, err := r.data.Queries.GetAudience(ctx, id)
	if err != nil {
		return nil, err
	}
	return &biz.DmpAudience{
		ID:           row.ID,
		AdvertiserID: row.AdvertiserID,
		Name:         row.Name,
		AudienceType: ptrInt16(row.AudienceType),
		Rules:        row.Rules,
		DeviceCount:  ptrInt64(row.DeviceCount),
		Status:       ptrInt16(row.Status),
		CreatedAt:    row.CreatedAt.Time,
	}, nil
}

func (r *dmpRepo) ListAudiences(ctx context.Context, advertiserID int64, audienceType *int16) ([]biz.DmpAudience, error) {
	var col2 int16
	if audienceType != nil {
		col2 = *audienceType
	}
	rows, err := r.data.Queries.ListAudiences(ctx, &dbsqlc.ListAudiencesParams{
		AdvertiserID: advertiserID,
		Column2:      col2,
	})
	if err != nil {
		return nil, err
	}
	var audiences []biz.DmpAudience
	for _, row := range rows {
		audiences = append(audiences, biz.DmpAudience{
			ID:           row.ID,
			AdvertiserID: row.AdvertiserID,
			Name:         row.Name,
			AudienceType: ptrInt16(row.AudienceType),
			Rules:        row.Rules,
			DeviceCount:  ptrInt64(row.DeviceCount),
			Status:       ptrInt16(row.Status),
			CreatedAt:    row.CreatedAt.Time,
		})
	}
	return audiences, nil
}

func (r *dmpRepo) DeleteAudience(ctx context.Context, id int64) error {
	return r.data.Queries.DeleteAudience(ctx, id)
}

func (r *dmpRepo) UpsertDevice(ctx context.Context, deviceID, deviceType string, tagIDs []int64) error {
	return r.data.Queries.UpsertDevice(ctx, &dbsqlc.UpsertDeviceParams{
		DeviceID:   deviceID,
		DeviceType: deviceType,
		TagIds:     tagIDs,
	})
}

func (r *dmpRepo) GetDeviceTags(ctx context.Context, deviceID, deviceType string) ([]int64, error) {
	return r.data.Queries.GetDeviceTags(ctx, &dbsqlc.GetDeviceTagsParams{
		DeviceID:   deviceID,
		DeviceType: deviceType,
	})
}

package data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
)

type syncRepo struct {
	data *Data
}

func NewSyncRepo(data *Data) biz.SyncRepo {
	return &syncRepo{data: data}
}

func (r *syncRepo) UpsertCreativeSync(ctx context.Context, creativeID int64, platform string, status int16, externalID, externalTvID, reason string, rawResponse []byte) error {
	now := time.Now()
	_, err := r.data.Queries.UpsertCreativeSync(ctx, &dbsqlc.UpsertCreativeSyncParams{
		CreativeID:   creativeID,
		Platform:     platform,
		Status:       status,
		ExternalID:   nullableString(externalID),
		ExternalTvid: nullableString(externalTvID),
		Reason:       nullableString(reason),
		RawResponse:  rawResponse,
		SyncedAt:     pgtype.Timestamptz{Time: now, Valid: true},
	})
	return err
}

func (r *syncRepo) GetCreativeSync(ctx context.Context, creativeID int64, platform string) (*biz.CreativeSyncStatus, error) {
	row, err := r.data.Queries.GetCreativeSync(ctx, &dbsqlc.GetCreativeSyncParams{
		CreativeID: creativeID,
		Platform:   platform,
	})
	if err != nil {
		return nil, err
	}
	return &biz.CreativeSyncStatus{
		ID:           row.ID,
		CreativeID:   row.CreativeID,
		Platform:     row.Platform,
		Status:       row.Status,
		ExternalID:   stringPtr(row.ExternalID),
		ExternalTvID: stringPtr(row.ExternalTvid),
		Reason:       stringPtr(row.Reason),
		RawResponse:  row.RawResponse,
	}, nil
}

func (r *syncRepo) ListPendingCreativeSync(ctx context.Context, platform string) ([]biz.PendingCreativeRow, error) {
	rows, err := r.data.Queries.ListPendingCreativeSync(ctx, platform)
	if err != nil {
		return nil, err
	}
	result := make([]biz.PendingCreativeRow, len(rows))
	for i, row := range rows {
		result[i] = biz.PendingCreativeRow{
			ID:           row.ID,
			CreativeID:   row.CreativeID,
			Platform:     row.Platform,
			Status:       row.Status,
			ExternalID:   row.ExternalID,
			ExternalTvID: row.ExternalTvid,
			Reason:       row.Reason,
		}
	}
	return result, nil
}

func (r *syncRepo) UpsertAdvertiserSync(ctx context.Context, advertiserID int64, platform string, status int16, externalAdID, reason string, rawResponse []byte) error {
	now := time.Now()
	_, err := r.data.Queries.UpsertAdvertiserSync(ctx, &dbsqlc.UpsertAdvertiserSyncParams{
		AdvertiserID:  advertiserID,
		Platform:      platform,
		Status:        status,
		ExternalAdID:  nullableString(externalAdID),
		Reason:        nullableString(reason),
		RawResponse:   rawResponse,
		SyncedAt:      pgtype.Timestamptz{Time: now, Valid: true},
	})
	return err
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

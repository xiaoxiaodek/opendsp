package data

import "context"

type PlatformSyncRepo struct {
	data *Data
}

func NewPlatformSyncRepo(data *Data) *PlatformSyncRepo {
	return &PlatformSyncRepo{data: data}
}

type ApprovedCreativeSync struct {
	CreativeID   int64
	Platform     string
	ExternalTvID string
}

func (r *PlatformSyncRepo) ListApprovedCreativeSync(ctx context.Context) ([]ApprovedCreativeSync, error) {
	rows, err := r.data.Queries.ListApprovedCreativeSync(ctx)
	if err != nil {
		return nil, err
	}
	var result []ApprovedCreativeSync
	for _, row := range rows {
		result = append(result, ApprovedCreativeSync{
			CreativeID:   row.CreativeID,
			Platform:     row.Platform,
			ExternalTvID: ptrStr(row.ExternalTvid),
		})
	}
	return result, nil
}

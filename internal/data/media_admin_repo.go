package data

import (
	"context"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
)

type mediaRepo struct {
	data *Data
}

func NewMediaRepo(data *Data) biz.MediaRepo {
	return &mediaRepo{data: data}
}

func (r *mediaRepo) Create(ctx context.Context, name, code, domain string) (int64, error) {
	return r.data.Queries.CreateMedia(ctx, &dbsqlc.CreateMediaParams{
		Name:   name,
		Code:   code,
		Domain: &domain,
	})
}

func (r *mediaRepo) Update(ctx context.Context, id int64, name, domain *string) error {
	n := ""
	if name != nil {
		n = *name
	}
	return r.data.Queries.UpdateMedia(ctx, &dbsqlc.UpdateMediaParams{
		ID:     id,
		Name:   n,
		Domain: domain,
	})
}

func (r *mediaRepo) UpdateStatus(ctx context.Context, id int64, status int16) error {
	return r.data.Queries.UpdateMediaStatus(ctx, &dbsqlc.UpdateMediaStatusParams{
		ID:     id,
		Status: &status,
	})
}

type adPositionRepo struct {
	data *Data
}

func NewAdPositionRepo(data *Data) biz.AdPositionRepo {
	return &adPositionRepo{data: data}
}

func (r *adPositionRepo) Create(ctx context.Context, mediaID int64, name string, positionType, adFormat int16, width, height, maxSize, durationMin, durationMax int32, mimeTypes string) (int64, error) {
	return r.data.Queries.CreateAdPosition(ctx, &dbsqlc.CreateAdPositionParams{
		MediaID:      mediaID,
		Name:         name,
		PositionType: positionType,
		AdFormat:     adFormat,
		Width:        &width,
		Height:       &height,
		MaxSize:      &maxSize,
		DurationMin:  &durationMin,
		DurationMax:  &durationMax,
		MimeTypes:    &mimeTypes,
	})
}

func (r *adPositionRepo) Update(ctx context.Context, id int64, name *string, width, height, maxSize, durationMin, durationMax *int32) error {
	n := ""
	if name != nil {
		n = *name
	}
	return r.data.Queries.UpdateAdPosition(ctx, &dbsqlc.UpdateAdPositionParams{
		ID:          id,
		Name:        n,
		Width:       width,
		Height:      height,
		MaxSize:     maxSize,
		DurationMin: durationMin,
		DurationMax: durationMax,
	})
}

type adminRepo struct {
	data *Data
}

func NewAdminRepo(data *Data) biz.AdminRepo {
	return &adminRepo{data: data}
}

func (r *adminRepo) ListUsers(ctx context.Context, role *string, page, pageSize int32) ([]biz.User, int64, error) {
	total, err := r.data.Queries.CountUsers(ctx, role)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.data.Queries.ListUsers(ctx, &dbsqlc.ListUsersParams{
		Role:   role,
		Limit:  pageSize,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	result := make([]biz.User, len(rows))
	for i, row := range rows {
		result[i] = biz.User{
			ID:           row.ID,
			Email:        row.Email,
			Name:         row.Name,
			AdvertiserID: row.AdvertiserID,
			Role:         ptrStr(row.Role),
			CreatedAt:    row.CreatedAt.Time,
		}
	}
	return result, total, nil
}

func (r *adminRepo) UpdateUserRole(ctx context.Context, id int64, role string) error {
	return r.data.Queries.UpdateUserRole(ctx, &dbsqlc.UpdateUserRoleParams{
		ID:   id,
		Role: &role,
	})
}

func (r *adminRepo) CreateUser(ctx context.Context, email, passwordHash, name string, advertiserID *int64, role string) (int64, error) {
	return r.data.Queries.CreateUser(ctx, &dbsqlc.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
		Name:         &name,
		AdvertiserID: advertiserID,
		Role:         &role,
	})
}

func (r *adminRepo) ListPendingAudits(ctx context.Context, auditType *int32, page, pageSize int32) ([]biz.PendingAudit, int64, error) {
	total, err := r.data.Queries.CountPendingAudits(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	var result []biz.PendingAudit

	if auditType == nil || *auditType == biz.AuditTypeCreative {
		creativeRows, err := r.data.Queries.ListPendingCreativeAudits(ctx, &dbsqlc.ListPendingCreativeAuditsParams{
			Limit:  pageSize,
			Offset: offset,
		})
		if err != nil {
			return nil, 0, err
		}
		for _, row := range creativeRows {
			result = append(result, biz.PendingAudit{
				ID:             row.ID,
				AuditType:      biz.AuditTypeCreative,
				Name:           row.Name,
				AdvertiserID:   row.AdvertiserID,
				AdvertiserName: row.AdvertiserName,
				Status:         ptrInt16(row.Status),
				Reason:         row.Reason,
				CreatedAt:      row.CreatedAt.Time,
			})
		}
	}

	if auditType == nil || *auditType == biz.AuditTypeAdvertiser {
		advRows, err := r.data.Queries.ListPendingAdvertiserAudits(ctx, &dbsqlc.ListPendingAdvertiserAuditsParams{
			Limit:  pageSize,
			Offset: offset,
		})
		if err != nil {
			return nil, 0, err
		}
		for _, row := range advRows {
			result = append(result, biz.PendingAudit{
				ID:             row.ID,
				AuditType:      biz.AuditTypeAdvertiser,
				Name:           row.Name,
				AdvertiserID:   row.AdvertiserID,
				AdvertiserName: row.AdvertiserName,
				Status:         row.Status,
				Reason:         row.Reason,
				CreatedAt:      row.CreatedAt.Time,
			})
		}
	}

	return result, total, nil
}

package data

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
)

type advertiserRepo struct {
	data *Data
}

func NewAdvertiserRepo(data *Data) biz.AdvertiserRepo {
	return &advertiserRepo{data: data}
}

func (r *advertiserRepo) Create(ctx context.Context, a *biz.Advertiser) error {
	id, err := r.data.Queries.CreateAdvertiser(ctx, &dbsqlc.CreateAdvertiserParams{
		Name:         a.Name,
		Industry:     a.Industry,
		ContactName:  a.ContactName,
		ContactEmail: a.ContactEmail,
		Address:      a.Address,
		Website:      a.Website,
		BrandNames:   a.BrandNames,
	})
	if err != nil {
		return fmt.Errorf("create advertiser: %w", err)
	}
	a.ID = id
	return nil
}

func (r *advertiserRepo) Get(ctx context.Context, id int64) (*biz.Advertiser, error) {
	row, err := r.data.Queries.GetAdvertiser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get advertiser: %w", err)
	}
	return advertiserFromRow(row), nil
}

func (r *advertiserRepo) Update(ctx context.Context, a *biz.Advertiser) error {
	return r.data.Queries.UpdateAdvertiser(ctx, &dbsqlc.UpdateAdvertiserParams{
		ID:           a.ID,
		Name:         a.Name,
		Industry:     a.Industry,
		ContactName:  a.ContactName,
		ContactEmail: a.ContactEmail,
		Address:      a.Address,
		Website:      a.Website,
		BrandNames:   a.BrandNames,
	})
}

func (r *advertiserRepo) List(ctx context.Context, status, qualStatus *int16, page, pageSize int32) ([]biz.Advertiser, int64, error) {
	total, err := r.data.Queries.CountAdvertisers(ctx, &dbsqlc.CountAdvertisersParams{
		Status:              status,
		QualificationStatus: qualStatus,
	})
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.data.Queries.ListAdvertisers(ctx, &dbsqlc.ListAdvertisersParams{
		Status:              status,
		QualificationStatus: qualStatus,
		Limit:               pageSize,
		Offset:              offset,
	})
	if err != nil {
		return nil, 0, err
	}

	result := make([]biz.Advertiser, len(rows))
	for i, row := range rows {
		result[i] = *advertiserFromListRow(row)
	}
	return result, total, nil
}

func (r *advertiserRepo) UpdateQualification(ctx context.Context, id int64, status int16, reason string) error {
	return r.data.Queries.UpdateAdvertiserQualification(ctx, &dbsqlc.UpdateAdvertiserQualificationParams{
		ID:                   id,
		QualificationStatus:  &status,
		QualificationReason:  &reason,
	})
}

func (r *advertiserRepo) Delete(ctx context.Context, id int64) error {
	return r.data.Queries.DeleteAdvertiser(ctx, id)
}

func advertiserFromRow(row *dbsqlc.GetAdvertiserRow) *biz.Advertiser {
	return &biz.Advertiser{
		ID:                  row.ID,
		Name:                row.Name,
		Industry:            row.Industry,
		ContactName:         row.ContactName,
		ContactEmail:        row.ContactEmail,
		Balance:             numericToFloat64Val(row.Balance),
		Status:              ptrInt16(row.Status),
		QualificationStatus: row.QualificationStatus,
		QualificationReason: row.QualificationReason,
		CreditLimit:         row.CreditLimit,
		Address:             row.Address,
		Website:             row.Website,
		BrandNames:          row.BrandNames,
		CreatedAt:           row.CreatedAt.Time,
		UpdatedAt:           row.UpdatedAt.Time,
	}
}

func advertiserFromListRow(row *dbsqlc.ListAdvertisersRow) *biz.Advertiser {
	return &biz.Advertiser{
		ID:                  row.ID,
		Name:                row.Name,
		Industry:            row.Industry,
		ContactName:         row.ContactName,
		ContactEmail:        row.ContactEmail,
		Balance:             numericToFloat64Val(row.Balance),
		Status:              ptrInt16(row.Status),
		QualificationStatus: row.QualificationStatus,
		QualificationReason: row.QualificationReason,
		CreditLimit:         row.CreditLimit,
		Address:             row.Address,
		Website:             row.Website,
		BrandNames:          row.BrandNames,
		CreatedAt:           row.CreatedAt.Time,
		UpdatedAt:           row.UpdatedAt.Time,
	}
}

func numericToFloat64Val(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

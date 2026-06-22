package data

import (
	"context"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
)

type proofMaterialRepo struct {
	data *Data
}

func NewProofMaterialRepo(data *Data) biz.ProofMaterialRepo {
	return &proofMaterialRepo{data: data}
}

func (r *proofMaterialRepo) Create(ctx context.Context, m *biz.ProofMaterial) error {
	return r.data.Queries.CreateProofMaterial(ctx, &dbsqlc.CreateProofMaterialParams{
		AdvertiserID: m.AdvertiserID,
		MaterialType: m.MaterialType,
		FileUrl:      m.FileURL,
		FileName:     m.FileName,
		FileSize:     m.FileSize,
	})
}

func (r *proofMaterialRepo) ListByAdvertiser(ctx context.Context, advertiserID int64) ([]biz.ProofMaterial, error) {
	rows, err := r.data.Queries.ListProofMaterials(ctx, advertiserID)
	if err != nil {
		return nil, err
	}
	result := make([]biz.ProofMaterial, len(rows))
	for i, row := range rows {
		result[i] = biz.ProofMaterial{
			ID:           row.ID,
			AdvertiserID: row.AdvertiserID,
			MaterialType: row.MaterialType,
			FileURL:      row.FileUrl,
			FileName:     row.FileName,
			FileSize:     row.FileSize,
			AuditStatus:  ptrInt16(row.AuditStatus),
			AuditReason:  row.AuditReason,
			CreatedAt:    row.CreatedAt.Time,
		}
	}
	return result, nil
}

package biz

import (
	"context"
	"fmt"
	"time"
)

type AdvertiserUseCase struct {
	repo AdvertiserRepo
}

func NewAdvertiserUseCase(repo AdvertiserRepo) *AdvertiserUseCase {
	return &AdvertiserUseCase{repo: repo}
}

func (uc *AdvertiserUseCase) Create(ctx context.Context, a *Advertiser) error {
	return uc.repo.Create(ctx, a)
}

func (uc *AdvertiserUseCase) Get(ctx context.Context, id int64) (*Advertiser, error) {
	return uc.repo.Get(ctx, id)
}

func (uc *AdvertiserUseCase) Update(ctx context.Context, a *Advertiser) error {
	return uc.repo.Update(ctx, a)
}

func (uc *AdvertiserUseCase) List(ctx context.Context, status, qualStatus *int16, page, pageSize int32) ([]Advertiser, int64, error) {
	return uc.repo.List(ctx, status, qualStatus, page, pageSize)
}

func (uc *AdvertiserUseCase) SubmitQualification(ctx context.Context, id int64) error {
	a, err := uc.repo.Get(ctx, id)
	if err != nil || a == nil {
		return fmt.Errorf("advertiser not found")
	}
	return uc.repo.UpdateQualification(ctx, id, QualificationPending, "")
}

func (uc *AdvertiserUseCase) Audit(ctx context.Context, id int64, status int16, reason string) error {
	if status != QualificationApproved && status != QualificationRejected {
		return fmt.Errorf("invalid qualification status")
	}
	return uc.repo.UpdateQualification(ctx, id, status, reason)
}

func (uc *AdvertiserUseCase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

type ProofMaterialUseCase struct {
	repo ProofMaterialRepo
}

func NewProofMaterialUseCase(repo ProofMaterialRepo) *ProofMaterialUseCase {
	return &ProofMaterialUseCase{repo: repo}
}

func (uc *ProofMaterialUseCase) Upload(ctx context.Context, m *ProofMaterial) error {
	return uc.repo.Create(ctx, m)
}

func (uc *ProofMaterialUseCase) List(ctx context.Context, advertiserID int64) ([]ProofMaterial, error) {
	return uc.repo.ListByAdvertiser(ctx, advertiserID)
}

type BalanceUseCase struct {
	repo BalanceRepo
}

func NewBalanceUseCase(repo BalanceRepo) *BalanceUseCase {
	return &BalanceUseCase{repo: repo}
}

func (uc *BalanceUseCase) Get(ctx context.Context, advertiserID int64) (float64, float64, error) {
	return uc.repo.GetBalance(ctx, advertiserID)
}

func (uc *BalanceUseCase) Recharge(ctx context.Context, advertiserID int64, amount float64, description string, operatorID *int64) (*BalanceTransaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	return uc.repo.Recharge(ctx, advertiserID, amount, description, operatorID)
}

func (uc *BalanceUseCase) ListTransactions(ctx context.Context, advertiserID int64, page, pageSize int32) ([]BalanceTransaction, int64, error) {
	return uc.repo.ListTransactions(ctx, advertiserID, page, pageSize)
}

type MediaUseCase struct {
	repo MediaRepo
}

func NewMediaUseCase(repo MediaRepo) *MediaUseCase {
	return &MediaUseCase{repo: repo}
}

func (uc *MediaUseCase) Create(ctx context.Context, name, code, domain string) (int64, error) {
	return uc.repo.Create(ctx, name, code, domain)
}

func (uc *MediaUseCase) Update(ctx context.Context, id int64, name, domain *string) error {
	return uc.repo.Update(ctx, id, name, domain)
}

func (uc *MediaUseCase) UpdateStatus(ctx context.Context, id int64, status int16) error {
	return uc.repo.UpdateStatus(ctx, id, status)
}

type AdPositionUseCase struct {
	repo AdPositionRepo
}

func NewAdPositionUseCase(repo AdPositionRepo) *AdPositionUseCase {
	return &AdPositionUseCase{repo: repo}
}

func (uc *AdPositionUseCase) Create(ctx context.Context, mediaID int64, name string, positionType, adFormat int16, width, height, maxSize, durationMin, durationMax int32, mimeTypes string) (int64, error) {
	return uc.repo.Create(ctx, mediaID, name, positionType, adFormat, width, height, maxSize, durationMin, durationMax, mimeTypes)
}

func (uc *AdPositionUseCase) Update(ctx context.Context, id int64, name *string, width, height, maxSize, durationMin, durationMax *int32) error {
	return uc.repo.Update(ctx, id, name, width, height, maxSize, durationMin, durationMax)
}

type AdminUseCase struct {
	repo AdminRepo
}

func NewAdminUseCase(repo AdminRepo) *AdminUseCase {
	return &AdminUseCase{repo: repo}
}

func (uc *AdminUseCase) ListUsers(ctx context.Context, role *string, page, pageSize int32) ([]User, int64, error) {
	return uc.repo.ListUsers(ctx, role, page, pageSize)
}

func (uc *AdminUseCase) UpdateUserRole(ctx context.Context, id int64, role string) error {
	return uc.repo.UpdateUserRole(ctx, id, role)
}

func (uc *AdminUseCase) ListPendingAudits(ctx context.Context, auditType *int32, page, pageSize int32) ([]PendingAudit, int64, error) {
	return uc.repo.ListPendingAudits(ctx, auditType, page, pageSize)
}

func (uc *AdminUseCase) CreateUser(ctx context.Context, email, passwordHash, name string, advertiserID *int64, role string) (int64, error) {
	return uc.repo.CreateUser(ctx, email, passwordHash, name, advertiserID, role)
}

func init() {
	_ = time.Now
}

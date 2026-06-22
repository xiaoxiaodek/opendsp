package biz

import "context"

type CreativeUseCase struct {
	repo CreativeRepo
}

func NewCreativeUseCase(repo CreativeRepo) *CreativeUseCase {
	return &CreativeUseCase{repo: repo}
}

func (uc *CreativeUseCase) Create(ctx context.Context, c *Creative) error {
	return uc.repo.Create(ctx, c)
}

func (uc *CreativeUseCase) Get(ctx context.Context, id int64) (*Creative, error) {
	creatives, _, err := uc.repo.ListByAdGroup(ctx, 0, 1, 1000)
	if err != nil {
		return nil, err
	}
	for i := range creatives {
		if creatives[i].ID == id {
			return &creatives[i], nil
		}
	}
	return nil, nil
}

func (uc *CreativeUseCase) Update(ctx context.Context, c *Creative) error {
	return uc.repo.Update(ctx, c)
}

func (uc *CreativeUseCase) List(ctx context.Context, adGroupID int64, page, pageSize int32) ([]Creative, int64, error) {
	return uc.repo.ListByAdGroup(ctx, adGroupID, page, pageSize)
}

func (uc *CreativeUseCase) SubmitAudit(ctx context.Context, id int64) error {
	return uc.repo.SubmitAudit(ctx, id)
}

func (uc *CreativeUseCase) Approve(ctx context.Context, id int64) error {
	return uc.repo.UpdateAuditStatus(ctx, id, AuditStatusApproved, "")
}

func (uc *CreativeUseCase) Reject(ctx context.Context, id int64, reason string) error {
	return uc.repo.UpdateAuditStatus(ctx, id, AuditStatusRejected, reason)
}

func (uc *CreativeUseCase) ListApproved(ctx context.Context, adGroupID int64) ([]Creative, error) {
	return uc.repo.ListApprovedByAdGroup(ctx, adGroupID)
}

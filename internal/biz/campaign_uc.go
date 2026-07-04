package biz

import (
	"context"
	"encoding/json"
)

// CampaignUseCase orchestrates campaign lifecycle operations.
type CampaignUseCase struct {
	repo      CampaignRepo
	publisher EventPublisher
}

func NewCampaignUseCase(repo CampaignRepo, publisher EventPublisher) *CampaignUseCase {
	return &CampaignUseCase{repo: repo, publisher: publisher}
}

func (uc *CampaignUseCase) Create(ctx context.Context, c *Campaign) error {
	if err := uc.repo.Create(ctx, c); err != nil {
		return err
	}
	uc.publish(ctx, "campaign:create", c.ID)
	return nil
}

func (uc *CampaignUseCase) Get(ctx context.Context, id int64) (*Campaign, error) {
	return uc.repo.Get(ctx, id)
}

func (uc *CampaignUseCase) Update(ctx context.Context, c *Campaign) error {
	if err := uc.repo.Update(ctx, c); err != nil {
		return err
	}
	uc.publish(ctx, "campaign:update", c.ID)
	return nil
}

func (uc *CampaignUseCase) Activate(ctx context.Context, id int64) error {
	c, err := uc.repo.Get(ctx, id)
	if err != nil || c == nil {
		return ErrCampaignNotFound
	}
	if err := c.Activate(); err != nil {
		return err
	}
	if err := uc.repo.UpdateStatus(ctx, id, c.Status); err != nil {
		return err
	}
	uc.publish(ctx, "campaign:status", id)
	return nil
}

func (uc *CampaignUseCase) Pause(ctx context.Context, id int64) error {
	c, err := uc.repo.Get(ctx, id)
	if err != nil || c == nil {
		return ErrCampaignNotFound
	}
	if err := c.Pause(); err != nil {
		return err
	}
	if err := uc.repo.UpdateStatus(ctx, id, c.Status); err != nil {
		return err
	}
	uc.publish(ctx, "campaign:status", id)
	return nil
}

func (uc *CampaignUseCase) List(ctx context.Context, advertiserID int64, status *int16, page, pageSize int32) ([]Campaign, int64, error) {
	return uc.repo.List(ctx, advertiserID, status, page, pageSize)
}

func (uc *CampaignUseCase) publish(ctx context.Context, eventType string, id int64) {
	if uc.publisher == nil {
		return
	}
	data, _ := json.Marshal(map[string]interface{}{"type": eventType, "id": id})
	uc.publisher.Publish(ctx, "ad:change", data)
}

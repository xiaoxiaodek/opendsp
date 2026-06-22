package biz

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type AdGroupUseCase struct {
	repo AdGroupRepo
	rdb  *redis.Client
}

func NewAdGroupUseCase(repo AdGroupRepo, rdb *redis.Client) *AdGroupUseCase {
	return &AdGroupUseCase{repo: repo, rdb: rdb}
}

func (uc *AdGroupUseCase) Create(ctx context.Context, ag *AdGroup) error {
	if err := uc.repo.Create(ctx, ag); err != nil {
		return err
	}
	uc.publish(ctx, "adgroup:create", ag.ID)
	return nil
}

func (uc *AdGroupUseCase) Get(ctx context.Context, id int64) (*AdGroup, error) {
	return uc.repo.Get(ctx, id)
}

func (uc *AdGroupUseCase) Update(ctx context.Context, ag *AdGroup) error {
	if err := uc.repo.Update(ctx, ag); err != nil {
		return err
	}
	uc.publish(ctx, "adgroup:update", ag.ID)
	return nil
}

func (uc *AdGroupUseCase) Activate(ctx context.Context, id int64) error {
	ag, err := uc.repo.Get(ctx, id)
	if err != nil || ag == nil {
		return ErrAdGroupNotFound
	}
	if err := ag.Activate(); err != nil {
		return err
	}
	if err := uc.repo.UpdateStatus(ctx, id, ag.Status); err != nil {
		return err
	}
	uc.publish(ctx, "adgroup:status", id)
	return nil
}

func (uc *AdGroupUseCase) Pause(ctx context.Context, id int64) error {
	ag, err := uc.repo.Get(ctx, id)
	if err != nil || ag == nil {
		return ErrAdGroupNotFound
	}
	if err := ag.Pause(); err != nil {
		return err
	}
	if err := uc.repo.UpdateStatus(ctx, id, ag.Status); err != nil {
		return err
	}
	uc.publish(ctx, "adgroup:status", id)
	return nil
}

func (uc *AdGroupUseCase) List(ctx context.Context, campaignID int64, status *int16, page, pageSize int32) ([]AdGroup, int64, error) {
	return uc.repo.List(ctx, campaignID, status, page, pageSize)
}

func (uc *AdGroupUseCase) ListActive(ctx context.Context) ([]AdGroup, error) {
	return uc.repo.ListActive(ctx)
}

func (uc *AdGroupUseCase) publish(ctx context.Context, eventType string, id int64) {
	data, _ := json.Marshal(map[string]interface{}{"type": eventType, "id": id})
	uc.rdb.Publish(ctx, "ad:change", data)
}

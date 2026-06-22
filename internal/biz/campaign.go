package biz

import "time"

type Campaign struct {
	ID           int64
	AdvertiserID int64
	Name         string
	Budget       *float64
	DailyBudget  *float64
	StartTime    *time.Time
	EndTime      *time.Time
	Pacing       int16
	Status       int16
	Version      int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

const (
	CampaignStatusDraft     int16 = 0
	CampaignStatusActive    int16 = 1
	CampaignStatusPaused    int16 = 2
	CampaignStatusCompleted int16 = 3
)

func (c *Campaign) Activate() error {
	if c.Status == CampaignStatusCompleted {
		return ErrCampaignCompleted
	}
	c.Status = CampaignStatusActive
	return nil
}

func (c *Campaign) Pause() error {
	if c.Status != CampaignStatusActive {
		return ErrCampaignNotActive
	}
	c.Status = CampaignStatusPaused
	return nil
}

func (c *Campaign) Complete() error {
	c.Status = CampaignStatusCompleted
	return nil
}

func (c *Campaign) IsActive() bool {
	return c.Status == CampaignStatusActive
}

func (c *Campaign) CanUpdateBudget(newBudget float64) bool {
	return newBudget >= 0
}

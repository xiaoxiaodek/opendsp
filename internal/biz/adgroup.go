package biz

import (
	"encoding/json"
	"time"
)

type AdGroup struct {
	ID                 int64
	CampaignID         int64
	AdvertiserID       int64
	Name               string
	BidType            int16
	BidPrice           float64
	DailyBudget        *float64
	FreqCap            *int32
	FreqPeriod         int32
	Targeting          json.RawMessage
	Status             int16
	Version            int64
	CampaignStartTime  *time.Time
	CampaignEndTime    *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

const (
	BidTypeCPM int16 = 1
	BidTypeCPC int16 = 2
	BidTypeCPV int16 = 3
	BidTypeCPA int16 = 4
)

func (ag *AdGroup) Activate() error {
	ag.Status = CampaignStatusActive
	return nil
}

func (ag *AdGroup) Pause() error {
	if ag.Status != CampaignStatusActive {
		return ErrCampaignNotActive
	}
	ag.Status = CampaignStatusPaused
	return nil
}

func (ag *AdGroup) IsActive() bool {
	return ag.Status == CampaignStatusActive
}

func (ag *AdGroup) GetTargeting() (*Targeting, error) {
	var t Targeting
	if err := json.Unmarshal(ag.Targeting, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

type Targeting struct {
	Geo       *GeoTargeting       `json:"geo,omitempty"`
	Device    *DeviceTargeting    `json:"device,omitempty"`
	Inventory *InventoryTargeting `json:"inventory,omitempty"`
	Audience  *AudienceTargeting  `json:"audience,omitempty"`
	Time      *TimeTargeting      `json:"time,omitempty"`
	Context   *ContextTargeting   `json:"context,omitempty"`
}

type GeoTargeting struct {
	Country []string `json:"country,omitempty"`
	City    []string `json:"city,omitempty"`
	Region  []string `json:"region,omitempty"`
}

type DeviceTargeting struct {
	OS         []string `json:"os,omitempty"`
	DeviceType []string `json:"device_type,omitempty"`
	Carrier    []string `json:"carrier,omitempty"`
	Make       []string `json:"make,omitempty"`
}

type InventoryTargeting struct {
	Media           []string `json:"media,omitempty"`
	MediaBlacklist  []string `json:"media_blacklist,omitempty"`
	AdPosition      []string `json:"ad_position,omitempty"`
	ContentCategory []string `json:"content_category,omitempty"`
}

type AudienceTargeting struct {
	Gender      string  `json:"gender,omitempty"`
	AgeRange    []int32 `json:"age_range,omitempty"`
	DmpIDs      []int64 `json:"dmp_ids,omitempty"`
	DevicePrice []int32 `json:"device_price,omitempty"`
}

type TimeTargeting struct {
	DayOfWeek []int32 `json:"day_of_week,omitempty"`
	HourRange []int32 `json:"hour_range,omitempty"`
}

type ContextTargeting struct {
	Keywords      []string `json:"keywords,omitempty"`
	IABCategories []string `json:"iab_categories,omitempty"`
}

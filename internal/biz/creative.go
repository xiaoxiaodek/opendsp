package biz

import (
	"encoding/json"
	"time"
)

type Creative struct {
	ID                 int64
	AdGroupID          int64
	Name               string
	CreativeType       int16
	AssetURL           string
	AssetSize          *int32
	AssetDuration      int32
	AssetWidth         int32
	AssetHeight        int32
	AssetMime          string
	Title              string
	Description        string
	CTAText            string
	BrandName          string
	BrandLogo          string
	LandingURL         string
	DeeplinkURL        string
	ImpTracker         string
	ClickTracker       string
	ThirdPartyTrackers json.RawMessage
	AuditStatus        int16
	AuditReason        string
	IsValid            int16
	Version            int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

const (
	CreativeTypeImage  int16 = 1
	CreativeTypeVideo  int16 = 2
	CreativeTypeNative int16 = 3
	CreativeTypeAudio  int16 = 4
	CreativeTypeHTML   int16 = 5

	AuditStatusPending  int16 = 0
	AuditStatusApproved int16 = 1
	AuditStatusRejected int16 = 2
)

func (c *Creative) SubmitForAudit() {
	c.AuditStatus = AuditStatusPending
}

func (c *Creative) Approve() {
	c.AuditStatus = AuditStatusApproved
}

func (c *Creative) Reject(reason string) {
	c.AuditStatus = AuditStatusRejected
	c.AuditReason = reason
}

func (c *Creative) IsApproved() bool {
	return c.AuditStatus == AuditStatusApproved && c.IsValid == 1
}

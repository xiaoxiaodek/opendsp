package biz

import (
	"context"
	"time"
)

type CampaignRepo interface {
	Create(ctx context.Context, c *Campaign) error
	Get(ctx context.Context, id int64) (*Campaign, error)
	Update(ctx context.Context, c *Campaign) error
	UpdateStatus(ctx context.Context, id int64, status int16) error
	List(ctx context.Context, advertiserID int64, status *int16, page, pageSize int32) ([]Campaign, int64, error)
}

type AdGroupRepo interface {
	Create(ctx context.Context, ag *AdGroup) error
	Get(ctx context.Context, id int64) (*AdGroup, error)
	Update(ctx context.Context, ag *AdGroup) error
	UpdateStatus(ctx context.Context, id int64, status int16) error
	List(ctx context.Context, campaignID int64, status *int16, page, pageSize int32) ([]AdGroup, int64, error)
	ListActive(ctx context.Context) ([]AdGroup, error)
}

type CreativeRepo interface {
	Create(ctx context.Context, c *Creative) error
	Update(ctx context.Context, c *Creative) error
	ListByAdGroup(ctx context.Context, adGroupID int64, page, pageSize int32) ([]Creative, int64, error)
	ListApprovedByAdGroup(ctx context.Context, adGroupID int64) ([]Creative, error)
	SubmitAudit(ctx context.Context, id int64) error
	UpdateAuditStatus(ctx context.Context, id int64, status int16, reason string) error
}

type ReportRepo interface {
	InsertEvents(ctx context.Context, events []StatEvent) error
	AggregateHourly(ctx context.Context, start, end interface{}) error
	Query(ctx context.Context, advertiserID int64, campaignID, adGroupID *int64, start, end interface{}) ([]ReportHourly, error)
}

type AdvertiserRepo interface {
	Create(ctx context.Context, a *Advertiser) error
	Get(ctx context.Context, id int64) (*Advertiser, error)
	Update(ctx context.Context, a *Advertiser) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, status, qualStatus *int16, page, pageSize int32) ([]Advertiser, int64, error)
	UpdateQualification(ctx context.Context, id int64, status int16, reason string) error
}

type ProofMaterialRepo interface {
	Create(ctx context.Context, m *ProofMaterial) error
	ListByAdvertiser(ctx context.Context, advertiserID int64) ([]ProofMaterial, error)
}

type BalanceRepo interface {
	GetBalance(ctx context.Context, advertiserID int64) (float64, float64, error)
	Recharge(ctx context.Context, advertiserID int64, amount float64, description string, operatorID *int64) (*BalanceTransaction, error)
	ListTransactions(ctx context.Context, advertiserID int64, page, pageSize int32) ([]BalanceTransaction, int64, error)
}

type MediaRepo interface {
	Create(ctx context.Context, name, code, domain string) (int64, error)
	Update(ctx context.Context, id int64, name, domain *string) error
	UpdateStatus(ctx context.Context, id int64, status int16) error
}

type AdPositionRepo interface {
	Create(ctx context.Context, mediaID int64, name string, positionType, adFormat int16, width, height, maxSize, durationMin, durationMax int32, mimeTypes string) (int64, error)
	Update(ctx context.Context, id int64, name *string, width, height, maxSize, durationMin, durationMax *int32) error
}

type AdminRepo interface {
	ListUsers(ctx context.Context, role *string, page, pageSize int32) ([]User, int64, error)
	UpdateUserRole(ctx context.Context, id int64, role string) error
	CreateUser(ctx context.Context, email, passwordHash, name string, advertiserID *int64, role string) (int64, error)
	ListPendingAudits(ctx context.Context, auditType *int32, page, pageSize int32) ([]PendingAudit, int64, error)
}

type User struct {
	ID           int64
	Email        string
	Name         *string
	AdvertiserID *int64
	Role         string
	CreatedAt    time.Time
}

type DmpRepo interface {
	CreateTag(ctx context.Context, tag *DmpTag) (int64, error)
	UpdateTagDeviceCount(ctx context.Context, id int64, count int64, status int16) error
	GetTag(ctx context.Context, id int64) (*DmpTag, error)
	ListTags(ctx context.Context, advertiserID int64, tagType *int16) ([]DmpTag, error)
	DeleteTag(ctx context.Context, id int64) error

	CreateAudience(ctx context.Context, audience *DmpAudience) (int64, error)
	UpdateAudienceDeviceCount(ctx context.Context, id int64, count int64, status int16) error
	GetAudience(ctx context.Context, id int64) (*DmpAudience, error)
	ListAudiences(ctx context.Context, advertiserID int64, audienceType *int16) ([]DmpAudience, error)
	DeleteAudience(ctx context.Context, id int64) error

	UpsertDevice(ctx context.Context, deviceID, deviceType string, tagIDs []int64) error
	GetDeviceTags(ctx context.Context, deviceID, deviceType string) ([]int64, error)
}

type SyncRepo interface {
	UpsertCreativeSync(ctx context.Context, creativeID int64, platform string, status int16, externalID, externalTvID, reason string, rawResponse []byte) error
	GetCreativeSync(ctx context.Context, creativeID int64, platform string) (*CreativeSyncStatus, error)
	ListPendingCreativeSync(ctx context.Context, platform string) ([]PendingCreativeRow, error)
	UpsertAdvertiserSync(ctx context.Context, advertiserID int64, platform string, status int16, externalAdID, reason string, rawResponse []byte) error
}

type CreativeSyncStatus struct {
	ID           int64
	CreativeID   int64
	Platform     string
	Status       int16
	ExternalID   string
	ExternalTvID string
	Reason       string
	RawResponse  []byte
}

type PendingCreativeRow struct {
	ID           int64
	CreativeID   int64
	Platform     string
	Status       int16
	ExternalID   *string
	ExternalTvID *string
	Reason       *string
}

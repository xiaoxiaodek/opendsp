package biz

import "time"

const (
	QualificationPending  int16 = 0
	QualificationApproved int16 = 1
	QualificationRejected int16 = 2

	AdvertiserStatusActive   int16 = 1
	AdvertiserStatusDisabled int16 = 2
)

type Advertiser struct {
	ID                   int64
	Name                 string
	Industry             *string
	ContactName          *string
	ContactEmail         *string
	Balance              float64
	Status               int16
	QualificationStatus  int16
	QualificationReason  *string
	CreditLimit          float64
	Address              *string
	Website              *string
	BrandNames           *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

const (
	ProofMaterialTypeLicense   int16 = 1
	ProofMaterialTypeIDCard    int16 = 2
	ProofMaterialTypeTax       int16 = 3
	ProofMaterialTypeOther     int16 = 4

	ProofAuditPending  int16 = 0
	ProofAuditApproved int16 = 1
	ProofAuditRejected int16 = 2
)

type ProofMaterial struct {
	ID           int64
	AdvertiserID int64
	MaterialType int16
	FileURL      string
	FileName     *string
	FileSize     *int32
	AuditStatus  int16
	AuditReason  *string
	CreatedAt    time.Time
}

const (
	TxTypeRecharge int16 = 1
	TxTypeConsume  int16 = 2
	TxTypeRefund   int16 = 3
)

type BalanceTransaction struct {
	ID            int64
	AdvertiserID  int64
	Amount        float64
	BalanceBefore float64
	BalanceAfter  float64
	TxType        int16
	Description   *string
	OperatorID    *int64
	CreatedAt     time.Time
}

const (
	AuditTypeCreative   int32 = 1
	AuditTypeAdvertiser int32 = 2
)

type PendingAudit struct {
	ID             int64
	AuditType      int32
	Name           string
	AdvertiserID   int64
	AdvertiserName string
	Status         int16
	Reason         *string
	CreatedAt      time.Time
}

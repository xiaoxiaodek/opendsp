package biz

import (
	"encoding/json"
	"time"
)

type DmpTag struct {
	ID           int64
	AdvertiserID int64
	Name         string
	TagType      int16
	DeviceCount  int64
	Source       string
	SourceConfig json.RawMessage
	Status       int16
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

const (
	TagTypeUpload    int16 = 1
	TagTypeBehavior  int16 = 2
	TagTypeLookalike int16 = 3

	TagStatusComputing int16 = 1
	TagStatusReady     int16 = 2
	TagStatusInvalid   int16 = 3
)

type DmpAudience struct {
	ID           int64
	AdvertiserID int64
	Name         string
	AudienceType int16
	Rules        json.RawMessage
	DeviceCount  int64
	Status       int16
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

const (
	AudienceTypeTagCombo  int16 = 1
	AudienceTypeUpload    int16 = 2
	AudienceTypeBehavior  int16 = 3
	AudienceTypeLookalike int16 = 4

	AudienceStatusComputing int16 = 1
	AudienceStatusReady     int16 = 2
	AudienceStatusInvalid   int16 = 3
)

type DmpDevice struct {
	DeviceID   string
	DeviceType string
	TagIDs     []int64
	FirstSeen  time.Time
	LastSeen   time.Time
}

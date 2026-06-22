package biz

import "time"

type StatEvent struct {
	ID           int64
	EventType    int16
	AdGroupID    int64
	CreativeID   int64
	CampaignID   int64
	AdvertiserID int64
	MediaID      int64
	AdPositionID int64
	Price        float64
	ChargeType   *int16
	DeviceID     string
	IP           string
	UA           string
	GeoCity      string
	FreqResult   string
	ClickID      string
	EventTime    time.Time
	CreatedAt    time.Time
}

type ConversionEvent struct {
	ID            int64
	ClickID       string
	EventType     string
	AdGroupID     *int64
	CreativeID    *int64
	CampaignID    *int64
	AdvertiserID  *int64
	MediaID       *int64
	AdPositionID  *int64
	Price         float64
	Revenue       float64
	DeviceID      string
	IP            string
	UA            string
	GeoCity       string
	Extra         []byte
	EventTime     time.Time
	CreatedAt     time.Time
}

const (
	EventImpression   int16 = 1
	EventClick        int16 = 2
	EventConversion   int16 = 3
	EventPlayProgress int16 = 4
)

type ReportHourly struct {
	ID           int64
	Hour         time.Time
	AdvertiserID int64
	CampaignID   int64
	AdGroupID    int64
	CreativeID   int64
	MediaID      int64
	AdPositionID int64
	Impressions  int64
	Clicks       int64
	Conversions  int64
	Revenue      float64
	Cost         float64
	WinCount     int64
	BidCount     int64
}

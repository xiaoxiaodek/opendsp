// Package kafka provides Kafka event types and async producer for DSP events.
package kafka

import "time"

// BidEvent is emitted for every bid response (win or no-bid).
type BidEvent struct {
	EventTime    time.Time `json:"event_time"`
	RequestID    string    `json:"request_id"`
	MediaID      string    `json:"media_id"`
	PositionType int32     `json:"position_type"`
	AdGroupID    uint32    `json:"adgroup_id"`
	BidPrice     float64   `json:"bid_price"`
	ECPM         int64     `json:"ecpm"`
	PredCTR      float64   `json:"pctr"`
	PredCVR      float64   `json:"pcvr"`
	Won          bool      `json:"won"`
	StageDropped string    `json:"stage_dropped,omitempty"`
	LatencyMs    float64   `json:"latency_ms"`
}

// ImpressionEvent is emitted when an impression is tracked.
type ImpressionEvent struct {
	EventTime    time.Time `json:"event_time"`
	ClickID      string    `json:"click_id"`
	AdGroupID    int64     `json:"adgroup_id"`
	CreativeID   int64     `json:"creative_id"`
	CampaignID   int64     `json:"campaign_id"`
	AdvertiserID int64     `json:"advertiser_id"`
	MediaID      string    `json:"media_id"`
	Cost         float64   `json:"cost"`
	DeviceID     string    `json:"device_id"`
	GeoCity      string    `json:"geo_city"`
}

// ClickEvent is emitted when a click is tracked.
type ClickEvent struct {
	EventTime    time.Time `json:"event_time"`
	ClickID      string    `json:"click_id"`
	AdGroupID    int64     `json:"adgroup_id"`
	CreativeID   int64     `json:"creative_id"`
	CampaignID   int64     `json:"campaign_id"`
	AdvertiserID int64     `json:"advertiser_id"`
	MediaID      string    `json:"media_id"`
	DeviceID     string    `json:"device_id"`
	GeoCity      string    `json:"geo_city"`
}

// ConversionEvent is emitted when a conversion postback is received.
type ConversionEvent struct {
	EventTime    time.Time `json:"event_time"`
	ClickID      string    `json:"click_id"`
	EventType    string    `json:"event_type"`
	AdGroupID    int64     `json:"adgroup_id"`
	CreativeID   int64     `json:"creative_id"`
	CampaignID   int64     `json:"campaign_id"`
	AdvertiserID int64     `json:"advertiser_id"`
	MediaID      string    `json:"media_id"`
	Revenue      float64   `json:"revenue"`
	Price        float64   `json:"price"`
	DeviceID     string    `json:"device_id"`
	GeoCity      string    `json:"geo_city"`
}

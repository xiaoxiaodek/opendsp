package clickhouse

import (
	"context"
	"fmt"
	"log"

	kafkaevents "github.com/opendsp/opendsp/internal/infrastructure/messaging/kafka"
)

// Writer handles batch insertion of DSP events into ClickHouse.
type Writer struct {
	client *Client
}

// NewWriter creates a ClickHouse event writer.
func NewWriter(client *Client) *Writer {
	return &Writer{client: client}
}

// EnsureTables creates the ClickHouse tables if they don't exist.
func (w *Writer) EnsureTables(ctx context.Context) error {
	ddls := []string{
		`CREATE TABLE IF NOT EXISTS bid_events (
			event_time    DateTime,
			request_id    String,
			media_id      String,
			position_type Int32,
			adgroup_id    Int64,
			bid_price     Decimal(12,6),
			ecpm          Int64,
			pctr          Decimal(10,8),
			pcvr          Decimal(10,8),
			won           UInt8,
			stage_dropped String,
			latency_ms    Float64
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(event_time)
		ORDER BY (media_id, event_time)`,

		`CREATE TABLE IF NOT EXISTS impression_events (
			event_time    DateTime,
			click_id      String,
			adgroup_id    Int64,
			creative_id   Int64,
			campaign_id   Int64,
			advertiser_id Int64,
			media_id      String,
			cost          Decimal(12,6),
			device_id     String,
			geo_city      String
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(event_time)
		ORDER BY (advertiser_id, event_time)`,

		`CREATE TABLE IF NOT EXISTS click_events (
			event_time    DateTime,
			click_id      String,
			adgroup_id    Int64,
			creative_id   Int64,
			campaign_id   Int64,
			advertiser_id Int64,
			media_id      String,
			device_id     String,
			geo_city      String
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(event_time)
		ORDER BY (advertiser_id, event_time)`,

		`CREATE TABLE IF NOT EXISTS conversion_events (
			event_time    DateTime,
			click_id      String,
			event_type    String,
			adgroup_id    Int64,
			creative_id   Int64,
			campaign_id   Int64,
			advertiser_id Int64,
			media_id      String,
			revenue       Decimal(12,6),
			price         Decimal(12,6),
			device_id     String,
			geo_city      String
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(event_time)
		ORDER BY (advertiser_id, event_time)`,

		`CREATE TABLE IF NOT EXISTS settlement_events (
			event_time    DateTime,
			dsp_bid_id    String,
			adx_bid_id    String,
			media_id      String,
			advertiser_id Int64,
			adgroup_id    Int64,
			dsp_cost      Decimal(18,6),
			adx_cost      Decimal(18,6),
			currency      String,
			discrepancy   Decimal(18,6)
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(event_time)
		ORDER BY (advertiser_id, event_time)`,
	}

	for _, ddl := range ddls {
		if _, err := w.client.DB().ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("clickhouse ddl: %w", err)
		}
	}
	log.Println("clickhouse: tables ensured")
	return nil
}

// WriteBidEvent inserts a bid event into ClickHouse.
func (w *Writer) WriteBidEvent(ctx context.Context, event kafkaevents.BidEvent) error {
	query := `INSERT INTO bid_events (event_time, request_id, media_id, position_type,
		adgroup_id, bid_price, ecpm, pctr, pcvr, won, stage_dropped, latency_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	won := uint8(0)
	if event.Won {
		won = 1
	}

	_, err := w.client.DB().ExecContext(ctx, query,
		event.EventTime, event.RequestID, event.MediaID, event.PositionType,
		event.AdGroupID, event.BidPrice, event.ECPM, event.PredCTR, event.PredCVR,
		won, event.StageDropped, event.LatencyMs,
	)
	return err
}

// WriteImpressionEvent inserts an impression event.
func (w *Writer) WriteImpressionEvent(ctx context.Context, event kafkaevents.ImpressionEvent) error {
	query := `INSERT INTO impression_events (event_time, click_id, adgroup_id, creative_id,
		campaign_id, advertiser_id, media_id, cost, device_id, geo_city)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := w.client.DB().ExecContext(ctx, query,
		event.EventTime, event.ClickID, event.AdGroupID, event.CreativeID,
		event.CampaignID, event.AdvertiserID, event.MediaID, event.Cost,
		event.DeviceID, event.GeoCity,
	)
	return err
}

// WriteClickEvent inserts a click event.
func (w *Writer) WriteClickEvent(ctx context.Context, event kafkaevents.ClickEvent) error {
	query := `INSERT INTO click_events (event_time, click_id, adgroup_id, creative_id,
		campaign_id, advertiser_id, media_id, device_id, geo_city)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := w.client.DB().ExecContext(ctx, query,
		event.EventTime, event.ClickID, event.AdGroupID, event.CreativeID,
		event.CampaignID, event.AdvertiserID, event.MediaID,
		event.DeviceID, event.GeoCity,
	)
	return err
}

// WriteConversionEvent inserts a conversion event.
func (w *Writer) WriteConversionEvent(ctx context.Context, event kafkaevents.ConversionEvent) error {
	query := `INSERT INTO conversion_events (event_time, click_id, event_type, adgroup_id,
		creative_id, campaign_id, advertiser_id, media_id, revenue, price, device_id, geo_city)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := w.client.DB().ExecContext(ctx, query,
		event.EventTime, event.ClickID, event.EventType, event.AdGroupID,
		event.CreativeID, event.CampaignID, event.AdvertiserID, event.MediaID,
		event.Revenue, event.Price, event.DeviceID, event.GeoCity,
	)
	return err
}

// Health checks the ClickHouse connection.
func (w *Writer) Health(ctx context.Context) error {
	return w.client.DB().PingContext(ctx)
}

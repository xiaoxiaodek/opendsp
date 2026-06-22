package data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
)

type reportRepo struct {
	data *Data
}

func NewReportRepo(data *Data) biz.ReportRepo {
	return &reportRepo{data: data}
}

func (r *reportRepo) InsertEvents(ctx context.Context, events []biz.StatEvent) error {
	batch := &pgx.Batch{}
	for _, e := range events {
		batch.Queue(
			`INSERT INTO stat_event (event_type, adgroup_id, creative_id, campaign_id, advertiser_id, media_id, ad_position_id, price, charge_type, device_id, ip, ua, geo_city, freq_result, click_id, event_time)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
			e.EventType, e.AdGroupID, e.CreativeID, e.CampaignID, e.AdvertiserID, e.MediaID, e.AdPositionID, e.Price, e.ChargeType, e.DeviceID, e.IP, e.UA, e.GeoCity, e.FreqResult, e.ClickID, e.EventTime,
		)
	}
	br := r.data.Pool.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < len(events); i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *reportRepo) AggregateHourly(ctx context.Context, start, end interface{}) error {
	s := start.(time.Time)
	e := end.(time.Time)
	return r.data.Queries.AggregateReportHourly(ctx, &dbsqlc.AggregateReportHourlyParams{
		EventTime:   s,
		EventTime_2: e,
	})
}

func (r *reportRepo) Query(ctx context.Context, advertiserID int64, campaignID, adGroupID *int64, start, end interface{}) ([]biz.ReportHourly, error) {
	s := start.(time.Time)
	e := end.(time.Time)
	rows, err := r.data.Queries.QueryReport(ctx, &dbsqlc.QueryReportParams{
		AdvertiserID: advertiserID,
		Hour:         s,
		Hour_2:       e,
		CampaignID:   campaignID,
		AdGroupID:    adGroupID,
	})
	if err != nil {
		return nil, err
	}

	var reports []biz.ReportHourly
	for _, row := range rows {
		reports = append(reports, biz.ReportHourly{
			Hour:         row.Hour,
			AdvertiserID: row.AdvertiserID,
			CampaignID:   row.CampaignID,
			AdGroupID:    row.AdGroupID,
			CreativeID:   row.CreativeID,
			MediaID:      row.MediaID,
			AdPositionID: row.AdPositionID,
			Impressions:  ptrInt64(row.Impressions),
			Clicks:       ptrInt64(row.Clicks),
			Conversions:  ptrInt64(row.Conversions),
			Revenue:      ptrFloat64(numericToFloat64(row.Revenue)),
			Cost:         ptrFloat64(numericToFloat64(row.Cost)),
			WinCount:     ptrInt64(row.WinCount),
			BidCount:     ptrInt64(row.BidCount),
		})
	}
	return reports, nil
}

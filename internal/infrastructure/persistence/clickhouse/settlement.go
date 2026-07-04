package clickhouse

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"
)

type SettlementRow struct {
	Date          time.Time `json:"date"`
	AdvertiserID  int64     `json:"advertiser_id"`
	DSPCost       float64   `json:"dsp_cost"`
	ADXCost       float64   `json:"adx_cost"`
	Difference    float64   `json:"difference"`
	DifferencePct float64   `json:"difference_pct"`
}

type SettlementSummary struct {
	TotalDSPCost     float64         `json:"total_dsp_cost"`
	TotalADXCost     float64         `json:"total_adx_cost"`
	TotalDiscrepancy float64         `json:"total_discrepancy"`
	Discrepancies    []SettlementRow `json:"discrepancies"`
}

func (w *Writer) Reconcile(ctx context.Context, advertiserID int64, start, end time.Time) (*SettlementSummary, error) {
	query := `
		SELECT
			toDate(event_time) as date,
			SUM(toFloat64(dsp_cost)) as dsp_cost,
			SUM(toFloat64(adx_cost)) as adx_cost,
			SUM(toFloat64(dsp_cost) - toFloat64(adx_cost)) as difference,
			CASE WHEN SUM(toFloat64(dsp_cost)) > 0
				THEN (SUM(toFloat64(dsp_cost) - toFloat64(adx_cost)) / SUM(toFloat64(dsp_cost))) * 100
				ELSE 0
			END as difference_pct
		FROM settlement_events
		WHERE advertiser_id = $1 AND event_time >= $2 AND event_time <= $3
		GROUP BY date ORDER BY date
	`

	rows, err := w.client.DB().QueryContext(ctx, query, advertiserID, start, end)
	if err != nil {
		return nil, fmt.Errorf("settlement reconcile: %w", err)
	}
	defer rows.Close()

	var summary SettlementSummary
	for rows.Next() {
		var row SettlementRow
		if err := rows.Scan(&row.Date, &row.DSPCost, &row.ADXCost, &row.Difference, &row.DifferencePct); err != nil {
			continue
		}
		row.AdvertiserID = advertiserID
		summary.Discrepancies = append(summary.Discrepancies, row)
		summary.TotalDSPCost += row.DSPCost
		summary.TotalADXCost += row.ADXCost
		summary.TotalDiscrepancy += row.Difference
	}

	return &summary, nil
}

func (w *Writer) ImportCSV(ctx context.Context, reader io.Reader) (int, error) {
	csvReader := csv.NewReader(reader)
	records, err := csvReader.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("settlement csv read: %w", err)
	}
	if len(records) < 2 {
		return 0, fmt.Errorf("settlement csv: empty or header only")
	}

	query := `INSERT INTO settlement_events
		(event_time, adx_bid_id, media_id, advertiser_id, adgroup_id, adx_cost, currency, dsp_cost, discrepancy)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0)`

	count := 0
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) < 7 {
			continue
		}

		date, _ := time.Parse("2006-01-02", record[0])
		advertiserID, _ := strconv.ParseInt(record[3], 10, 64)
		adgroupID, _ := strconv.ParseInt(record[4], 10, 64)
		adxCost, _ := strconv.ParseFloat(record[5], 64)

		_, err := w.client.DB().ExecContext(ctx, query,
			date, record[1], record[2], advertiserID, adgroupID, adxCost, record[6],
		)
		if err != nil {
			return count, fmt.Errorf("settlement csv row %d: %w", i, err)
		}
		count++
	}

	return count, nil
}

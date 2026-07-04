package adserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/opendsp/opendsp/internal/data"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
	postgresRoi "github.com/opendsp/opendsp/internal/infrastructure/persistence/postgres/roi"
)

const (
	DefaultClickWindow     = 7 * 24 * time.Hour
	DefaultImpressionWindow = 24 * time.Hour
)

type AttributionTracker struct {
	data             *data.Data
	clickWindow      time.Duration
	impressionWindow time.Duration
}

func NewAttributionTracker(d *data.Data) *AttributionTracker {
	return &AttributionTracker{
		data:             d,
		clickWindow:      DefaultClickWindow,
		impressionWindow: DefaultImpressionWindow,
	}
}

func NewAttributionTrackerWithWindows(d *data.Data, clickWindow, impressionWindow time.Duration) *AttributionTracker {
	return &AttributionTracker{
		data:             d,
		clickWindow:      clickWindow,
		impressionWindow: impressionWindow,
	}
}

func GenerateClickID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *AttributionTracker) HandlePostback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("is_test") == "1" {
		writeJSON(w, 200, map[string]string{"status": "ok", "test": "true"})
		return
	}
	clickID := q.Get("click_id")
	if clickID == "" {
		writeJSON(w, 400, map[string]string{"error": "missing click_id"})
		return
	}

	eventType := q.Get("event_type")
	if eventType == "" {
		eventType = "install"
	}

	revenue, _ := strconv.ParseFloat(q.Get("revenue"), 64)
	if revenue == 0 {
		if v := q.Get("value"); v != "" {
			revenue, _ = strconv.ParseFloat(v, 64)
		}
	}
	price, _ := strconv.ParseFloat(q.Get("price"), 64)
	adgroupID, _ := strconv.ParseInt(q.Get("adgroup_id"), 10, 64)
	creativeID, _ := strconv.ParseInt(q.Get("creative_id"), 10, 64)
	campaignID, _ := strconv.ParseInt(q.Get("campaign_id"), 10, 64)
	advertiserID, _ := strconv.ParseInt(q.Get("advertiser_id"), 10, 64)
	mediaID, _ := strconv.ParseInt(q.Get("media_id"), 10, 64)
	adPositionID, _ := strconv.ParseInt(q.Get("ad_position_id"), 10, 64)
	deviceID := q.Get("device_id")
	geoCity := q.Get("geo_city")
	extraJSON := q.Get("extra")
	extra := []byte("{}")
	if extraJSON != "" {
		extra = []byte(extraJSON)
	}

	now := time.Now()

	if a.data != nil {
		windowStart := now.Add(-a.clickWindow)
		click, err := a.data.Queries.FindClickIDInWindow(r.Context(), &dbsqlc.FindClickIDInWindowParams{
			ClickID:   &clickID,
			EventTime: windowStart,
		})
		if err != nil {
			log.Printf("attribution: click_id %s not found in window: %v", clickID, err)
		}

		if click != nil && adgroupID == 0 {
			adgroupID = click.AdgroupID
		}
		if click != nil && creativeID == 0 {
			creativeID = click.CreativeID
		}
		if click != nil && campaignID == 0 {
			campaignID = click.CampaignID
		}
		if click != nil && advertiserID == 0 {
			advertiserID = click.AdvertiserID
		}
		if click != nil && mediaID == 0 {
			mediaID = click.MediaID
		}
		if click != nil && adPositionID == 0 {
			adPositionID = click.AdPositionID
		}

		revNumeric := pgtype.Numeric{}
		_ = revNumeric.Scan(fmt.Sprintf("%.4f", revenue))
		priceNumeric := pgtype.Numeric{}
		_ = priceNumeric.Scan(fmt.Sprintf("%.4f", price))

		err = a.data.Queries.InsertConversionEvent(r.Context(), &dbsqlc.InsertConversionEventParams{
			ClickID:      clickID,
			EventType:    eventType,
			AdgroupID:    nullInt64Ptr(adgroupID),
			CreativeID:   nullInt64Ptr(creativeID),
			CampaignID:   nullInt64Ptr(campaignID),
			AdvertiserID: nullInt64Ptr(advertiserID),
			MediaID:      nullInt64Ptr(mediaID),
			AdPositionID: nullInt64Ptr(adPositionID),
			Price:        priceNumeric,
			Revenue:      revNumeric,
			DeviceID:     nullStrPtr(deviceID),
			Ip:           nullStrPtr(r.RemoteAddr),
			Ua:           nullStrPtr(r.Header.Get("User-Agent")),
			GeoCity:      nullStrPtr(geoCity),
			Extra:        extra,
			EventTime:    now,
		})
		if err != nil {
			log.Printf("attribution: insert conversion: %v", err)
			writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("insert conversion: %v", err)})
			return
		}

		if adgroupID > 0 && creativeID > 0 {
			err = a.data.Queries.UpsertReportConversions(r.Context(), &dbsqlc.UpsertReportConversionsParams{
				AdvertiserID: advertiserID,
				CampaignID:   campaignID,
				AdGroupID:    adgroupID,
				CreativeID:   creativeID,
				MediaID:      mediaID,
				AdPositionID: adPositionID,
				Revenue:      revNumeric,
			})
			if err != nil {
				log.Printf("attribution: upsert report: %v", err)
			}
		}

		if adgroupID > 0 && revenue > 0 {
			roiRepo := postgresRoi.NewConversionRepo(a.data.Pool)
			costEstimate := int64(0)
			if click != nil {
				if fv, err := click.Price.Float64Value(); err == nil {
					costEstimate = int64(fv.Float64 * 1_000_000)
				}
			}
			if err := roiRepo.UpsertMetrics(r.Context(),
				advertiserID, campaignID, adgroupID,
				now.Truncate(24*time.Hour),
				costEstimate, int64(revenue*1_000_000), 1,
			); err != nil {
				log.Printf("attribution: upsert roi metrics: %v", err)
			}
		}
	}

	writeJSON(w, 200, map[string]interface{}{
		"status":       "ok",
		"click_id":     clickID,
		"event_type":   eventType,
		"attributed":   a.data != nil && adgroupID > 0,
	})
}

func (a *AttributionTracker) HandlePostbackBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return
	}

	if r.URL.Query().Get("is_test") == "1" {
		writeJSON(w, 200, map[string]string{"status": "ok", "test": "true"})
		return
	}

	type batchEvent struct {
		ClickID   string  `json:"click_id"`
		EventType string  `json:"event_type"`
		Revenue   float64 `json:"revenue"`
		Price     float64 `json:"price"`
		DeviceID  string  `json:"device_id"`
		GeoCity   string  `json:"geo_city"`
		Extra     string  `json:"extra"`
	}

	var events []batchEvent
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		writeJSON(w, 400, map[string]string{"error": fmt.Sprintf("invalid JSON: %v", err)})
		return
	}

	if len(events) == 0 {
		writeJSON(w, 400, map[string]string{"error": "empty batch"})
		return
	}

	now := time.Now()
	successCount := 0

	for _, ev := range events {
		if ev.ClickID == "" {
			continue
		}
		if ev.EventType == "" {
			ev.EventType = "install"
		}

		if a.data != nil {
			revNumeric := pgtype.Numeric{}
			_ = revNumeric.Scan(fmt.Sprintf("%.4f", ev.Revenue))
			priceNumeric := pgtype.Numeric{}
			_ = priceNumeric.Scan(fmt.Sprintf("%.4f", ev.Price))
			extra := []byte("{}")
			if ev.Extra != "" {
				extra = []byte(ev.Extra)
			}

			err := a.data.Queries.InsertConversionEvent(r.Context(), &dbsqlc.InsertConversionEventParams{
				ClickID:   ev.ClickID,
				EventType: ev.EventType,
				Revenue:   revNumeric,
				Price:     priceNumeric,
				DeviceID:  nullStrPtr(ev.DeviceID),
				GeoCity:   nullStrPtr(ev.GeoCity),
				Extra:     extra,
				EventTime: now,
			})
			if err != nil {
				log.Printf("attribution: batch insert conversion: %v", err)
				continue
			}

			if ev.Revenue > 0 {
				roiRepo := postgresRoi.NewConversionRepo(a.data.Pool)
				roiRepo.UpsertMetrics(r.Context(),
					0, 0, 0,
					now.Truncate(24*time.Hour),
					0, int64(ev.Revenue*1_000_000), 1,
				)
			}
		}
		successCount++
	}

	writeJSON(w, 200, map[string]interface{}{
		"status":  "ok",
		"total":   len(events),
		"success": successCount,
	})
}

func (a *AttributionTracker) HandleMMPPostback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("is_test") == "1" {
		writeJSON(w, 200, map[string]string{"status": "ok", "test": "true"})
		return
	}
	platform := q.Get("platform")
	switch strings.ToLower(platform) {
	case "adjust":
		a.handleAdjustPostback(w, r)
	case "appsflyer":
		a.handleAppsFlyerPostback(w, r)
	default:
		writeJSON(w, 400, map[string]string{"error": "unsupported MMP platform, use ?platform=adjust or ?platform=appsflyer"})
	}
}

func (a *AttributionTracker) handleAdjustPostback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clickID := q.Get("click_id")
	if clickID == "" {
		clickID = q.Get("gps_adid")
	}
	if clickID == "" {
		clickID = q.Get("idfa")
	}
	if clickID == "" {
		writeJSON(w, 400, map[string]string{"error": "missing click_id or device identifier"})
		return
	}

	eventType := q.Get("event_name")
	if eventType == "" {
		eventType = q.Get("activity_kind")
	}
	if eventType == "" {
		eventType = "install"
	}

	revenueStr := q.Get("revenue")
	currency := q.Get("currency")
	if revenueStr == "" {
		revenueStr = q.Get("revenue_usd")
	}
	revenue, _ := strconv.ParseFloat(revenueStr, 64)

	adgroupID, _ := strconv.ParseInt(q.Get("adgroup_id"), 10, 64)
	creativeID, _ := strconv.ParseInt(q.Get("creative_id"), 10, 64)
	campaignID, _ := strconv.ParseInt(q.Get("campaign_id"), 10, 64)
	advertiserID, _ := strconv.ParseInt(q.Get("advertiser_id"), 10, 64)

	extra := map[string]interface{}{
		"mmp":        "adjust",
		"currency":   currency,
		"app_id":     q.Get("app_id"),
		"app_name":   q.Get("app_name"),
		"event_token": q.Get("event_token"),
		"environment": q.Get("environment"),
		"os_name":    q.Get("os_name"),
		"os_version": q.Get("os_version"),
		"device_type": q.Get("device_type"),
		"country":    q.Get("country"),
		"language":   q.Get("language"),
		"sdk_version": q.Get("sdk_version"),
	}
	extraBytes, _ := json.Marshal(extra)

	now := time.Now()

	if a.data != nil {
		revNumeric := pgtype.Numeric{}
		_ = revNumeric.Scan(fmt.Sprintf("%.4f", revenue))

		err := a.data.Queries.InsertConversionEvent(r.Context(), &dbsqlc.InsertConversionEventParams{
			ClickID:      clickID,
			EventType:    eventType,
			AdgroupID:    nullInt64Ptr(adgroupID),
			CreativeID:   nullInt64Ptr(creativeID),
			CampaignID:   nullInt64Ptr(campaignID),
			AdvertiserID: nullInt64Ptr(advertiserID),
			Revenue:      revNumeric,
			DeviceID:     nullStrPtr(q.Get("gps_adid")),
			Ip:           nullStrPtr(r.RemoteAddr),
			Ua:           nullStrPtr(r.Header.Get("User-Agent")),
			GeoCity:      nullStrPtr(q.Get("country")),
			Extra:        extraBytes,
			EventTime:    now,
		})
		if err != nil {
			log.Printf("attribution: adjust postback insert: %v", err)
			writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("insert: %v", err)})
			return
		}
	}

	writeJSON(w, 200, map[string]interface{}{
		"status":     "ok",
		"platform":   "adjust",
		"click_id":   clickID,
		"event_type": eventType,
	})
}

func (a *AttributionTracker) handleAppsFlyerPostback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clickID := q.Get("click_id")
	if clickID == "" {
		clickID = q.Get("advertising_id")
	}
	if clickID == "" {
		clickID = q.Get("idfa")
	}
	if clickID == "" {
		writeJSON(w, 400, map[string]string{"error": "missing click_id or device identifier"})
		return
	}

	eventType := q.Get("event_name")
	if eventType == "" {
		eventType = q.Get("event_type")
	}
	if eventType == "" {
		eventType = "install"
	}

	revenueStr := q.Get("event_revenue")
	if revenueStr == "" {
		revenueStr = q.Get("revenue")
	}
	revenue, _ := strconv.ParseFloat(revenueStr, 64)

	adgroupID, _ := strconv.ParseInt(q.Get("adgroup_id"), 10, 64)
	creativeID, _ := strconv.ParseInt(q.Get("creative_id"), 10, 64)
	campaignID, _ := strconv.ParseInt(q.Get("campaign_id"), 10, 64)
	advertiserID, _ := strconv.ParseInt(q.Get("advertiser_id"), 10, 64)

	extra := map[string]interface{}{
		"mmp":          "appsflyer",
		"app_id":       q.Get("app_id"),
		"app_name":     q.Get("app_name"),
		"bundle_id":    q.Get("bundle_id"),
		"af_channel":   q.Get("af_channel"),
		"media_source": q.Get("media_source"),
		"campaign":     q.Get("campaign"),
		"af_adset":     q.Get("af_adset"),
		"af_ad":        q.Get("af_ad"),
		"platform":     q.Get("platform"),
		"os_version":   q.Get("os_version"),
		"country_code": q.Get("country_code"),
		"language":     q.Get("language"),
		"http_referrer": q.Get("http_referrer"),
		"install_time": q.Get("install_time"),
		"event_time":   q.Get("event_time"),
	}
	extraBytes, _ := json.Marshal(extra)

	now := time.Now()

	if a.data != nil {
		revNumeric := pgtype.Numeric{}
		_ = revNumeric.Scan(fmt.Sprintf("%.4f", revenue))

		err := a.data.Queries.InsertConversionEvent(r.Context(), &dbsqlc.InsertConversionEventParams{
			ClickID:      clickID,
			EventType:    eventType,
			AdgroupID:    nullInt64Ptr(adgroupID),
			CreativeID:   nullInt64Ptr(creativeID),
			CampaignID:   nullInt64Ptr(campaignID),
			AdvertiserID: nullInt64Ptr(advertiserID),
			Revenue:      revNumeric,
			DeviceID:     nullStrPtr(q.Get("advertising_id")),
			Ip:           nullStrPtr(r.RemoteAddr),
			Ua:           nullStrPtr(r.Header.Get("User-Agent")),
			GeoCity:      nullStrPtr(q.Get("country_code")),
			Extra:        extraBytes,
			EventTime:    now,
		})
		if err != nil {
			log.Printf("attribution: appsflyer postback insert: %v", err)
			writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("insert: %v", err)})
			return
		}
	}

	writeJSON(w, 200, map[string]interface{}{
		"status":     "ok",
		"platform":   "appsflyer",
		"click_id":   clickID,
		"event_type": eventType,
	})
}

func nullInt64Ptr(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

func nullStrPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

package admanager

import (
	"net/http"
	"strconv"
	"time"

	domainroi "github.com/opendsp/opendsp/internal/domain/roi"
	postgresroi "github.com/opendsp/opendsp/internal/infrastructure/persistence/postgres/roi"
)

// ROIHandler serves ROI query API endpoints.
type ROIHandler struct {
	repo *postgresroi.ConversionRepo
}

// NewROIHandler creates a ROI API handler.
func NewROIHandler(repo *postgresroi.ConversionRepo) *ROIHandler {
	return &ROIHandler{repo: repo}
}

type roiSummaryResponse struct {
	TotalCost        float64 `json:"total_cost"`
	TotalRevenue     float64 `json:"total_revenue"`
	TotalConversions int64   `json:"total_conversions"`
	OverallROAS      float64 `json:"overall_roas"`
}

type roiMetricsRow struct {
	AdvertiserID  int64   `json:"advertiser_id"`
	CampaignID    *int64  `json:"campaign_id"`
	AdgroupID     *int64  `json:"adgroup_id"`
	Date          string  `json:"date"`
	CostMicros    int64   `json:"cost_micros"`
	RevenueMicros int64   `json:"revenue_micros"`
	Conversions   int64   `json:"conversions"`
	ROAS          float64 `json:"roas"`
}

// ServeHTTP routes ROI API requests.
func (h *ROIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/api/roi/summary":
		if r.Method == "GET" {
			h.handleSummary(w, r)
		} else {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	case "/api/roi/by-campaign":
		if r.Method == "GET" {
			h.handleByCampaign(w, r)
		} else {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func (h *ROIHandler) handleSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	advertiserID, _ := strconv.ParseInt(q.Get("advertiser_id"), 10, 64)
	if advertiserID == 0 {
		advertiserID = 1
	}

	startDate := q.Get("start_date")
	endDate := q.Get("end_date")
	if startDate == "" {
		startDate = time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	end = end.Add(24 * time.Hour)

	result, err := h.repo.GetCostAndRevenue(r.Context(), domainroi.CostRevenueParams{
		AdvertiserID: advertiserID,
		Start:        start,
		End:          end,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	cost := float64(result.CostMicros) / 1_000_000
	revenue := float64(result.RevenueMicros) / 1_000_000
	roas := float64(0)
	if result.CostMicros > 0 {
		roas = float64(result.RevenueMicros) / float64(result.CostMicros)
	}

	writeJSON(w, http.StatusOK, roiSummaryResponse{
		TotalCost:        cost,
		TotalRevenue:     revenue,
		TotalConversions: result.Conversions,
		OverallROAS:      roas,
	})
}

func (h *ROIHandler) handleByCampaign(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	advertiserID, _ := strconv.ParseInt(q.Get("advertiser_id"), 10, 64)
	if advertiserID == 0 {
		advertiserID = 1
	}

	startDate := q.Get("start_date")
	endDate := q.Get("end_date")
	if startDate == "" {
		startDate = time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	end = end.Add(24 * time.Hour)

	result, err := h.repo.GetCostAndRevenue(r.Context(), domainroi.CostRevenueParams{
		AdvertiserID: advertiserID,
		Start:        start,
		End:          end,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	roas := float64(0)
	if result.CostMicros > 0 {
		roas = float64(result.RevenueMicros) / float64(result.CostMicros)
	}

	today := time.Now().Format("2006-01-02")
	items := []roiMetricsRow{{
		AdvertiserID:  advertiserID,
		Date:          today,
		CostMicros:    result.CostMicros,
		RevenueMicros: result.RevenueMicros,
		Conversions:   result.Conversions,
		ROAS:          roas,
	}}

	writeJSON(w, http.StatusOK, items)
}

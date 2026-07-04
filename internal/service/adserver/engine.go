package adserver

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	appbidding "github.com/opendsp/opendsp/internal/application/bidding"
	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/freq"
	"github.com/opendsp/opendsp/internal/index"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	bidRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dsp_bid_requests_total",
	}, []string{"media", "position_type"})

	bidLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dsp_bid_latency_seconds",
		Buckets: []float64{.001, .002, .005, .01, .02, .05, .1},
	}, []string{"media"})

	bidWins = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dsp_bid_wins_total",
	}, []string{"media"})

	bidNoFill = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dsp_bid_nofill_total",
	}, []string{"media", "reason"})
)

type Engine struct {
	index    *index.InvertedIndex
	freqCtrl *freq.Controller
	pipeline *appbidding.Pipeline

	// Legacy exclusion cache, preserved for backward compat during migration
	exclusion *ExclusionCache
}

const exclusionCacheTTL = 5 * time.Second

type ExclusionCache struct {
	mu             sync.RWMutex
	budgetExcluded map[uint32]string
	budgetDate     string
	budgetAt       time.Time
	freqExcluded   map[string]map[uint32]string
	freqDate       string
	freqAt         map[string]time.Time
}

func newExclusionCache() *ExclusionCache {
	return &ExclusionCache{
		budgetExcluded: make(map[uint32]string),
		freqExcluded:   make(map[string]map[uint32]string),
		freqAt:         make(map[string]time.Time),
	}
}

func (c *ExclusionCache) getExcluded(userID string) map[uint32]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[uint32]string, len(c.budgetExcluded)+len(c.freqExcluded[userID]))
	for k, v := range c.budgetExcluded {
		result[k] = v
	}
	for k, v := range c.freqExcluded[userID] {
		result[k] = v
	}
	return result
}

func NewEngine(idx *index.InvertedIndex, freqCtrl *freq.Controller, pipeline *appbidding.Pipeline) *Engine {
	return &Engine{
		index:     idx,
		freqCtrl:  freqCtrl,
		pipeline:  pipeline,
		exclusion: newExclusionCache(),
	}
}

func (e *Engine) RefreshExclusions(ctx context.Context, userID string) {
	now := time.Now()
	date := now.Format("20060102")

	e.exclusion.mu.RLock()
	budgetFresh := e.exclusion.budgetDate == date && now.Sub(e.exclusion.budgetAt) < exclusionCacheTTL
	freqFresh := e.exclusion.freqDate == date && now.Sub(e.exclusion.freqAt[userID]) < exclusionCacheTTL
	e.exclusion.mu.RUnlock()

	if budgetFresh && freqFresh {
		return
	}

	if !budgetFresh {
		excluded, err := e.freqCtrl.GetBudgetExcludedAdGroups(ctx, date)
		if err == nil {
			e.exclusion.mu.Lock()
			e.exclusion.budgetExcluded = excluded
			e.exclusion.budgetDate = date
			e.exclusion.budgetAt = now
			e.exclusion.mu.Unlock()
		}
	}

	if !freqFresh {
		excluded, err := e.freqCtrl.GetFreqExcludedAdGroups(ctx, date, userID)
		if err == nil {
			e.exclusion.mu.Lock()
			if e.exclusion.freqDate != date {
				e.exclusion.freqExcluded = make(map[string]map[uint32]string)
				e.exclusion.freqDate = date
				e.exclusion.freqAt = make(map[string]time.Time)
			}
			e.exclusion.freqExcluded[userID] = excluded
			e.exclusion.freqAt[userID] = now
			e.exclusion.mu.Unlock()
		}
	}
}

type BidResult struct {
	AdGroupID      uint32
	Creative       *index.CreativeInfo
	Price          float64
	LandingURL     string
	DeeplinkURL    string
	ImpTrackers    []string
	ClickTrackers  []string
	Width          int32
	Height         int32
	Duration       int32
	AssetURL       string
	ClickID        string
	PlatformCrID   string
	ClickType      string
	ClickThroughURL string
	TrackingEvents map[string][]string
	DeeplinkApp    string
	IconURL        string
	ReservationID  string // token bucket reservation ID for impression tracking
	CampaignID     int64  // campaign ID for enriched tracker params
	AdvertiserID   int64  // advertiser ID for enriched tracker params
}

type BidParams struct {
	MediaID      string
	PositionType int32
	GeoCity      string
	OS           string
	DeviceType   string
	ContentID    string
	Category     string
	Width        int32
	Height       int32
	MinDuration  int32
	MaxDuration  int32
	UserID       string
	AudienceID   int64
	IsTest       bool
}

func (e *Engine) Bid(ctx context.Context, params BidParams) *BidResult {
	start := time.Now()
	defer func() {
		bidLatency.WithLabelValues(params.MediaID).Observe(time.Since(start).Seconds())
	}()

	bidRequests.WithLabelValues(params.MediaID, fmt.Sprintf("%d", params.PositionType)).Inc()

	// Build domain BidRequest
	req := &bidding.BidRequest{
		MediaID:      params.MediaID,
		PositionType: params.PositionType,
		GeoCity:      params.GeoCity,
		OS:           params.OS,
		DeviceType:   params.DeviceType,
		ContentID:    params.ContentID,
		Category:     params.Category,
		Width:        params.Width,
		Height:       params.Height,
		MinDuration:  params.MinDuration,
		MaxDuration:  params.MaxDuration,
		UserID:       params.UserID,
		IsTest:       params.IsTest,
	}

	// Pre-match pipeline (anti-fraud)
	if e.pipeline != nil && !e.pipeline.RunPreMatch(ctx, req) {
		bidNoFill.WithLabelValues(params.MediaID, "antifraud").Inc()
		return nil
	}

	// Legacy: refresh exclusion cache (budget/freq checks)
	e.RefreshExclusions(ctx, params.UserID)

	// Index matching (existing logic)
	matchReq := &index.MatchRequest{
		MediaID:      params.MediaID,
		PositionType: params.PositionType,
		GeoCity:      params.GeoCity,
		OS:           params.OS,
		DeviceType:   params.DeviceType,
		ContentID:    params.ContentID,
		Category:     params.Category,
		AudienceID:   params.AudienceID,
		Exclusion:    e.exclusion.getExcluded(params.UserID),
	}

	adGroupIDs := e.index.Match(matchReq)
	if len(adGroupIDs) == 0 {
		bidNoFill.WithLabelValues(params.MediaID, "no_match").Inc()
		return nil
	}

	// Convert matched ad group IDs to domain Candidates
	candidates := make([]*bidding.Candidate, 0, len(adGroupIDs))
	for _, agID := range adGroupIDs {
		ag := e.index.GetAdGroup(agID)
		if ag == nil {
			continue
		}
		creatives := e.index.GetCreatives(agID)
		for _, cr := range creatives {
			if cr.AuditStatus != 1 {
				continue
			}
			candidates = append(candidates, bidding.NewCandidateWithAdvertiser(agID, cr.ID, ag.BidPrice, ag.AdvertiserID, ag.CampaignID))
		}
	}

	// Post-match pipeline (RTA → feature → scoring → pricing → pacing → budget guard)
	if e.pipeline != nil {
		var aborted bool
		candidates, aborted = e.pipeline.RunPostMatch(ctx, req, candidates)
		if aborted || len(candidates) == 0 {
			bidNoFill.WithLabelValues(params.MediaID, "pipeline_filtered").Inc()
			return nil
		}
	}

	// Select best candidate and find its creative
	for _, c := range candidates {
		ag := e.index.GetAdGroup(c.AdGroupID)
		if ag == nil {
			continue
		}

		creatives := e.index.GetCreatives(c.AdGroupID)
		creative := findCreativeByID(creatives, c.CreativeID)
		if creative == nil {
			continue
		}
		if !creativeMatchesFormat(creative, params.Width, params.Height, params.MinDuration, params.MaxDuration) {
			continue
		}

		bidWins.WithLabelValues(params.MediaID).Inc()

		clickID := GenerateClickID()

		platformCrID := creative.PlatformCrIDs[params.MediaID]
		if platformCrID == "" {
			platformCrID = strconv.FormatInt(creative.ID, 10)
		}

		clickType := "0"
		if creative.DeeplinkURL != "" {
			clickType = "14"
		}

		return &BidResult{
			AdGroupID:       c.AdGroupID,
			Creative:        creative,
			Price:           ag.BidPrice,
			LandingURL:      creative.LandingURL,
			DeeplinkURL:     creative.DeeplinkURL,
			ImpTrackers:     []string{creative.ImpTracker},
			ClickTrackers:   []string{creative.ClickTracker},
			Width:           creative.AssetWidth,
			Height:          creative.AssetHeight,
			Duration:        creative.AssetDuration,
			AssetURL:        creative.AssetURL,
			ClickID:         clickID,
			PlatformCrID:    platformCrID,
			ClickType:       clickType,
			ClickThroughURL: creative.LandingURL,
			DeeplinkApp:     "",
			CampaignID:      ag.CampaignID,
			AdvertiserID:    ag.AdvertiserID,
		}
	}

	bidNoFill.WithLabelValues(params.MediaID, "no_creative_match").Inc()
	return nil
}

func selectCreative(creatives []index.CreativeInfo, width, height, minDuration, maxDuration int32) *index.CreativeInfo {
	for i := range creatives {
		c := &creatives[i]
		if c.AuditStatus != 1 {
			continue
		}
		if width > 0 && height > 0 {
			if c.AssetWidth != width || c.AssetHeight != height {
				continue
			}
		}
		if maxDuration > 0 && c.AssetDuration > maxDuration {
			continue
		}
		if minDuration > 0 && c.AssetDuration < minDuration {
			continue
		}
		return c
	}
	return nil
}

// findCreativeByID looks up a creative by its ID from the creative list.
func findCreativeByID(creatives []index.CreativeInfo, id int64) *index.CreativeInfo {
	for i := range creatives {
		if creatives[i].ID == id {
			return &creatives[i]
		}
	}
	return nil
}

// creativeMatchesFormat checks if a creative satisfies the format requirements.
func creativeMatchesFormat(c *index.CreativeInfo, width, height, minDuration, maxDuration int32) bool {
	if c.AuditStatus != 1 {
		return false
	}
	if width > 0 && height > 0 {
		if c.AssetWidth != width || c.AssetHeight != height {
			return false
		}
	}
	if maxDuration > 0 && c.AssetDuration > maxDuration {
		return false
	}
	if minDuration > 0 && c.AssetDuration < minDuration {
		return false
	}
	return true
}

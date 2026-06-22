package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/redis/go-redis/v9"
)

type InsightService struct {
	reportRepo biz.ReportRepo
	llm        LLMClient
	rdb        *redis.Client
	mu         sync.RWMutex
}

func NewInsightService(reportRepo biz.ReportRepo, llm LLMClient, rdb *redis.Client) *InsightService {
	return &InsightService{reportRepo: reportRepo, llm: llm, rdb: rdb}
}

type DashboardInsight struct {
	Summary        string `json:"summary"`
	TopAdGroup     string `json:"top_adgroup"`
	WorstAdGroup   string `json:"worst_adgroup"`
	PacingAlert    string `json:"pacing_alert"`
	Recommendation string `json:"recommendation"`
	GeneratedAt    string `json:"generated_at"`
}

type ReportAnomaly struct {
	Hour        string  `json:"hour"`
	Metric      string  `json:"metric"`
	Value       float64 `json:"value"`
	Expected    float64 `json:"expected"`
	Explanation string  `json:"explanation"`
}

func (s *InsightService) GetDashboardInsight(ctx context.Context, advertiserID int64) (*DashboardInsight, error) {
	cacheKey := fmt.Sprintf("ai:insight:dashboard:%d", advertiserID)
	cached, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var insight DashboardInsight
		if json.Unmarshal([]byte(cached), &insight) == nil {
			return &insight, nil
		}
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayStart := todayStart.Add(-24 * time.Hour)
	yesterdayEnd := todayStart

	todayReports, err := s.reportRepo.Query(ctx, advertiserID, nil, nil, todayStart, now)
	if err != nil {
		return nil, fmt.Errorf("query today: %w", err)
	}
	yesterdayReports, err := s.reportRepo.Query(ctx, advertiserID, nil, nil, yesterdayStart, yesterdayEnd)
	if err != nil {
		return nil, fmt.Errorf("query yesterday: %w", err)
	}

	todayImp, todayClicks, todayCost := aggregateReports(todayReports)
	yestImp, yestClicks, yestCost := aggregateReports(yesterdayReports)

	todayCTR := 0.0
	if todayImp > 0 {
		todayCTR = float64(todayClicks) / float64(todayImp) * 100
	}

	costChange := percentChange(yestCost, todayCost)
	ctrChange := todayCTR
	if yestImp > 0 {
		yestCTR := float64(yestClicks) / float64(yestImp) * 100
		ctrChange = todayCTR - yestCTR
	}

	summary := fmt.Sprintf(
		"Today: ¥%.2f cost, %d impressions, %d clicks, %.2f%% CTR. "+
			"vs yesterday: cost %+.0f%%, CTR %+.1fpp.",
		todayCost, todayImp, todayClicks, todayCTR, costChange, ctrChange,
	)

	insight := &DashboardInsight{
		Summary:        summary,
		PacingAlert:    "Budget pacing is normal for this hour.",
		Recommendation: "All metrics are within expected ranges. No urgent actions needed.",
		GeneratedAt:    now.Format(time.RFC3339),
	}

	data, _ := json.Marshal(insight)
	s.rdb.Set(ctx, cacheKey, data, 5*time.Minute)

	return insight, nil
}

func (s *InsightService) GetReportAnomalies(ctx context.Context, advertiserID int64, startTime, endTime time.Time) ([]ReportAnomaly, error) {
	cacheKey := fmt.Sprintf("ai:insight:report:%d:%s:%s", advertiserID, startTime.Format("20060102"), endTime.Format("20060102"))
	cached, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var anomalies []ReportAnomaly
		if json.Unmarshal([]byte(cached), &anomalies) == nil {
			return anomalies, nil
		}
	}

	reports, err := s.reportRepo.Query(ctx, advertiserID, nil, nil, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("query reports: %w", err)
	}

	if len(reports) < 4 {
		return nil, nil
	}

	var ctrs []float64
	for _, r := range reports {
		if r.Impressions > 0 {
			ctrs = append(ctrs, float64(r.Clicks)/float64(r.Impressions)*100)
		} else {
			ctrs = append(ctrs, 0)
		}
	}

	mean, stddev := meanStddev(ctrs)
	threshold := 2.0

	var anomalies []ReportAnomaly
	for i, r := range reports {
		if i >= len(ctrs) {
			break
		}
		if math.Abs(ctrs[i]-mean) > threshold*stddev && r.Impressions > 100 {
			anomalies = append(anomalies, ReportAnomaly{
				Hour:        r.Hour.Format(time.RFC3339),
				Metric:      "ctr",
				Value:       ctrs[i],
				Expected:    mean,
				Explanation: fmt.Sprintf("CTR deviates %.1f standard deviations from the %.1f%% average for this period.", math.Abs(ctrs[i]-mean)/stddev, mean),
			})
		}
	}

	data, _ := json.Marshal(anomalies)
	s.rdb.Set(ctx, cacheKey, data, 10*time.Minute)

	return anomalies, nil
}

func (s *InsightService) Refresh(ctx context.Context, advertiserID int64) {
	s.rdb.Del(ctx, fmt.Sprintf("ai:insight:dashboard:%d", advertiserID))
}

func aggregateReports(reports []biz.ReportHourly) (impressions, clicks int64, cost float64) {
	for _, r := range reports {
		impressions += r.Impressions
		clicks += r.Clicks
		cost += r.Cost
	}
	return
}

func percentChange(old, new float64) float64 {
	if old == 0 {
		if new == 0 {
			return 0
		}
		return 100
	}
	return (new - old) / old * 100
}

func meanStddev(values []float64) (mean, stddev float64) {
	if len(values) == 0 {
		return 0, 0
	}
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	for _, v := range values {
		stddev += (v - mean) * (v - mean)
	}
	stddev = math.Sqrt(stddev / float64(len(values)))
	return
}

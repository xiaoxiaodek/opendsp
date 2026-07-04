package freq

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
)

var (
	freqCheckLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dsp_freq_check_latency_seconds",
		Help:    "Frequency/capacity check latency in seconds.",
		Buckets: []float64{.0001, .0005, .001, .002, .005, .01, .02, .05, .1, .5},
	}, []string{"result"})
)

//go:embed lua/impression_check.lua
var impressionCheckScript string

type Controller struct {
	rdb    *redis.Client
	script *redis.Script
}

func NewController(rdb *redis.Client) *Controller {
	return &Controller{
		rdb:    rdb,
		script: redis.NewScript(impressionCheckScript),
	}
}

type CheckParams struct {
	AdGroupID           int64
	CampaignID          int64
	AdvertiserID        int64
	UserID              string
	AdGroupFreqCap      int32
	CampaignFreqCap     int32
	AdGroupDailyBudget  float64
	CampaignDailyBudget float64
	CampaignTotalBudget float64
	BidPrice            float64
}

type CheckResult struct {
	OK     bool
	Reason string
}

func (c *Controller) Check(ctx context.Context, p CheckParams) (res *CheckResult, err error) {
	start := time.Now()
	defer func() {
		label := "ok"
		if err != nil {
			label = "error"
		} else if res != nil && !res.OK {
			label = "rejected"
		} else if p.BidPrice <= 0 {
			label = "skip"
		}
		freqCheckLatency.WithLabelValues(label).Observe(time.Since(start).Seconds())
	}()
	date := time.Now().Format("20060102")

	priceCents := int64(p.BidPrice * 100)
	agDailyBudget := int64(0)
	if p.AdGroupDailyBudget > 0 {
		agDailyBudget = int64(p.AdGroupDailyBudget * 100)
	}
	cDailyBudget := int64(0)
	if p.CampaignDailyBudget > 0 {
		cDailyBudget = int64(p.CampaignDailyBudget * 100)
	}
	cTotalBudget := int64(0)
	if p.CampaignTotalBudget > 0 {
		cTotalBudget = int64(p.CampaignTotalBudget * 100)
	}

	keys := []string{
		fmt.Sprintf("freq:adgroup:%d:%s:%s", p.AdGroupID, date, p.UserID),
		fmt.Sprintf("freq:campaign:%d:%s:%s", p.CampaignID, date, p.UserID),
		fmt.Sprintf("budget:daily:%d:%s", p.AdGroupID, date),
		fmt.Sprintf("budget:daily:%d:%s", p.CampaignID, date),
		fmt.Sprintf("budget:total:%d", p.CampaignID),
		fmt.Sprintf("budget:excluded:%s", date),
		fmt.Sprintf("freq:excluded:%s:%s", date, p.UserID),
		fmt.Sprintf("balance:%d", p.AdvertiserID),
	}

	args := []interface{}{
		strconv.Itoa(int(p.AdGroupFreqCap)),
		strconv.Itoa(int(p.CampaignFreqCap)),
		strconv.FormatInt(agDailyBudget, 10),
		strconv.FormatInt(cDailyBudget, 10),
		strconv.FormatInt(cTotalBudget, 10),
		strconv.FormatInt(priceCents, 10),
		strconv.FormatInt(p.AdGroupID, 10),
		strconv.FormatInt(p.CampaignID, 10),
	}

	result, err := c.script.Run(ctx, c.rdb, keys, args...).Slice()
	if err != nil {
		return nil, fmt.Errorf("freq check script: %w", err)
	}

	ok, _ := result[0].(int64)
	reason, _ := result[1].(string)

	return &CheckResult{
		OK:     ok == 1,
		Reason: reason,
	}, nil
}

func (c *Controller) GetExcludedAdGroups(ctx context.Context, date, userID string) (map[uint32]string, error) {
	result := make(map[uint32]string)
	if c.rdb == nil {
		return result, nil
	}

	budgetKey := fmt.Sprintf("budget:excluded:%s", date)
	members, err := c.rdb.SMembers(ctx, budgetKey).Result()
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		id, _ := strconv.ParseUint(m, 10, 32)
		result[uint32(id)] = "budget_exhausted"
	}

	freqKey := fmt.Sprintf("freq:excluded:%s:%s", date, userID)
	members, err = c.rdb.SMembers(ctx, freqKey).Result()
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		id, _ := strconv.ParseUint(m, 10, 32)
		result[uint32(id)] = "freq_cap"
	}

	return result, nil
}

func (c *Controller) GetBudgetExcludedAdGroups(ctx context.Context, date string) (map[uint32]string, error) {
	result := make(map[uint32]string)
	if c.rdb == nil {
		return result, nil
	}

	budgetKey := fmt.Sprintf("budget:excluded:%s", date)
	members, err := c.rdb.SMembers(ctx, budgetKey).Result()
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		id, _ := strconv.ParseUint(m, 10, 32)
		result[uint32(id)] = "budget_exhausted"
	}
	return result, nil
}

func (c *Controller) GetFreqExcludedAdGroups(ctx context.Context, date, userID string) (map[uint32]string, error) {
	result := make(map[uint32]string)
	if c.rdb == nil {
		return result, nil
	}

	freqKey := fmt.Sprintf("freq:excluded:%s:%s", date, userID)
	members, err := c.rdb.SMembers(ctx, freqKey).Result()
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		id, _ := strconv.ParseUint(m, 10, 32)
		result[uint32(id)] = "freq_cap"
	}
	return result, nil
}

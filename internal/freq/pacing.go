package freq

import (
	"math"
	"strconv"
	"sync"
	"time"
)

// PacingMode defines the pacing strategy.
type PacingMode string

const (
	PacingModeEven PacingMode = "even"
	PacingModeASAP PacingMode = "asap"
)

// campaignPacing tracks pacing state for a single campaign.
type campaignPacing struct {
	campaignID         int64
	totalBudgetCents   int64
	dailyBudgetCents   int64 // 0 means unlimited
	startTime          time.Time
	endTime            time.Time
	mode               PacingMode
	maxTokensPerBucket int64

	currentRate    float64 // current refill rate (cents/second)
	idealRate      float64 // ideal refill rate
	lastAdjustAt   time.Time
	totalConsumed  int64
}

// PacingController manages pacing for all registered campaigns.
type PacingController struct {
	mu           sync.RWMutex
	campaigns    map[string]*campaignPacing // keyed by campaignID string
	adjustInterval time.Duration
}

// NewPacingController creates a new PacingController.
func NewPacingController(defaultMode string) *PacingController {
	return &PacingController{
		campaigns:      make(map[string]*campaignPacing),
		adjustInterval: 30 * time.Second,
	}
}

// RegisterCampaign registers a campaign for pacing control.
func (pc *PacingController) RegisterCampaign(campaignID int64, totalBudget float64, dailyBudget *float64,
	startTime, endTime time.Time, pacingMode string, maxTokensPerBucket int64) {

	totalCents := int64(totalBudget * 100)
	var dailyCents int64
	if dailyBudget != nil && *dailyBudget > 0 {
		dailyCents = int64(*dailyBudget * 100)
	}

	mode := PacingModeEven
	if pacingMode == "asap" {
		mode = PacingModeASAP
	}

	idealRate := pc.calculateIdealRate(totalCents, dailyCents, startTime, endTime, mode)

	cp := &campaignPacing{
		campaignID:         campaignID,
		totalBudgetCents:   totalCents,
		dailyBudgetCents:   dailyCents,
		startTime:          startTime,
		endTime:            endTime,
		mode:               mode,
		maxTokensPerBucket: maxTokensPerBucket,
		currentRate:        idealRate,
		idealRate:          idealRate,
		lastAdjustAt:       time.Now(),
	}

	pc.mu.Lock()
	pc.campaigns[strconv.FormatInt(campaignID, 10)] = cp
	pc.mu.Unlock()
}

// UnregisterCampaign removes a campaign from pacing control.
func (pc *PacingController) UnregisterCampaign(campaignID int64) {
	pc.mu.Lock()
	delete(pc.campaigns, strconv.FormatInt(campaignID, 10))
	pc.mu.Unlock()
}

// GetRefillRate returns the current refill rate for a campaign (cents/second total).
func (pc *PacingController) GetRefillRate(campaignID int64) float64 {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	cp, ok := pc.campaigns[strconv.FormatInt(campaignID, 10)]
	if !ok {
		return 0
	}
	return cp.currentRate
}

// AdjustRate recalculates the refill rate based on actual consumption vs expected.
// actualConsumed is the total confirmed tokens (cents) consumed so far.
func (pc *PacingController) AdjustRate(campaignID int64, actualConsumed int64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	cp, ok := pc.campaigns[strconv.FormatInt(campaignID, 10)]
	if !ok {
		return
	}

	cp.totalConsumed = actualConsumed
	now := time.Now()

	// Only adjust at intervals
	if now.Sub(cp.lastAdjustAt) < pc.adjustInterval {
		return
	}
	cp.lastAdjustAt = now

	// Calculate expected consumption
	elapsed := now.Sub(cp.startTime).Seconds()
	if elapsed <= 0 {
		return
	}
	totalDuration := cp.endTime.Sub(cp.startTime).Seconds()
	if totalDuration <= 0 {
		return
	}

	expectedConsumed := cp.idealRate * elapsed

	// Apply daily budget cap
	if cp.dailyBudgetCents > 0 {
		dayElapsed := now.Sub(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())).Seconds()
		dayFraction := dayElapsed / 86400.0
		dailyExpected := float64(cp.dailyBudgetCents) * dayFraction
		if dailyExpected < expectedConsumed {
			expectedConsumed = dailyExpected
		}
	}

	// Dynamic adjustment based on deviation
	if expectedConsumed > 0 {
		deviation := (float64(actualConsumed) - expectedConsumed) / expectedConsumed

		if deviation > 0.1 {
			// Over-consuming, slow down
			cp.currentRate *= 0.8
		} else if deviation < -0.3 {
			// Under-consuming, speed up
			cp.currentRate *= 1.2
		}
	}

	// Clamp rate to reasonable bounds
	if cp.currentRate < 1.0/float64(pc.adjustInterval.Seconds()) {
		cp.currentRate = 1.0 / float64(pc.adjustInterval.Seconds())
	}
	maxRate := float64(cp.totalBudgetCents) / float64(pc.adjustInterval.Seconds())
	if cp.currentRate > maxRate {
		cp.currentRate = maxRate
	}

	// Apply daily budget cap to rate
	if cp.dailyBudgetCents > 0 {
		remainingToday := cp.dailyBudgetCents - actualConsumed
		if remainingToday <= 0 {
			cp.currentRate = 0
		} else {
			secondsLeftToday := 86400.0 - float64(now.Hour()*3600+now.Minute()*60+now.Second())
			if secondsLeftToday <= 0 {
				secondsLeftToday = 1
			}
			dailyRate := float64(remainingToday) / secondsLeftToday
			if cp.currentRate > dailyRate {
				cp.currentRate = dailyRate
			}
		}
	}

	// Check total budget
	remainingTotal := cp.totalBudgetCents - actualConsumed
	if remainingTotal <= 0 {
		cp.currentRate = 0
	} else {
		secondsLeft := cp.endTime.Sub(now).Seconds()
		if secondsLeft <= 0 {
			secondsLeft = 1
		}
		totalRate := float64(remainingTotal) / secondsLeft
		if cp.currentRate > totalRate {
			cp.currentRate = totalRate
		}
	}
}

// calculateIdealRate computes the ideal refill rate for a campaign.
func (pc *PacingController) calculateIdealRate(totalCents, dailyCents int64, startTime, endTime time.Time, mode PacingMode) float64 {
	duration := endTime.Sub(startTime).Seconds()
	if duration <= 0 {
		return 0
	}

	baseRate := float64(totalCents) / duration

	switch mode {
	case PacingModeASAP:
		// ASAP: faster at the beginning, slower at the end
		// Phase 1 (first 1/3): 2x rate
		// Phase 2 (middle 1/3): 1x rate
		// Phase 3 (final 1/3): 0.3x rate
		now := time.Now()
		elapsed := now.Sub(startTime).Seconds()
		fraction := elapsed / duration

		switch {
		case fraction < 1.0/3.0:
			return baseRate * 2.0
		case fraction < 2.0/3.0:
			return baseRate * 1.0
		default:
			return baseRate * 0.3
		}

	default: // PacingModeEven
		// Apply daily budget cap if set
		if dailyCents > 0 {
			dailyRate := float64(dailyCents) / 86400.0
			if dailyRate < baseRate {
				return dailyRate
			}
		}
		return baseRate
	}
}

// CalculateRefillAmount computes the tokens to refill per bucket per interval.
func (pc *PacingController) CalculateRefillAmount(campaignID int64, bucketCount int, intervalSeconds float64) int64 {
	rate := pc.GetRefillRate(campaignID)
	if rate <= 0 {
		return 0
	}

	amount := int64(math.Ceil(rate * intervalSeconds / float64(bucketCount)))
	if amount <= 0 {
		amount = 1
	}
	return amount
}

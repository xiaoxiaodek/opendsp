package bidding

import (
	"context"
	"log"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/domain/budget"
)

// BudgetGuardStage performs pre-freeze financial checks using two-phase commit.
// This is the final POST-MATCH stage before bid response.
// Pre-freezes budget for the top candidate; if the bid wins, Confirm is called;
// if the bid loses, Release is called to refund the pre-frozen amount.
type BudgetGuardStage struct {
	service budget.BudgetGuardService
}

// NewBudgetGuardStage creates a budget guard pipeline stage.
func NewBudgetGuardStage(service budget.BudgetGuardService) *BudgetGuardStage {
	return &BudgetGuardStage{service: service}
}

// Name returns the stage name.
func (s *BudgetGuardStage) Name() string { return "budget_guard" }

// Process pre-freezes budget for each candidate using two-phase commit.
// Candidates that fail pre-freeze are removed.
// Callers should call Confirm() on the winner and Release() on losers
// after the ADX responds.
func (s *BudgetGuardStage) Process(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
	if req.IsTest {
		return candidates, nil
	}

	if s.service == nil {
		return candidates, nil
	}

	var kept []*bidding.Candidate
	for _, c := range candidates {
		bidAmount := budget.NewMoneyFromFloat64(c.BidPrice)
		token, err := s.service.PreFreeze(ctx, c.AdvertiserID, int64(c.AdGroupID), bidAmount)
		if err != nil {
			// Insufficient balance or prefreeze failed → drop candidate
			log.Printf("budget_guard: prefreeze failed for adgroup %d advertiser %d: %v", c.AdGroupID, c.AdvertiserID, err)
			continue
		}
		// Store token for later confirm/release
		c.FinalScore = float64(token.Amount.AmountMicros)
		c.BudgetToken = token
		kept = append(kept, c)
	}

	return kept, nil
}

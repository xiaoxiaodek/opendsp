package budget

import "context"

// PacingService decides whether to bid based on budget consumption rate.
type PacingService interface {
	ShouldBid(ctx context.Context, adGroupID int64, dailyBudget, spentToday float64) (bool, error)
}

// BudgetGuardService handles pre-freeze and confirmation of budget.
type BudgetGuardService interface {
	PreFreeze(ctx context.Context, advertiserID, adGroupID int64, amount Money) (*PreFreezeToken, error)
	Confirm(ctx context.Context, token *PreFreezeToken) error
	Release(ctx context.Context, token *PreFreezeToken) error
}

package bidding

import (
	"context"
	"sort"

	"github.com/opendsp/opendsp/internal/domain/bidding"
)

// PricingStage calculates eCPM for each candidate and re-ranks them.
type PricingStage struct {
	strategy   bidding.PricingStrategy
	targetROAS float64
	multiplier bidding.MultiplierStore
}

func NewPricingStage(strategy bidding.PricingStrategy, targetROAS float64, multiplier bidding.MultiplierStore) *PricingStage {
	return &PricingStage{strategy: strategy, targetROAS: targetROAS, multiplier: multiplier}
}

func (s *PricingStage) Name() string { return "pricing" }

func (s *PricingStage) Process(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
	for _, c := range candidates {
		bidPriceMicros := int64(c.BidPrice * 1_000_000)

		var ecpm bidding.ECPM
		switch s.strategy {
		case bidding.PricingStrategyOXBI:
			oxbiMultiplier := 1.0
			if s.multiplier != nil {
				if val, err := s.multiplier.Get(ctx, 0, 0); err == nil && val > 0 {
					oxbiMultiplier = val
				}
			}
			if oxbiMultiplier == 1.0 && c.PredCVR > 0 && s.targetROAS > 0 {
				oxbiMultiplier = c.PredCVR / (1.0 / s.targetROAS)
			}
			if oxbiMultiplier > 2.0 {
				oxbiMultiplier = 2.0
			}
			if oxbiMultiplier < 0.5 {
				oxbiMultiplier = 0.5
			}
			adjustedBid := int64(float64(bidPriceMicros) * oxbiMultiplier)
			ecpm = bidding.NewECPM(adjustedBid, c.PredCTR, c.PredCVR)
		default:
			ecpm = bidding.NewECPM(bidPriceMicros, c.PredCTR, c.PredCVR)
		}

		c.ECPM = ecpm.ValueMicros
		c.FinalScore = float64(ecpm.ValueMicros)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].FinalScore > candidates[j].FinalScore
	})

	return candidates, nil
}

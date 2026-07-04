package bidding

import (
	"context"
	"sort"
	"time"

	domainBidding "github.com/opendsp/opendsp/internal/domain/bidding"
)

// CoarseRankingStage performs lightweight LR scoring to reduce candidate count.
type CoarseRankingStage struct {
	maxCandidates int
	model         domainBidding.LRModel
}

// NewCoarseRankingStage creates a coarse ranking stage with LR model.
func NewCoarseRankingStage(maxCandidates int, model domainBidding.LRModel) *CoarseRankingStage {
	if maxCandidates <= 0 {
		maxCandidates = 200
	}
	return &CoarseRankingStage{maxCandidates: maxCandidates, model: model}
}

// Name returns the stage name.
func (s *CoarseRankingStage) Name() string { return "coarse_ranking" }

// Process scores candidates with LR and keeps top N.
func (s *CoarseRankingStage) Process(ctx context.Context, req *domainBidding.BidRequest, candidates []*domainBidding.Candidate) ([]*domainBidding.Candidate, error) {
	if len(candidates) <= s.maxCandidates {
		return candidates, nil
	}

	hour := float64(time.Now().Hour())
	for _, c := range candidates {
		feats := map[string]float64{
			"bid_price":    c.BidPrice / 100.0,
			"hour":         hour,
			"os_match":     1.0,
			"device_match": 1.0,
			"media_ctr":    0.01,
			"geo_ctr":      0.01,
		}
		c.FinalScore = s.model.Score(feats)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].FinalScore > candidates[j].FinalScore
	})

	return candidates[:s.maxCandidates], nil
}

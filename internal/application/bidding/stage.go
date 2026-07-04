package bidding

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/bidding"
)

// Stage is a single step in the bid pipeline.
// Each stage receives candidates and returns filtered/enriched candidates.
// Returning empty candidates signals the pipeline should abort (no bid).
type Stage interface {
	Name() string
	Process(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error)
}

// StageFunc adapts a function to the Stage interface.
type StageFunc struct {
	name    string
	process func(context.Context, *bidding.BidRequest, []*bidding.Candidate) ([]*bidding.Candidate, error)
}

func (s *StageFunc) Name() string { return s.name }

func (s *StageFunc) Process(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
	return s.process(ctx, req, candidates)
}

// NewStage creates a Stage from a named function.
func NewStage(name string, fn func(context.Context, *bidding.BidRequest, []*bidding.Candidate) ([]*bidding.Candidate, error)) Stage {
	return &StageFunc{name: name, process: fn}
}

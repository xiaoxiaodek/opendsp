package bidding

import (
	"context"
	"testing"

	"github.com/opendsp/opendsp/internal/domain/bidding"
)

func TestPipelineRunPreMatch_AllPass(t *testing.T) {
	p := NewPipeline(
		[]Stage{
			NewStage("test_stage", func(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
				// Pre-match: return empty non-nil slice to signal "pass"
				return []*bidding.Candidate{}, nil
			}),
		},
		nil,
	)

	req := &bidding.BidRequest{RequestID: "test-1", MediaID: "iqiyi"}
	if !p.RunPreMatch(context.Background(), req) {
		t.Error("expected pre-match to pass")
	}
}

func TestPipelineRunPreMatch_AbortOnNil(t *testing.T) {
	p := NewPipeline(
		[]Stage{
			NewStage("blocker", func(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
				return nil, nil // abort
			}),
		},
		nil,
	)

	req := &bidding.BidRequest{RequestID: "test-2", MediaID: "iqiyi"}
	if p.RunPreMatch(context.Background(), req) {
		t.Error("expected pre-match to abort when stage returns nil candidates")
	}
}

func TestPipelineRunPreMatch_ErrorSkips(t *testing.T) {
	p := NewPipeline(
		[]Stage{
			NewStage("error_stage", func(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
				return nil, context.DeadlineExceeded
			}),
		},
		nil,
	)

	req := &bidding.BidRequest{RequestID: "test-3", MediaID: "iqiyi"}
	if !p.RunPreMatch(context.Background(), req) {
		t.Error("expected pre-match to continue on stage error")
	}
}

func TestPipelineRunPostMatch_FiltersCandidates(t *testing.T) {
	p := NewPipeline(nil, []Stage{
		NewStage("filter_half", func(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
			return candidates[:len(candidates)/2], nil
		}),
	})

	candidates := []*bidding.Candidate{
		bidding.NewCandidate(1, 10, 1.5),
		bidding.NewCandidate(2, 20, 2.0),
		bidding.NewCandidate(3, 30, 3.0),
		bidding.NewCandidate(4, 40, 1.0),
	}

	req := &bidding.BidRequest{RequestID: "test-4"}
	result, aborted := p.RunPostMatch(context.Background(), req, candidates)

	if aborted {
		t.Error("expected post-match to not abort")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(result))
	}
}

func TestPipelineRunPostMatch_AbortOnEmpty(t *testing.T) {
	p := NewPipeline(nil, []Stage{
		NewStage("drop_all", func(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
			return nil, nil
		}),
	})

	candidates := []*bidding.Candidate{
		bidding.NewCandidate(1, 10, 1.0),
	}

	req := &bidding.BidRequest{RequestID: "test-5"}
	result, aborted := p.RunPostMatch(context.Background(), req, candidates)

	if !aborted {
		t.Error("expected post-match to abort when all candidates dropped")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 candidates, got %d", len(result))
	}
}

func TestPipelineRunPostMatch_EmptyInput(t *testing.T) {
	p := NewPipeline(nil, []Stage{
		NewStage("noop", func(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
			return candidates, nil
		}),
	})

	req := &bidding.BidRequest{RequestID: "test-6"}
	result, aborted := p.RunPostMatch(context.Background(), req, nil)

	if aborted {
		t.Error("expected no abort on nil input (short-circuits)")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 candidates, got %d", len(result))
	}
}

func TestPipeline_EmptyStages(t *testing.T) {
	p := NewPipeline(nil, nil)

	req := &bidding.BidRequest{RequestID: "test-7"}
	if !p.RunPreMatch(context.Background(), req) {
		t.Error("empty pre-match should pass")
	}

	candidates := []*bidding.Candidate{bidding.NewCandidate(1, 10, 1.0)}
	result, aborted := p.RunPostMatch(context.Background(), req, candidates)
	if aborted {
		t.Error("empty post-match should not abort")
	}
	if len(result) != 1 {
		t.Errorf("empty post-match should pass through, got %d", len(result))
	}
}

func TestStageFunc(t *testing.T) {
	called := false
	s := NewStage("test", func(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
		called = true
		return candidates, nil
	})

	if s.Name() != "test" {
		t.Errorf("expected name 'test', got '%s'", s.Name())
	}

	req := &bidding.BidRequest{}
	_, err := s.Process(context.Background(), req, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected stage function to be called")
	}
}

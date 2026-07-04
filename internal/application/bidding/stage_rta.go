package bidding

import (
	"context"
	"sync"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/domain/rta"
)

type RTAStage struct {
	service rta.RTAService
	timeout int64
}

func NewRTAStage(service rta.RTAService, timeoutMs int64) *RTAStage {
	if timeoutMs <= 0 {
		timeoutMs = 15
	}
	return &RTAStage{service: service, timeout: timeoutMs}
}

func (s *RTAStage) Name() string { return "rta" }

func (s *RTAStage) Process(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
	if s.service == nil || len(candidates) == 0 {
		return candidates, nil
	}

	advIDs := make(map[int64]bool)
	for _, c := range candidates {
		advIDs[int64(c.AdGroupID)] = true
	}

	results := make(map[int64]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for advID := range advIDs {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			allowed, _ := s.service.Query(ctx, id, req.DeviceID, req.MediaID, req.RequestID)
			mu.Lock()
			results[id] = allowed
			mu.Unlock()
		}(advID)
	}
	wg.Wait()

	var filtered []*bidding.Candidate
	for _, c := range candidates {
		if allowed, ok := results[int64(c.AdGroupID)]; !ok || allowed {
			filtered = append(filtered, c)
		}
	}
	return filtered, nil
}

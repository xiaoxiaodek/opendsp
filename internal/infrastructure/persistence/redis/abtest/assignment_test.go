package abtest

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/redis/go-redis/v9"
)

func TestAssignmentService_AssignsVariant(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	rdb.SAdd(context.Background(), "abtest:experiments", "1")
	rdb.HSet(context.Background(), "abtest:exp:1", "config",
		`{"name":"BidStrategyTest","variants":[{"name":"control","percentage":50},{"name":"aggressive","percentage":50}]}`)

	svc := NewAssignmentService(rdb)
	req := &bidding.BidRequest{RequestID: "req-abc"}

	assignment, err := svc.Assign(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assignment == nil {
		t.Fatal("expected assignment, got nil")
	}
	if assignment.ExperimentID != 1 {
		t.Errorf("expected experiment 1, got %d", assignment.ExperimentID)
	}
	if assignment.VariantName != "control" && assignment.VariantName != "aggressive" {
		t.Errorf("expected control or aggressive, got %s", assignment.VariantName)
	}
}

func TestAssignmentService_NoExperiments(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	svc := NewAssignmentService(rdb)
	req := &bidding.BidRequest{RequestID: "req-abc"}

	assignment, err := svc.Assign(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assignment != nil {
		t.Error("expected nil assignment when no experiments")
	}
}

func TestHashRequest_Deterministic(t *testing.T) {
	a := hashRequest("hello")
	b := hashRequest("hello")
	if a != b {
		t.Errorf("hashRequest should be deterministic: %d != %d", a, b)
	}
}

func TestHashRequest_Range(t *testing.T) {
	for i := 0; i < 1000; i++ {
		h := hashRequest(string(rune('a' + i%26)) + string(rune('0'+i%10)))
		if h < 0 || h >= 100 {
			t.Errorf("hashRequest out of range: %d", h)
		}
	}
}

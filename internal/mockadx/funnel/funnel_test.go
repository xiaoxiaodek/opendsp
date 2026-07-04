package funnel

import (
	"testing"
	"time"
)

func TestStore_InsertAndTransition(t *testing.T) {
	s := NewStore(5 * time.Minute)

	s.Insert("bid-001", &BidContext{BidID: "bid-001", RequestID: "req-001"})

	ctx := s.Get("bid-001")
	if ctx == nil {
		t.Fatal("expected bid context to exist")
	}
	if ctx.State != StateBid {
		t.Errorf("expected StateBid, got %v", ctx.State)
	}

	s.Transition("bid-001", StateWin)
	ctx = s.Get("bid-001")
	if ctx.State != StateWin {
		t.Errorf("expected StateWin, got %v", ctx.State)
	}

	s.Transition("bid-001", StateImp)
	s.Transition("bid-001", StateClick)

	snap := s.Snapshot()
	if snap.ClickCount != 1 {
		t.Errorf("expected 1 click, got %d", snap.ClickCount)
	}
	if snap.Total != 1 {
		t.Errorf("expected 1 total, got %d", snap.Total)
	}
}

func TestStore_GetMissing(t *testing.T) {
	s := NewStore(5 * time.Minute)
	ctx := s.Get("nonexistent")
	if ctx != nil {
		t.Error("expected nil for missing bid")
	}
}

func TestStore_TTLExpiry(t *testing.T) {
	s := NewStore(10 * time.Millisecond)
	s.Insert("bid-ttl", &BidContext{BidID: "bid-ttl"})

	ctx := s.Get("bid-ttl")
	if ctx == nil {
		t.Fatal("expected bid context to exist")
	}
	if ctx.Deadline.IsZero() {
		t.Error("expected deadline to be set")
	}

	time.Sleep(50 * time.Millisecond)

	if !time.Now().After(ctx.Deadline) {
		t.Error("expected deadline to have passed")
	}
}
package funnel

import (
	"sync"
	"time"
)

type State int

const (
	StateBid    State = iota
	StateWin
	StateImp
	StateClick
	StateConv
	StateExpired
)

func (s State) String() string {
	switch s {
	case StateBid:
		return "bid"
	case StateWin:
		return "win"
	case StateImp:
		return "imp"
	case StateClick:
		return "click"
	case StateConv:
		return "conv"
	case StateExpired:
		return "expired"
	default:
		return "unknown"
	}
}

type StateTransition struct {
	From      State
	To        State
	Timestamp time.Time
}

type BidContext struct {
	BidID            string
	RequestID        string
	Price            float64
	State            State
	Deadline         time.Time
	CreatedAt        time.Time
	StateTransitions []StateTransition
}

type Store struct {
	m           sync.Map
	ttl         time.Duration
	insertCount int64
	stateCounts [6]int64
}

func NewStore(ttl time.Duration) *Store {
	s := &Store{ttl: ttl}
	go s.cleanupLoop()
	return s
}

func (s *Store) Insert(bidID string, ctx *BidContext) {
	ctx.CreatedAt = time.Now()
	ctx.Deadline = time.Now().Add(s.ttl)
	ctx.State = StateBid
	s.m.Store(bidID, ctx)
	s.insertCount++
	s.stateCounts[StateBid]++
}

func (s *Store) Transition(bidID string, newState State) {
	val, ok := s.m.Load(bidID)
	if !ok {
		return
	}
	ctx := val.(*BidContext)
	trans := StateTransition{
		From:      ctx.State,
		To:        newState,
		Timestamp: time.Now(),
	}
	ctx.State = newState
	ctx.StateTransitions = append(ctx.StateTransitions, trans)
	s.stateCounts[newState]++
}

func (s *Store) Get(bidID string) *BidContext {
	val, ok := s.m.Load(bidID)
	if !ok {
		return nil
	}
	return val.(*BidContext)
}

func (s *Store) Snapshot() StoreSnapshot {
	var snapshot StoreSnapshot
	s.m.Range(func(key, value interface{}) bool {
		ctx := value.(*BidContext)
		snapshot.Total++
		switch ctx.State {
		case StateBid:
			snapshot.BidCount++
		case StateWin:
			snapshot.WinCount++
		case StateImp:
			snapshot.ImpCount++
		case StateClick:
			snapshot.ClickCount++
		case StateConv:
			snapshot.ConvCount++
		case StateExpired:
			snapshot.ExpiredCount++
		}
		return true
	})
	return snapshot
}

type StoreSnapshot struct {
	Total        int64
	BidCount     int64
	WinCount     int64
	ImpCount     int64
	ClickCount   int64
	ConvCount    int64
	ExpiredCount int64
}

func (s *Store) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.m.Range(func(key, value interface{}) bool {
			ctx := value.(*BidContext)
			if now.After(ctx.Deadline) && ctx.State != StateExpired {
				ctx.State = StateExpired
				s.stateCounts[StateExpired]++
			}
			return true
		})
	}
}
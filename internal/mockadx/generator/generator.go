package generator

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"

	"github.com/opendsp/opendsp/internal/mockadx/config"
)

type DeviceSpec struct {
	OS         string
	DeviceType string
	IP         string
	UA         string
	DeviceID   string
	GeoCity    string
	GeoCountry string
	Make       string
	Model      string
}

type ContentSpec struct {
	ContentID string
	Title     string
	Category  string
	Duration  int32
	URL       string
}

type ImpSpec struct {
	ImpID        string
	PositionType int32
	Width        int32
	Height       int32
	MinDuration  int32
	MaxDuration  int32
	BidFloor     float64
}

type BidRequestSpec struct {
	RequestID string
	UserID    string
	Device    DeviceSpec
	Content   ContentSpec
	Imp       ImpSpec
	IsTest    bool
}

type Generator interface {
	Next(ctx context.Context) *BidRequestSpec
}

type hotUserProfile struct {
	rand    *Randomizer
	pool    []string
	nextIdx atomic.Int64
}

func newHotUserProfile(poolSize int, rand *Randomizer) *hotUserProfile {
	pool := make([]string, poolSize)
	for i := 0; i < poolSize; i++ {
		pool[i] = fmt.Sprintf("hot_user_%d", i)
	}
	return &hotUserProfile{rand: rand, pool: pool}
}

func (p *hotUserProfile) Next(ctx context.Context) *BidRequestSpec {
	idx := p.nextIdx.Add(1) % int64(len(p.pool))
	userID := p.pool[idx]
	return &BidRequestSpec{
		RequestID: fmt.Sprintf("hot-%d", p.rand.Int63()),
		UserID:    userID,
		IsTest:    true,
		Device: DeviceSpec{
			OS:         p.rand.OS(),
			DeviceType: p.rand.DeviceType(),
			IP:         fmt.Sprintf("10.%d.%d.%d", p.rand.Intn(256), p.rand.Intn(256), p.rand.Intn(256)),
			UA:         "MockADX/1.0",
			DeviceID:   fmt.Sprintf("dev_%s", userID),
			GeoCity:    p.rand.City(),
		},
		Content: ContentSpec{
			ContentID: p.rand.ContentID(),
			Title:     fmt.Sprintf("Hot Content %d", p.rand.Intn(1000)),
		},
		Imp: ImpSpec{
			ImpID:        fmt.Sprintf("imp-%d", p.rand.Int63()),
			PositionType: p.rand.PositionType(),
			BidFloor:     p.rand.BidFloor(),
		},
	}
}

type longTailProfile struct {
	rand  *Randomizer
	maxID int64
}

func newLongTailProfile(maxID int64, rand *Randomizer) *longTailProfile {
	return &longTailProfile{rand: rand, maxID: maxID}
}

func (p *longTailProfile) Next(ctx context.Context) *BidRequestSpec {
	userID := fmt.Sprintf("lt_user_%d", rand.Int63n(p.maxID))
	return &BidRequestSpec{
		RequestID: fmt.Sprintf("lt-%d", p.rand.Int63()),
		UserID:    userID,
		IsTest:    true,
		Device: DeviceSpec{
			OS:         p.rand.OS(),
			DeviceType: p.rand.DeviceType(),
			IP:         fmt.Sprintf("10.%d.%d.%d", p.rand.Intn(256), p.rand.Intn(256), p.rand.Intn(256)),
			UA:         "MockADX/1.0",
			DeviceID:   fmt.Sprintf("dev_%s", userID),
			GeoCity:    p.rand.City(),
		},
		Content: ContentSpec{
			ContentID: p.rand.ContentID(),
			Title:     fmt.Sprintf("Long Tail Content %d", p.rand.Intn(10000)),
		},
		Imp: ImpSpec{
			ImpID:        fmt.Sprintf("imp-%d", p.rand.Int63()),
			PositionType: p.rand.PositionType(),
			BidFloor:     p.rand.BidFloor(),
		},
	}
}

type ScenarioMux struct {
	profiles []Generator
	weights  []float64
	rng      *rand.Rand
}

func NewScenarioMux(cfg *config.ScenarioConfig, rnd *Randomizer) Generator {
	hot := newHotUserProfile(cfg.HotUserPool, rnd)
	long := newLongTailProfile(cfg.LongTailIDs, rnd)

	switch cfg.Profile {
	case "hot-user":
		return hot
	case "long-tail":
		return long
	case "peak":
		return long
	case "mixed":
		return &ScenarioMux{
			profiles: []Generator{hot, long},
			weights:  []float64{cfg.MixedWeights["hot-user"], cfg.MixedWeights["long-tail"]},
			rng:      rand.New(rand.NewSource(rnd.Int63())),
		}
	default:
		return long
	}
}

func (m *ScenarioMux) Next(ctx context.Context) *BidRequestSpec {
	r := m.rng.Float64()
	cum := 0.0
	for i, w := range m.weights {
		cum += w
		if r <= cum {
			return m.profiles[i].Next(ctx)
		}
	}
	return m.profiles[0].Next(ctx)
}
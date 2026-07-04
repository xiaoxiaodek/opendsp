package generator

import (
	"fmt"
	"math/rand"
	"sync"
)

type Randomizer struct {
	mu            sync.Mutex
	rng           *rand.Rand
	cities        []string
	oses          []string
	deviceTypes   []string
	contentIDs    []string
	positionTypes []int32
	posWeights    []float64
}

func NewRandomizer(seed int64) *Randomizer {
	r := &Randomizer{
		rng: rand.New(rand.NewSource(seed)),
		cities: []string{
			"861100", "863100", "864400", "864401", "865100",
			"865300", "865700", "865900", "867310", "867320",
			"867330", "867340", "867350", "867360", "867370",
			"867410", "867420", "867430", "867440", "867450",
			"867460", "867510", "867520", "867530", "867540",
			"867710", "867720", "867730", "867740", "867750",
			"867760", "867810", "867820", "867830", "867840",
			"867850", "867860", "867910", "867920", "867930",
			"867940", "867950", "867960", "868110", "868120",
			"868130", "868140", "868150", "868160", "868210",
		},
		oses:          []string{"android", "android", "android", "android", "ios"},
		deviceTypes:   []string{"mobile", "mobile", "mobile", "mobile", "mobile", "mobile", "mobile", "mobile", "mobile", "mobile", "mobile", "mobile", "mobile", "mobile", "mobile", "mobile", "mobile", "tablet", "tablet", "tablet"},
		contentIDs:    makeContentIDs(500),
		positionTypes: []int32{1, 1, 1, 1, 2, 2, 2, 4, 5, 3},
		posWeights:    []float64{0.4, 0.4, 0.4, 0.4, 0.3, 0.3, 0.3, 0.1, 0.1, 0.1},
	}
	return r
}

func (r *Randomizer) City() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cities[r.rng.Intn(len(r.cities))]
}

func (r *Randomizer) OS() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.oses[r.rng.Intn(len(r.oses))]
}

func (r *Randomizer) DeviceType() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.deviceTypes[r.rng.Intn(len(r.deviceTypes))]
}

func (r *Randomizer) ContentID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.contentIDs[r.rng.Intn(len(r.contentIDs))]
}

func (r *Randomizer) PositionType() int32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.positionTypes[r.rng.Intn(len(r.positionTypes))]
}

func (r *Randomizer) BidFloor() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return max(100, r.rng.NormFloat64()*150+500)
}

func (r *Randomizer) UserID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fmt.Sprintf("user_%d", r.rng.Int63())
}

func (r *Randomizer) Int63() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.Int63()
}

func (r *Randomizer) Intn(n int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.Intn(n)
}

func makeContentIDs(n int) []string {
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = fmt.Sprintf("content_%d", i+1)
	}
	return ids
}
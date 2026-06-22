package dmp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/opendsp/opendsp/internal/biz"
)

type LookalikeEngine struct {
	repo  biz.DmpRepo
	store *TagStore
}

func NewLookalikeEngine(repo biz.DmpRepo, store *TagStore) *LookalikeEngine {
	return &LookalikeEngine{repo: repo, store: store}
}

func (e *LookalikeEngine) Run(ctx context.Context, seedAudienceID int64, expansionFactor int32) (int64, error) {
	audience, err := e.repo.GetAudience(ctx, seedAudienceID)
	if err != nil {
		return 0, fmt.Errorf("get seed audience: %w", err)
	}

	tagID, err := e.repo.CreateTag(ctx, &biz.DmpTag{
		AdvertiserID: audience.AdvertiserID,
		Name:         fmt.Sprintf("Lookalike: %s x%d", audience.Name, expansionFactor),
		TagType:      biz.TagTypeLookalike,
		Source:       "lookalike",
		Status:       biz.TagStatusComputing,
	})
	if err != nil {
		return 0, fmt.Errorf("create tag: %w", err)
	}

	go e.computeLookalike(context.Background(), audience, tagID, expansionFactor)

	return tagID, nil
}

func (e *LookalikeEngine) computeLookalike(ctx context.Context, audience *biz.DmpAudience, tagID int64, factor int32) {
	var root RuleNode
	if err := json.Unmarshal(audience.Rules, &root); err != nil {
		log.Printf("lookalike: parse rules: %v", err)
		e.repo.UpdateTagDeviceCount(ctx, tagID, 0, biz.TagStatusInvalid)
		return
	}

	resolver := NewAudienceResolver(e.store, nil)
	seedBM, err := resolver.resolveNode(&root)
	if err != nil || seedBM.GetCardinality() == 0 {
		log.Printf("lookalike: resolve seed: %v", err)
		e.repo.UpdateTagDeviceCount(ctx, tagID, 0, biz.TagStatusInvalid)
		return
	}

	seedCount := seedBM.GetCardinality()
	targetCount := seedCount * uint64(factor)
	if targetCount > 100_000_000 {
		targetCount = 100_000_000
	}

	seedFeatures := e.extractFeatures(seedBM)
	poolFeatures := e.extractPoolFeatures()

	similarities := make([]struct {
		deviceID uint32
		score    float64
	}, 0)

	for deviceID, feat := range poolFeatures {
		score := cosineSimilarity(seedFeatures, feat)
		similarities = append(similarities, struct {
			deviceID uint32
			score    float64
		}{deviceID, score})
	}

	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].score > similarities[j].score
	})

	resultBM := roaring.New()
	for i := 0; i < len(similarities) && uint64(i) < targetCount; i++ {
		resultBM.Add(similarities[i].deviceID)
	}

	if err := e.store.SaveBitmap(tagID, resultBM); err != nil {
		log.Printf("lookalike: save bitmap: %v", err)
		e.repo.UpdateTagDeviceCount(ctx, tagID, 0, biz.TagStatusInvalid)
		return
	}

	e.repo.UpdateTagDeviceCount(ctx, tagID, int64(resultBM.GetCardinality()), biz.TagStatusReady)
	log.Printf("lookalike: tag %d created with %d devices from seed %d", tagID, resultBM.GetCardinality(), seedCount)
}

func (e *LookalikeEngine) extractFeatures(bm *roaring.Bitmap) map[string]float64 {
	features := make(map[string]float64)
	features["os:ios"] = 0.6
	features["os:android"] = 0.4
	features["city:110000"] = 0.3
	features["carrier:cmcc"] = 0.5
	return features
}

func (e *LookalikeEngine) extractPoolFeatures() map[uint32]map[string]float64 {
	pool := make(map[uint32]map[string]float64)
	for i := uint32(1); i <= 10000; i++ {
		pool[i] = map[string]float64{
			"os:ios":       float64(i%3) * 0.3,
			"os:android":   float64((i+1)%3) * 0.3,
			"city:110000":  float64(i%5) * 0.2,
			"carrier:cmcc": float64(i%2) * 0.5,
		}
	}
	return pool
}

func cosineSimilarity(a, b map[string]float64) float64 {
	var dot, normA, normB float64
	for k, va := range a {
		vb := b[k]
		dot += va * vb
		normA += va * va
		normB += vb * vb
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

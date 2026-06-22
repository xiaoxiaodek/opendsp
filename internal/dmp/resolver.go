package dmp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/redis/go-redis/v9"
)

type RuleNode struct {
	Operator string     `json:"operator,omitempty"`
	TagID    int64      `json:"tag_id,omitempty"`
	Include  []RuleNode `json:"include,omitempty"`
	Exclude  []RuleNode `json:"exclude,omitempty"`
}

type AudienceResolver struct {
	store *TagStore
	rdb   *redis.Client
}

func NewAudienceResolver(store *TagStore, rdb *redis.Client) *AudienceResolver {
	return &AudienceResolver{store: store, rdb: rdb}
}

func (r *AudienceResolver) Resolve(audienceID int64, rules json.RawMessage) (*roaring.Bitmap, error) {
	cacheKey := fmt.Sprintf("dmp:audience:%d", audienceID)
	data, err := r.rdb.Get(context.Background(), cacheKey).Bytes()
	if err == nil {
		bm := roaring.New()
		if err := bm.UnmarshalBinary(data); err == nil {
			return bm, nil
		}
	}

	var root RuleNode
	if err := json.Unmarshal(rules, &root); err != nil {
		return nil, fmt.Errorf("parse rules: %w", err)
	}

	bm, err := r.resolveNode(&root)
	if err != nil {
		return nil, err
	}

	data, _ = bm.ToBytes()
	r.rdb.Set(context.Background(), cacheKey, data, 0)

	return bm, nil
}

func (r *AudienceResolver) resolveNode(node *RuleNode) (*roaring.Bitmap, error) {
	if node.TagID > 0 {
		return r.store.GetBitmap(node.TagID)
	}

	var result *roaring.Bitmap
	for _, child := range node.Include {
		childBM, err := r.resolveNode(&child)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = childBM.Clone()
		} else if node.Operator == "AND" {
			result.And(childBM)
		} else {
			result.Or(childBM)
		}
	}

	for _, child := range node.Exclude {
		childBM, err := r.resolveNode(&child)
		if err != nil {
			return nil, err
		}
		if result != nil {
			result.AndNot(childBM)
		}
	}

	if result == nil {
		return roaring.New(), nil
	}
	return result, nil
}

func (r *AudienceResolver) InvalidateCache(audienceID int64) {
	r.rdb.Del(context.Background(), fmt.Sprintf("dmp:audience:%d", audienceID))
}

package abtest

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"strconv"

	"github.com/opendsp/opendsp/internal/domain/abtest"
	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/redis/go-redis/v9"
)

type AssignmentService struct {
	rdb *redis.Client
}

func NewAssignmentService(rdb *redis.Client) *AssignmentService {
	return &AssignmentService{rdb: rdb}
}

type variantConfig struct {
	Name       string `json:"name"`
	Percentage int32  `json:"percentage"`
}

type experimentConfig struct {
	ID       int64           `json:"id"`
	Name     string          `json:"name"`
	Variants []variantConfig `json:"variants"`
}

func (s *AssignmentService) Assign(ctx context.Context, req *bidding.BidRequest) (*abtest.Assignment, error) {
	expIDs, err := s.rdb.SMembers(ctx, "abtest:experiments").Result()
	if err != nil || len(expIDs) == 0 {
		return nil, nil
	}

	for _, idStr := range expIDs {
		expID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}

		data, err := s.rdb.HGet(ctx, "abtest:exp:"+idStr, "config").Result()
		if err != nil {
			continue
		}

		var cfg experimentConfig
		if err := json.Unmarshal([]byte(data), &cfg); err != nil {
			continue
		}
		cfg.ID = expID

		bucket := hashRequest(req.RequestID + idStr)
		var cumulative int32
		for _, v := range cfg.Variants {
			cumulative += v.Percentage
			if bucket < int(cumulative) {
				return &abtest.Assignment{
					ExperimentID: expID,
					VariantName:  v.Name,
				}, nil
			}
		}
	}

	return nil, nil
}

func hashRequest(input string) int {
	h := fnv.New32a()
	h.Write([]byte(input))
	return int(h.Sum32() % 100)
}

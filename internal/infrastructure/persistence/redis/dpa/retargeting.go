package dpa

import (
	"context"
	"strconv"
	"time"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	domaindpa "github.com/opendsp/opendsp/internal/domain/dpa"
	"github.com/redis/go-redis/v9"
)

type RetargetingService struct {
	rdb *redis.Client
}

func NewRetargetingService(rdb *redis.Client) *RetargetingService {
	return &RetargetingService{rdb: rdb}
}

func (s *RetargetingService) SelectProducts(ctx context.Context, req *bidding.BidRequest, behaviors []domaindpa.UserBehavior) ([]domaindpa.Product, error) {
	now := time.Now().UnixMilli()
	cutoff := now - 24*60*60*1000
	key := "dpa:behavior:" + req.UserID
	if req.UserID == "" {
		return nil, nil
	}

	members, err := s.rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: strconv.FormatInt(cutoff, 10),
		Max: strconv.FormatInt(now, 10),
	}).Result()
	if err != nil || len(members) == 0 {
		return nil, nil
	}

	productIDs := make(map[string]bool)
	for _, m := range members {
		productIDs[m] = true
	}

	var products []domaindpa.Product
	for pid := range productIDs {
		if len(products) >= 5 {
			break
		}
		p, err := s.getProduct(ctx, pid)
		if err != nil || p.ID == "" {
			continue
		}
		products = append(products, p)
	}

	return products, nil
}

func (s *RetargetingService) getProduct(ctx context.Context, productID string) (domaindpa.Product, error) {
	data, err := s.rdb.HGetAll(ctx, "dpa:product:"+productID).Result()
	if err != nil || len(data) == 0 {
		return domaindpa.Product{}, err
	}

	price, _ := strconv.ParseFloat(data["price"], 64)
	return domaindpa.Product{
		ID:         productID,
		Title:      data["title"],
		ImageURL:   data["image_url"],
		LandingURL: data["landing_url"],
		Price:      price,
		Category:   data["category"],
	}, nil
}

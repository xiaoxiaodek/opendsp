package dpa

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/redis/go-redis/v9"
)

func TestRetargetingService_SelectsProducts(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	rdb.HSet(context.Background(), "dpa:product:p1",
		"title", "Shoes", "image_url", "/img/shoes.jpg",
		"landing_url", "/p/shoes", "price", "99.99", "category", "footwear")
	rdb.ZAdd(context.Background(), "dpa:behavior:u1",
		redis.Z{Score: float64(time.Now().UnixMilli()), Member: "p1"})

	svc := NewRetargetingService(rdb)
	req := &bidding.BidRequest{UserID: "u1"}
	products, err := svc.SelectProducts(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	if products[0].Title != "Shoes" {
		t.Errorf("expected Shoes, got %s", products[0].Title)
	}
}

func TestRetargetingService_NoBehaviors(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	svc := NewRetargetingService(rdb)
	req := &bidding.BidRequest{UserID: "u1"}
	products, err := svc.SelectProducts(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("expected 0 products, got %d", len(products))
	}
}

func TestRetargetingService_EmptyUserID(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	svc := NewRetargetingService(rdb)
	req := &bidding.BidRequest{UserID: ""}
	products, err := svc.SelectProducts(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("expected 0 products for empty user ID, got %d", len(products))
	}
}

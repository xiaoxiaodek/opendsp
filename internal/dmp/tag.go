package dmp

import (
	"context"
	"fmt"
	"sync"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/redis/go-redis/v9"
)

type TagStore struct {
	rdb     *redis.Client
	bitmaps map[int64]*roaring.Bitmap
	mu      sync.RWMutex
}

func NewTagStore(rdb *redis.Client) *TagStore {
	return &TagStore{
		rdb:     rdb,
		bitmaps: make(map[int64]*roaring.Bitmap),
	}
}

func (s *TagStore) GetBitmap(tagID int64) (*roaring.Bitmap, error) {
	s.mu.RLock()
	if bm, ok := s.bitmaps[tagID]; ok {
		s.mu.RUnlock()
		return bm, nil
	}
	s.mu.RUnlock()

	key := fmt.Sprintf("dmp:tag:%d", tagID)
	data, err := s.rdb.Get(context.Background(), key).Bytes()
	if err != nil {
		return nil, err
	}

	bm := roaring.New()
	if err := bm.UnmarshalBinary(data); err != nil {
		return nil, fmt.Errorf("unmarshal tag %d: %w", tagID, err)
	}

	s.mu.Lock()
	s.bitmaps[tagID] = bm
	s.mu.Unlock()

	return bm, nil
}

func (s *TagStore) SaveBitmap(tagID int64, bm *roaring.Bitmap) error {
	data, err := bm.ToBytes()
	if err != nil {
		return fmt.Errorf("marshal tag %d: %w", tagID, err)
	}

	key := fmt.Sprintf("dmp:tag:%d", tagID)
	if err := s.rdb.Set(context.Background(), key, data, 0).Err(); err != nil {
		return err
	}

	s.mu.Lock()
	s.bitmaps[tagID] = bm
	s.mu.Unlock()

	s.rdb.Publish(context.Background(), fmt.Sprintf("dmp:tag:updated:%d", tagID), "1")
	return nil
}

func (s *TagStore) AddDevices(tagID int64, deviceIDs []uint32) error {
	bm, err := s.GetBitmap(tagID)
	if err != nil {
		bm = roaring.New()
	}
	for _, id := range deviceIDs {
		bm.Add(id)
	}
	return s.SaveBitmap(tagID, bm)
}

func (s *TagStore) Invalidate(tagID int64) {
	s.mu.Lock()
	delete(s.bitmaps, tagID)
	s.mu.Unlock()
}

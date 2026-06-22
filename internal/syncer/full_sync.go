package syncer

import (
	"context"
	"log"
	"time"

	"github.com/opendsp/opendsp/internal/data"
	"github.com/opendsp/opendsp/internal/index"
)

type FullSyncer struct {
	index *index.InvertedIndex
	data  *data.Data
}

func NewFullSyncer(idx *index.InvertedIndex, d *data.Data) *FullSyncer {
	return &FullSyncer{index: idx, data: d}
}

func (s *FullSyncer) Run(ctx context.Context) {
	if err := s.index.BuildFromDB(ctx, s.data); err != nil {
		log.Printf("initial index build error: %v", err)
	} else {
		log.Printf("initial index built: %d ad groups, version %d", s.index.AdCount(), s.index.Version())
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.index.BuildFromDB(ctx, s.data); err != nil {
				log.Printf("index refresh error: %v", err)
			} else {
				log.Printf("index refreshed: %d ad groups, version %d", s.index.AdCount(), s.index.Version())
			}
		}
	}
}

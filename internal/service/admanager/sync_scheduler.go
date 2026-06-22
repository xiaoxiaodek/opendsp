package admanager

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/opendsp/opendsp/internal/biz"
)

type SyncScheduler struct {
	syncUC    *biz.SyncUseCase
	syncRepo  biz.SyncRepo
	platforms map[string]biz.PlatformSyncer
}

func NewSyncScheduler(syncUC *biz.SyncUseCase, syncRepo biz.SyncRepo) *SyncScheduler {
	s := &SyncScheduler{
		syncUC:    syncUC,
		syncRepo:  syncRepo,
		platforms: make(map[string]biz.PlatformSyncer),
	}

	token := os.Getenv("IQIYI_DSP_TOKEN")
	if token != "" {
		s.platforms["iqiyi"] = NewIqiyiClient(token, "")
	}

	return s
}

func (s *SyncScheduler) RegisterPlatform(name string, syncer biz.PlatformSyncer) {
	s.platforms[name] = syncer
}

func (s *SyncScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Printf("sync scheduler started for platforms: %s", strings.Join(s.platformNames(), ", "))

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for platform, syncer := range s.platforms {
				s.syncUC.RefreshAllPending(ctx, platform, syncer)
			}
		}
	}
}

func (s *SyncScheduler) platformNames() []string {
	names := make([]string, 0, len(s.platforms))
	for name := range s.platforms {
		names = append(names, name)
	}
	return names
}

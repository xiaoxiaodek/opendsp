package biz

import (
	"context"
	"encoding/json"
	"log"
)

type PlatformSyncer interface {
	Name() string
	UploadCreative(ctx context.Context, params *CreativeUploadParams) (string, error)
	QueryCreativeStatus(ctx context.Context, externalID string) (*SyncStatusInfo, error)
	BatchQueryCreativeStatus(ctx context.Context, externalIDs []string) (map[string]*SyncStatusInfo, error)
	UploadAdvertiser(ctx context.Context, params *AdvertiserUploadParams) (string, error)
	QueryAdvertiserStatus(ctx context.Context, externalAdID string) (*SyncStatusInfo, error)
}

type CreativeUploadParams struct {
	CreativeID    int64
	AdvertiserID  string
	Token         string
	CreativeType  int16
	PlatformType  int32
	ClickURL      string
	FileName      string
	FileData      []byte
	AssetMime     string
	AssetWidth    int32
	AssetHeight   int32
	AssetDuration int32
	Title         string
	Description   string
	DeeplinkURL   string
	ImpTracker    string
	ClickTracker  string
}

type AdvertiserUploadParams struct {
	AdvertiserID int64
	Name         string
	Token        string
	IndustryID   int32
	FileData     []byte
	FileName     string
}

type SyncStatusInfo struct {
	Status string
	TVid   string
	AdID   string
	Reason string
	Raw    json.RawMessage
}

type SyncUseCase struct {
	syncRepo SyncRepo
}

func NewSyncUseCase(syncRepo SyncRepo) *SyncUseCase {
	return &SyncUseCase{syncRepo: syncRepo}
}

const (
	SyncStatusUploading     int16 = 1
	SyncStatusPendingReview int16 = 2
	SyncStatusApproved      int16 = 3
	SyncStatusRejected      int16 = 4
)

func (uc *SyncUseCase) SyncCreativeToPlatform(ctx context.Context, params *CreativeUploadParams, syncer PlatformSyncer) error {
	if err := uc.syncRepo.UpsertCreativeSync(ctx, params.CreativeID, syncer.Name(),
		SyncStatusUploading, "", "", "", nil); err != nil {
		return err
	}

	externalID, err := syncer.UploadCreative(ctx, params)
	if err != nil {
		uc.syncRepo.UpsertCreativeSync(ctx, params.CreativeID, syncer.Name(),
			SyncStatusRejected, "", "", err.Error(), nil)
		return err
	}

	rawResp, _ := json.Marshal(map[string]string{"m_id": externalID})
	return uc.syncRepo.UpsertCreativeSync(ctx, params.CreativeID, syncer.Name(),
		SyncStatusPendingReview, externalID, "", "", rawResp)
}

func (uc *SyncUseCase) RefreshCreativeStatus(ctx context.Context, creativeID int64, platform string, syncer PlatformSyncer) (*CreativeSyncStatus, error) {
	sync, err := uc.syncRepo.GetCreativeSync(ctx, creativeID, platform)
	if err != nil {
		return nil, err
	}
	if sync == nil || sync.ExternalID == "" {
		return sync, nil
	}

	status, err := syncer.QueryCreativeStatus(ctx, sync.ExternalID)
	if err != nil {
		return sync, err
	}

	newStatus := mapStatus(status.Status)
	return sync, uc.syncRepo.UpsertCreativeSync(ctx, creativeID, platform,
		newStatus, sync.ExternalID, status.TVid, status.Reason, status.Raw)
}

func (uc *SyncUseCase) RefreshAllPending(ctx context.Context, platform string, syncer PlatformSyncer) {
	rows, err := uc.syncRepo.ListPendingCreativeSync(ctx, platform)
	if err != nil {
		log.Printf("sync: list pending creative sync: %v", err)
		return
	}

	if len(rows) == 0 {
		return
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.ExternalID != nil {
			ids = append(ids, *row.ExternalID)
		}
	}

	if len(ids) == 0 {
		return
	}

	statuses, err := syncer.BatchQueryCreativeStatus(ctx, ids)
	if err != nil {
		log.Printf("sync: batch query creative status: %v", err)
		return
	}

	for _, row := range rows {
		if row.ExternalID == nil {
			continue
		}
		s, ok := statuses[*row.ExternalID]
		if !ok {
			continue
		}
		newStatus := mapStatus(s.Status)
		if err := uc.syncRepo.UpsertCreativeSync(ctx, row.CreativeID, platform,
			newStatus, *row.ExternalID, s.TVid, s.Reason, s.Raw); err != nil {
			log.Printf("sync: update creative %d status: %v", row.CreativeID, err)
		}
	}
}

func mapStatus(platformStatus string) int16 {
	switch platformStatus {
	case "PENDING", "PENDING_REVIEW":
		return SyncStatusPendingReview
	case "APPROVED", "COMPLETE", "ON":
		return SyncStatusApproved
	case "REJECTED", "FAILED":
		return SyncStatusRejected
	default:
		return SyncStatusPendingReview
	}
}

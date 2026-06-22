package admanager

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opendsp/opendsp/internal/biz"
)

type SyncHandler struct {
	syncUC   *biz.SyncUseCase
	syncRepo biz.SyncRepo
	creativeUC *biz.CreativeUseCase
	adGroupUC  *biz.AdGroupUseCase
	campaignUC *biz.CampaignUseCase
}

func NewSyncHandler(syncUC *biz.SyncUseCase, syncRepo biz.SyncRepo, creativeUC *biz.CreativeUseCase, adGroupUC *biz.AdGroupUseCase, campaignUC *biz.CampaignUseCase) *SyncHandler {
	return &SyncHandler{
		syncUC:     syncUC,
		syncRepo:   syncRepo,
		creativeUC: creativeUC,
		adGroupUC:  adGroupUC,
		campaignUC: campaignUC,
	}
}

func (h *SyncHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/sync")

	switch {
	case strings.HasPrefix(path, "/creatives/"):
		h.handleCreativeSync(w, r)
	case strings.HasPrefix(path, "/advertisers/"):
		h.handleAdvertiserSync(w, r)
	default:
		http.Error(w, "unknown sync endpoint", 404)
	}
}

func (h *SyncHandler) handleCreativeSync(w http.ResponseWriter, r *http.Request) {
	// Path: /api/v1/sync/creatives/{id}/{platform} or /api/v1/sync/creatives/{id}/{platform}/refresh
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/sync/creatives/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeJSON(w, 400, errResp("invalid path"))
		return
	}
	creativeID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, 400, errResp("invalid creative_id"))
		return
	}

	platform := parts[1]
	isRefresh := len(parts) > 2 && parts[2] == "refresh"

	token := os.Getenv("IQIYI_DSP_TOKEN")
	syncer := NewIqiyiClient(token, "")

	if isRefresh {
		status, err := h.syncUC.RefreshCreativeStatus(r.Context(), creativeID, platform, syncer)
		if err != nil {
			writeJSON(w, 500, errResp(err.Error()))
			return
		}
		writeJSON(w, 200, map[string]interface{}{
			"creative_id":   creativeID,
			"platform":      platform,
			"status":        status.Status,
			"external_id":   status.ExternalID,
			"external_tvid": status.ExternalTvID,
			"reason":        status.Reason,
		})
		return
	}

	// Sync: download creative, upload to platform
	cr, err := h.creativeUC.Get(r.Context(), creativeID)
	if err != nil || cr == nil {
		writeJSON(w, 404, errResp("creative not found"))
		return
	}

	ag, err := h.adGroupUC.Get(r.Context(), cr.AdGroupID)
	if err != nil || ag == nil {
		writeJSON(w, 404, errResp("ad group not found"))
		return
	}
	campaign, err := h.campaignUC.Get(r.Context(), ag.CampaignID)
	if err != nil || campaign == nil {
		writeJSON(w, 404, errResp("campaign not found"))
		return
	}

	fileData, err := downloadFile(r.Context(), cr.AssetURL)
	if err != nil {
		writeJSON(w, 500, errResp("download creative file: "+err.Error()))
		return
	}

	params := &biz.CreativeUploadParams{
		CreativeID:    cr.ID,
		AdvertiserID:  strconv.FormatInt(campaign.AdvertiserID, 10),
		Token:         token,
		CreativeType:  cr.CreativeType,
		PlatformType:  1,
		ClickURL:      cr.LandingURL,
		FileName:      extractFileName(cr.AssetURL),
		FileData:      fileData,
		AssetMime:     cr.AssetMime,
		AssetWidth:    cr.AssetWidth,
		AssetHeight:   cr.AssetHeight,
		AssetDuration: cr.AssetDuration,
		Title:         cr.Title,
		Description:   cr.Description,
		DeeplinkURL:   cr.DeeplinkURL,
		ImpTracker:    cr.ImpTracker,
		ClickTracker:  cr.ClickTracker,
	}

	if err := h.syncUC.SyncCreativeToPlatform(r.Context(), params, syncer); err != nil {
		writeJSON(w, 500, errResp("sync creative: "+err.Error()))
		return
	}

	status, _ := h.syncRepo.GetCreativeSync(r.Context(), cr.ID, platform)
	writeJSON(w, 200, map[string]interface{}{
		"creative_id":   cr.ID,
		"platform":      platform,
		"status":        status.Status,
		"external_id":   status.ExternalID,
	})
}

func (h *SyncHandler) handleAdvertiserSync(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/sync/advertisers/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeJSON(w, 400, errResp("invalid path"))
		return
	}
	advertiserID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, 400, errResp("invalid advertiser_id"))
		return
	}
	platform := parts[2]

	if err := h.syncRepo.UpsertAdvertiserSync(r.Context(), advertiserID, platform,
		biz.SyncStatusUploading, strconv.FormatInt(advertiserID, 10), "", nil); err != nil {
		writeJSON(w, 500, errResp("sync advertiser: "+err.Error()))
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"advertiser_id": advertiserID,
		"platform":      platform,
		"status":        biz.SyncStatusUploading,
	})
}

func downloadFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func extractFileName(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		if ext := filepath.Ext(name); ext != "" {
			return name
		}
	}
	return "file"
}

func errResp(msg string) map[string]string {
	return map[string]string{"error": msg}
}

package index

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data"
)

func (idx *InvertedIndex) BuildFromDB(ctx context.Context, d *data.Data) error {
	repo := data.NewAdGroupRepo(d)
	groups, err := repo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list active ad groups: %w", err)
	}

	newIdx := New()

	for _, ag := range groups {
		agID := uint32(ag.ID)

		var targeting biz.Targeting
		if err := json.Unmarshal(ag.Targeting, &targeting); err != nil {
			continue
		}

		info := &AdGroupInfo{
			ID:          ag.ID,
			CampaignID:  ag.CampaignID,
			BidPrice:    ag.BidPrice,
			DailyBudget: ag.DailyBudget,
			FreqCap:     ag.FreqCap,
			Targeting:   &targeting,
		}
		newIdx.adGroups[agID] = info
		newIdx.online.Add(agID)

		creativeRepo := data.NewCreativeRepo(d)
		creatives, err := creativeRepo.ListApprovedByAdGroup(ctx, ag.ID)
		if err == nil {
			for _, c := range creatives {
				newIdx.creatives[agID] = append(newIdx.creatives[agID], CreativeInfo{
					ID:            c.ID,
					AssetURL:      c.AssetURL,
					AssetDuration: c.AssetDuration,
					AssetWidth:    c.AssetWidth,
					AssetHeight:   c.AssetHeight,
					AssetMime:     c.AssetMime,
					Title:         c.Title,
					Description:   c.Description,
					LandingURL:    c.LandingURL,
					DeeplinkURL:   c.DeeplinkURL,
					ImpTracker:    c.ImpTracker,
					ClickTracker:  c.ClickTracker,
					AuditStatus:   c.AuditStatus,
				})
			}
		}

		addToInvertedIndex(newIdx, agID, &targeting)
	}

	// Load platform-side creative IDs for bidding
	creativeToAdGroup := make(map[int64]uint32)
	for agID, infos := range newIdx.creatives {
		for _, c := range infos {
			creativeToAdGroup[c.ID] = agID
		}
	}

	syncRepo := data.NewPlatformSyncRepo(d)
	syncRows, err := syncRepo.ListApprovedCreativeSync(ctx)
	if err == nil {
		for _, row := range syncRows {
			agID, ok := creativeToAdGroup[row.CreativeID]
			if !ok {
				continue
			}
			creatives := newIdx.creatives[agID]
			for i := range creatives {
				if creatives[i].ID == row.CreativeID {
					if creatives[i].PlatformCrIDs == nil {
						creatives[i].PlatformCrIDs = make(map[string]string)
					}
					creatives[i].PlatformCrIDs[row.Platform] = row.ExternalTvID
					newIdx.creatives[agID] = creatives
					break
				}
			}
		}
	}

	idx.mu.Lock()
	idx.adGroups = newIdx.adGroups
	idx.creatives = newIdx.creatives
	idx.media = newIdx.media
	idx.position = newIdx.position
	idx.geoCity = newIdx.geoCity
	idx.os = newIdx.os
	idx.deviceType = newIdx.deviceType
	idx.dayOfWeek = newIdx.dayOfWeek
	idx.hourRange = newIdx.hourRange
	idx.contentID = newIdx.contentID
	idx.category = newIdx.category
	idx.audience = newIdx.audience
	idx.online = newIdx.online
	idx.version.Add(1)
	idx.ready.Store(true)
	idx.mu.Unlock()

	indexAdCount.Set(float64(len(newIdx.adGroups)))
	indexVersion.Set(float64(idx.version.Load()))

	return nil
}

func addToInvertedIndex(ix *InvertedIndex, agID uint32, t *biz.Targeting) {
	if t == nil {
		return
	}

	if t.Inventory != nil {
		for _, media := range t.Inventory.Media {
			getOrCreateBitmap(ix.media, media).Add(agID)
		}
		for _, pos := range t.Inventory.AdPosition {
			var posType int32
			switch pos {
			case "pre_roll":
				posType = 1
			case "mid_roll":
				posType = 2
			case "post_roll":
				posType = 3
			case "pause":
				posType = 4
			case "overlay":
				posType = 5
			}
			if posType > 0 {
				getOrCreateInt32Bitmap(ix.position, posType).Add(agID)
			}
		}
		for _, cat := range t.Inventory.ContentCategory {
			getOrCreateBitmap(ix.category, cat).Add(agID)
		}
	}

	if t.Geo != nil {
		for _, city := range t.Geo.City {
			getOrCreateBitmap(ix.geoCity, city).Add(agID)
		}
	}

	if t.Device != nil {
		for _, os := range t.Device.OS {
			getOrCreateBitmap(ix.os, os).Add(agID)
		}
		for _, dt := range t.Device.DeviceType {
			getOrCreateBitmap(ix.deviceType, dt).Add(agID)
		}
	}

	if t.Time != nil {
		for _, dow := range t.Time.DayOfWeek {
			getOrCreateUint8Bitmap(ix.dayOfWeek, uint8(dow)).Add(agID)
		}
		if len(t.Time.HourRange) >= 2 {
			for h := t.Time.HourRange[0]; h <= t.Time.HourRange[1]; h++ {
				getOrCreateUint8Bitmap(ix.hourRange, uint8(h)).Add(agID)
			}
		}
	}

	if t.Audience != nil {
		for _, dmpID := range t.Audience.DmpIDs {
			getOrCreateInt64Bitmap(ix.audience, dmpID).Add(agID)
		}
	}
}

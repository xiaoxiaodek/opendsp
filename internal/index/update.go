package index

import (
	"encoding/json"

	"github.com/opendsp/opendsp/internal/biz"
)

func (idx *InvertedIndex) AddAdGroup(ag *biz.AdGroup, creatives []biz.Creative) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	agID := uint32(ag.ID)

	var targeting biz.Targeting
	json.Unmarshal(ag.Targeting, &targeting)

	info := &AdGroupInfo{
		ID:          ag.ID,
		CampaignID:  ag.CampaignID,
		BidPrice:    ag.BidPrice,
		DailyBudget: ag.DailyBudget,
		FreqCap:     ag.FreqCap,
		Targeting:   &targeting,
	}
	idx.adGroups[agID] = info
	idx.online.Add(agID)

	var ciList []CreativeInfo
	for _, c := range creatives {
		if c.AuditStatus == biz.AuditStatusApproved {
			ciList = append(ciList, CreativeInfo{
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
	idx.creatives[agID] = ciList

	addToInvertedIndex(idx, agID, &targeting)
	idx.version.Add(1)
}

func (idx *InvertedIndex) RemoveAdGroup(agID uint32) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.adGroups, agID)
	delete(idx.creatives, agID)
	idx.online.Remove(agID)
	idx.version.Add(1)
}

func (idx *InvertedIndex) UpdateAdGroupStatus(agID uint32, status int16) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if status == biz.CampaignStatusActive {
		idx.online.Add(agID)
	} else {
		idx.online.Remove(agID)
	}
	idx.version.Add(1)
}

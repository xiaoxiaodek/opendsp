package index

import (
	"time"
)

type MatchRequest struct {
	MediaID      string
	PositionType int32
	GeoCity      string
	OS           string
	DeviceType   string
	ContentID    string
	Category     string
	AudienceID   int64
	Exclusion    map[uint32]string
}

func (idx *InvertedIndex) Match(req *MatchRequest) []uint32 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	result := idx.online.Clone()

	for agID := range req.Exclusion {
		result.Remove(agID)
	}

	if bm, ok := idx.media[req.MediaID]; ok {
		result.And(bm)
	}
	if req.PositionType > 0 {
		if bm, ok := idx.position[req.PositionType]; ok {
			result.And(bm)
		}
	}
	if req.GeoCity != "" {
		if bm, ok := idx.geoCity[req.GeoCity]; ok {
			result.And(bm)
		}
	}
	if req.OS != "" {
		if bm, ok := idx.os[req.OS]; ok {
			result.And(bm)
		}
	}
	if req.DeviceType != "" {
		if bm, ok := idx.deviceType[req.DeviceType]; ok {
			result.And(bm)
		}
	}
	if req.ContentID != "" {
		if bm, ok := idx.contentID[req.MediaID+"_"+req.ContentID]; ok {
			result.And(bm)
		}
	}
	if req.Category != "" {
		if bm, ok := idx.category[req.Category]; ok {
			result.And(bm)
		}
	}

	if req.AudienceID > 0 {
		if bm, ok := idx.audience[req.AudienceID]; ok {
			result.And(bm)
		}
	}

	hour := uint8(time.Now().Hour())
	if bm, ok := idx.hourRange[hour]; ok {
		result.And(bm)
	}

	weekday := uint8(time.Now().Weekday())
	if bm, ok := idx.dayOfWeek[weekday]; ok {
		result.And(bm)
	}

	return result.ToArray()
}

func (idx *InvertedIndex) GetAdGroup(id uint32) *AdGroupInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.adGroups[id]
}

func (idx *InvertedIndex) GetCreatives(id uint32) []CreativeInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.creatives[id]
}

func (idx *InvertedIndex) AdCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.adGroups)
}

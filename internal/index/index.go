package index

import (
	"sync"
	"sync/atomic"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/opendsp/opendsp/internal/biz"
)

type AdGroupInfo struct {
	ID          int64
	CampaignID  int64
	BidPrice    float64
	DailyBudget *float64
	FreqCap     *int32
	Targeting   *biz.Targeting
}

type CreativeInfo struct {
	ID            int64
	AssetURL      string
	AssetDuration int32
	AssetWidth    int32
	AssetHeight   int32
	AssetMime     string
	Title         string
	Description   string
	LandingURL    string
	DeeplinkURL   string
	ImpTracker    string
	ClickTracker  string
	AuditStatus   int16
	PlatformCrIDs map[string]string // "iqiyi" → tv_id, "funshion" → xxx
}

type InvertedIndex struct {
	mu      sync.RWMutex
	version atomic.Int64
	ready   atomic.Bool

	adGroups  map[uint32]*AdGroupInfo
	creatives map[uint32][]CreativeInfo

	media      map[string]*roaring.Bitmap
	position   map[int32]*roaring.Bitmap
	geoCity    map[string]*roaring.Bitmap
	os         map[string]*roaring.Bitmap
	deviceType map[string]*roaring.Bitmap
	dayOfWeek  map[uint8]*roaring.Bitmap
	hourRange  map[uint8]*roaring.Bitmap
	contentID  map[string]*roaring.Bitmap
	category   map[string]*roaring.Bitmap
	audience   map[int64]*roaring.Bitmap

	online *roaring.Bitmap
}

func New() *InvertedIndex {
	online := roaring.New()
	online.SetCopyOnWrite(true)
	return &InvertedIndex{
		adGroups:   make(map[uint32]*AdGroupInfo),
		creatives:  make(map[uint32][]CreativeInfo),
		media:      make(map[string]*roaring.Bitmap),
		position:   make(map[int32]*roaring.Bitmap),
		geoCity:    make(map[string]*roaring.Bitmap),
		os:         make(map[string]*roaring.Bitmap),
		deviceType: make(map[string]*roaring.Bitmap),
		dayOfWeek:  make(map[uint8]*roaring.Bitmap),
		hourRange:  make(map[uint8]*roaring.Bitmap),
		contentID:  make(map[string]*roaring.Bitmap),
		category:   make(map[string]*roaring.Bitmap),
		audience:   make(map[int64]*roaring.Bitmap),
		online:     online,
	}
}

func (idx *InvertedIndex) IsReady() bool  { return idx.ready.Load() }
func (idx *InvertedIndex) Version() int64 { return idx.version.Load() }

func getOrCreateBitmap(m map[string]*roaring.Bitmap, key string) *roaring.Bitmap {
	bm, ok := m[key]
	if !ok {
		bm = roaring.New()
		m[key] = bm
	}
	return bm
}

func getOrCreateInt32Bitmap(m map[int32]*roaring.Bitmap, key int32) *roaring.Bitmap {
	bm, ok := m[key]
	if !ok {
		bm = roaring.New()
		m[key] = bm
	}
	return bm
}

func getOrCreateUint8Bitmap(m map[uint8]*roaring.Bitmap, key uint8) *roaring.Bitmap {
	bm, ok := m[key]
	if !ok {
		bm = roaring.New()
		m[key] = bm
	}
	return bm
}

func getOrCreateInt64Bitmap(m map[int64]*roaring.Bitmap, key int64) *roaring.Bitmap {
	bm, ok := m[key]
	if !ok {
		bm = roaring.New()
		m[key] = bm
	}
	return bm
}

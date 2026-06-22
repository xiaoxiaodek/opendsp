package adapter

type UnifiedBidRequest struct {
	RequestID      string
	MediaID        string
	IsTest         bool
	IsPing         bool
	IsPMP          bool         // PDB/PD deal flag
	CampaignID     string       // PDB campaign ID
	Device         UnifiedDevice
	User           UnifiedUser
	Content        UnifiedContent
	Imps           []UnifiedImpression
	FloorPrices    []FloorPrice // industry-specific bid floors
	UserDMPIDs     []string     // DMP audience IDs
	UserFeature    []string     // user feature tags
	ConnectionType string       // wifi / 4g / etc.
}

type FloorPrice struct {
	IndustryID     int32
	Price          float64
	SkippablePrice float64
}

type UnifiedDevice struct {
	OS             string
	DeviceType     string
	IP             string
	UA             string
	DeviceID       string
	GeoCity        string
	GeoCountry     string
	Carrier        string
	Make           string
	Model          string
	ScreenWidth    int32
	ScreenHeight   int32
	ConnectionType int32  // wifi/cellular/ethernet
	PlatformID     string // platform type code (iQiyi)
	OSVersion      string
}

type UnifiedUser struct {
	UserID  string
	Gender  string
	Age     int32
	DMPIDs  []string // DMP audience IDs
	Feature []string // user feature tags
	Session string   // session ID
}

type UnifiedContent struct {
	ContentID string
	Title     string
	Category  string
	Tags      []string
	Keywords  []string
	Duration  int32
	URL       string
}

type UnifiedImpression struct {
	ImpID             string
	PositionType      int32
	Width             int32
	Height            int32
	MinDuration       int32
	MaxDuration       int32
	BidFloor          float64
	AdPositionID      string
	CampaignID        string       // for PDB/PD deals
	IsPMP             bool
	BlockedAdTag      []string     // blocked advertiser tags
	CreativeTemplates []int32      // allowed creative templates (banner)
	FloorPrices       []FloorPrice // per-impression floor prices
}

type UnifiedBidResponse struct {
	RequestID        string
	ProcessingTimeMs int32
	SeatBids         []UnifiedSeatBid
}

type UnifiedSeatBid struct {
	Bids []UnifiedBid
}

type UnifiedBid struct {
	ImpID           string
	Price           float64
	AdMarkup        string
	CreativeID      string
	LandingURL      string
	DeeplinkURL     string
	ImpTrackers     []string
	ClickTrackers   []string
	Width           int32
	Height          int32
	Duration        int32
	StartDelay      int32                  // in-video position preference
	DeeplinkApp     string                 // APK package name
	TrackingEvents  map[string][]string    // playback progress events
	IconURL         string                 // DSP logo URL
	ClickType       string                 // 0/4/11/14/15/67
	PlatformCrID    string                 // platform-side creative ID (tv_id)
	ClickThroughURL string                 // explicit landing URL for ClickThrough
}

type BidAdapter interface {
	Name() string
	ContentType() string
	ResponseContentType() string
	ParseRequest(raw []byte) (*UnifiedBidRequest, error)
	BuildResponse(req *UnifiedBidRequest, resp *UnifiedBidResponse) ([]byte, error)
}

type Registry struct {
	adapters map[string]BidAdapter
	routes   map[string]BidAdapter
}

func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]BidAdapter),
		routes:   make(map[string]BidAdapter),
	}
}

func (r *Registry) Register(adapter BidAdapter, path string) {
	r.adapters[adapter.Name()] = adapter
	r.routes[path] = adapter
}

func (r *Registry) Match(path string) (BidAdapter, bool) {
	adapter, ok := r.routes[path]
	return adapter, ok
}

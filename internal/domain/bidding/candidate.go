package bidding

import (
	"github.com/opendsp/opendsp/internal/domain/budget"
	"github.com/opendsp/opendsp/internal/domain/feature"
)

// Candidate represents an ad group matched for a bid request,
// enriched through the pipeline stages.
type Candidate struct {
	AdGroupID    uint32
	CreativeID   int64
	BidPrice     float64 // original bid price from ad group
	AdvertiserID int64   // advertiser owning the campaign (for budget guard)
	CampaignID   int64   // campaign ID (for time-window checks)

	// Populated by Scoring stage
	PredCTR float64
	PredCVR float64

	// Populated by Pricing stage
	ECPM       int64  // effective CPM in micros
	FinalScore float64 // composite ranking score

	// Populated by FeatureAssembly stage
	Features feature.FeatureSet

	// Populated by BudgetGuard stage
	BudgetToken *budget.PreFreezeToken
}

// NewCandidate creates a Candidate from ad group match data.
func NewCandidate(adGroupID uint32, creativeID int64, bidPrice float64) *Candidate {
	return &Candidate{
		AdGroupID:  adGroupID,
		CreativeID: creativeID,
		BidPrice:   bidPrice,
		Features:   feature.NewFeatureSet(),
	}
}

// NewCandidateWithAdvertiser creates a Candidate with advertiser context.
func NewCandidateWithAdvertiser(adGroupID uint32, creativeID int64, bidPrice float64, advertiserID, campaignID int64) *Candidate {
	return &Candidate{
		AdGroupID:    adGroupID,
		CreativeID:   creativeID,
		BidPrice:     bidPrice,
		AdvertiserID: advertiserID,
		CampaignID:   campaignID,
		Features:     feature.NewFeatureSet(),
	}
}

// HasPredictions returns true if both pCTR and pCVR are set.
func (c *Candidate) HasPredictions() bool {
	return c.PredCTR > 0 && c.PredCVR > 0
}

// BidRequest represents a parsed and validated bid request from an ADX.
// It is immutable — stages should not modify it.
type BidRequest struct {
	RequestID    string
	MediaID      string
	PositionType int32

	// User context
	UserID     string
	DeviceID   string
	DeviceType string
	OS         string
	IP         string
	UserAgent  string
	IsTest     bool

	// Geo context
	GeoCity   string
	GeoRegion string

	// Content context
	ContentID string
	Category  string

	// Creative requirements
	Width       int32
	Height      int32
	MinDuration int32
	MaxDuration int32

	// DMP audience IDs from the request
	AudienceIDs []int64
}

// Package abtest defines the A/B Testing domain model for controlled experiments
// on bidding strategies, model versions, and pricing algorithms.
package abtest

import "time"

// Experiment represents an A/B test that splits traffic between variants.
type Experiment struct {
	ID          int64
	Name        string
	Description string
	Status      string // "draft", "running", "paused", "completed"

	// Traffic split: each variant gets a percentage of traffic (sum = 100).
	Variants []Variant

	// Targeting: which campaigns/advertisers are in the experiment.
	AdvertiserIDs []int64
	CampaignIDs   []int64

	StartAt time.Time
	EndAt   time.Time
	CreatedAt time.Time
}

// Variant is one arm of an A/B test.
type Variant struct {
	Name       string
	Percentage int32  // percentage of traffic (0-100)

	// Which pipeline configuration to use for this variant.
	// Each variant can have different scoring models, pricing strategies, etc.
	ConfigOverrides map[string]interface{}
}

// Assignment maps a bid request to an experiment variant.
type Assignment struct {
	ExperimentID int64
	VariantName  string
}

// Metric records a metric for a specific variant over time.
type Metric struct {
	ExperimentID int64
	VariantName  string
	Date         time.Time

	Impressions int64
	Clicks      int64
	Conversions int64
	CostMicros  int64
	RevenueMicros int64

	CTR   float64
	CVR   float64
	ROAS  float64
	ECPM  float64
}

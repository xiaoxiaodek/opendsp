package biz

import "errors"

var (
	ErrCampaignNotFound   = errors.New("campaign not found")
	ErrCampaignCompleted  = errors.New("campaign already completed")
	ErrCampaignNotActive  = errors.New("campaign is not active")
	ErrAdGroupNotFound    = errors.New("adgroup not found")
	ErrCreativeNotFound   = errors.New("creative not found")
	ErrInsufficientBudget = errors.New("insufficient budget")
)

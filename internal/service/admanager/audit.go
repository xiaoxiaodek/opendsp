package admanager

import (
	"context"
	"fmt"
	"strings"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data"
)

type AutoAuditor struct {
	data *data.Data
}

func NewAutoAuditor(d *data.Data) *AutoAuditor {
	return &AutoAuditor{data: d}
}

type AuditRule struct {
	MaxSizeKB    int32
	MinWidth     int32
	MinHeight    int32
	MaxWidth     int32
	MaxHeight    int32
	AllowedMimes []string
	MinDuration  int32
	MaxDuration  int32
}

var defaultRules = map[int16]AuditRule{
	biz.CreativeTypeImage: {
		MaxSizeKB:    2048,
		MinWidth:     100,
		MinHeight:    100,
		MaxWidth:     4096,
		MaxHeight:    4096,
		AllowedMimes: []string{"image/jpeg", "image/png", "image/gif"},
	},
	biz.CreativeTypeVideo: {
		MaxSizeKB:    102400,
		MinWidth:     640,
		MinHeight:    360,
		MaxWidth:     3840,
		MaxHeight:    2160,
		AllowedMimes: []string{"video/mp4", "video/x-flv", "video/webm"},
		MinDuration:  5,
		MaxDuration:  120,
	},
}

func (a *AutoAuditor) Audit(ctx context.Context, creative *biz.Creative) (bool, string) {
	rule, ok := defaultRules[creative.CreativeType]
	if !ok {
		return false, fmt.Sprintf("unsupported creative type: %d", creative.CreativeType)
	}

	if creative.AssetSize != nil && *creative.AssetSize > rule.MaxSizeKB*1024 {
		return false, fmt.Sprintf("file size %dKB exceeds limit %dKB", *creative.AssetSize/1024, rule.MaxSizeKB)
	}

	if creative.AssetWidth < rule.MinWidth || creative.AssetWidth > rule.MaxWidth {
		return false, fmt.Sprintf("width %d out of range [%d, %d]", creative.AssetWidth, rule.MinWidth, rule.MaxWidth)
	}

	if creative.AssetHeight < rule.MinHeight || creative.AssetHeight > rule.MaxHeight {
		return false, fmt.Sprintf("height %d out of range [%d, %d]", creative.AssetHeight, rule.MinHeight, rule.MaxHeight)
	}

	if creative.AssetMime != "" {
		mimeOK := false
		for _, m := range rule.AllowedMimes {
			if strings.EqualFold(creative.AssetMime, m) {
				mimeOK = true
				break
			}
		}
		if !mimeOK {
			return false, fmt.Sprintf("unsupported mime type: %s, allowed: %v", creative.AssetMime, rule.AllowedMimes)
		}
	}

	if rule.MinDuration > 0 && creative.AssetDuration < rule.MinDuration {
		return false, fmt.Sprintf("duration %ds below minimum %ds", creative.AssetDuration, rule.MinDuration)
	}
	if rule.MaxDuration > 0 && creative.AssetDuration > rule.MaxDuration {
		return false, fmt.Sprintf("duration %ds exceeds maximum %ds", creative.AssetDuration, rule.MaxDuration)
	}

	if creative.LandingURL == "" {
		return false, "landing URL is required"
	}

	return true, ""
}

func (s *AdManagerService) CreateCreativeWithAudit(ctx context.Context, creative *biz.Creative) error {
	auditor := NewAutoAuditor(nil)
	ok, reason := auditor.Audit(ctx, creative)

	if ok {
		creative.Approve()
	} else {
		creative.Reject(reason)
	}

	if err := s.creativeUC.Create(ctx, creative); err != nil {
		return err
	}

	return nil
}

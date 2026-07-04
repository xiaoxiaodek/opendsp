package fraud

import (
	"context"

	domainFraud "github.com/opendsp/opendsp/internal/domain/fraud"
)

// postBidChecker implements domainFraud.PostBidChecker using SlidingWindow.
type postBidChecker struct {
	sw     *SlidingWindow
	writer *EventWriter
}

// NewPostBidChecker creates a PostBidChecker backed by sliding window detection.
func NewPostBidChecker(sw *SlidingWindow, writer *EventWriter) domainFraud.PostBidChecker {
	return &postBidChecker{sw: sw, writer: writer}
}

// CheckImpression runs post-bid fraud checks on an impression event.
func (c *postBidChecker) CheckImpression(ctx context.Context, event domainFraud.ImpressionEvent) []string {
	if !c.sw.cfg.Enabled {
		return nil
	}

	var reasons []string

	// CTR anomaly: record impression
	if c.sw.cfg.CTRAnomaly.WindowMs > 0 {
		blocked, _, _, err := c.sw.CheckCTR(ctx, event.MediaID, event.PositionID, event.RequestID, false)
		if err != nil {
			return reasons
		}
		if blocked {
			reasons = append(reasons, domainFraud.ReasonCTRAnomaly)
			c.writer.Write(ctx, FraudEvent{
				RequestID: event.RequestID,
				RuleType:  domainFraud.ReasonCTRAnomaly,
				RuleValue: event.IP,
				RiskScore: 1.0,
				Action:    "flagged",
			})
		}
	}

	return reasons
}

// CheckClick runs post-bid fraud checks on a click event.
func (c *postBidChecker) CheckClick(ctx context.Context, event domainFraud.ClickEvent) []string {
	if !c.sw.cfg.Enabled {
		return nil
	}

	var reasons []string

	// CTR anomaly: record click
	if c.sw.cfg.CTRAnomaly.WindowMs > 0 {
		blocked, _, _, err := c.sw.CheckCTR(ctx, event.MediaID, event.PositionID, event.RequestID, true)
		if err != nil {
			return reasons
		}
		if blocked {
			reasons = append(reasons, domainFraud.ReasonCTRAnomaly)
			c.writer.Write(ctx, FraudEvent{
				RequestID: event.RequestID,
				RuleType:  domainFraud.ReasonCTRAnomaly,
				RuleValue: event.IP,
				RiskScore: 1.0,
				Action:    "flagged",
			})
		}
	}

	// Device diversity checks (only when DeviceID is present)
	if event.DeviceID != "" && c.sw.cfg.DeviceDiversity.WindowMs > 0 {
		if blocked, err := c.sw.CheckIPDiversity(ctx, event.DeviceID, event.IP); err == nil && blocked {
			reasons = append(reasons, domainFraud.ReasonIPDiversity)
			c.flagAndBlacklist(ctx, event, domainFraud.ReasonIPDiversity)
		}
		if event.UserAgent != "" {
			if blocked, err := c.sw.CheckUADiversity(ctx, event.DeviceID, event.UserAgent); err == nil && blocked {
				reasons = append(reasons, domainFraud.ReasonUADiversity)
				c.flagAndBlacklist(ctx, event, domainFraud.ReasonUADiversity)
			}
		}
	}

	return reasons
}

func (c *postBidChecker) flagAndBlacklist(ctx context.Context, event domainFraud.ClickEvent, reason string) {
	c.writer.Write(ctx, FraudEvent{
		RequestID: event.RequestID,
		RuleType:  reason,
		RuleValue: event.IP,
		RiskScore: 1.0,
		Action:    "flagged",
	})

	if event.IP != "" {
		c.sw.AddDynamicBlacklist(ctx, "ip", event.IP)
	}
	if event.DeviceID != "" {
		c.sw.AddDynamicBlacklist(ctx, "device", event.DeviceID)
	}
}

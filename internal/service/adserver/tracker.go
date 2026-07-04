package adserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data"
	domainFraud "github.com/opendsp/opendsp/internal/domain/fraud"
	"github.com/opendsp/opendsp/internal/freq"
)

var pixelGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61,
	0x01, 0x00, 0x01, 0x00, 0x80, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0x00, 0x00,
	0x00, 0x21, 0xF9, 0x04, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x2C, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00,
	0x00, 0x02, 0x02, 0x44, 0x01, 0x00,
	0x3B,
}

// Tracker handles impression and click tracking.
type Tracker struct {
	freqCtrl       *freq.Controller
	tokenBucket    *freq.TokenBucket
	data           *data.Data
	eventBuf       chan biz.StatEvent
	postBidChecker domainFraud.PostBidChecker
	mu             sync.Mutex
}

// NewTracker creates a new Tracker with frequency control only (backward compatible).
func NewTracker(freqCtrl *freq.Controller, d *data.Data) *Tracker {
	t := &Tracker{
		freqCtrl: freqCtrl,
		data:     d,
		eventBuf: make(chan biz.StatEvent, 10000),
	}
	go t.flushLoop()
	return t
}

// NewTrackerWithBudget creates a new Tracker with token bucket budget control.
func NewTrackerWithBudget(freqCtrl *freq.Controller, tb *freq.TokenBucket, d *data.Data) *Tracker {
	t := &Tracker{
		freqCtrl:    freqCtrl,
		tokenBucket: tb,
		data:        d,
		eventBuf:    make(chan biz.StatEvent, 10000),
	}
	go t.flushLoop()
	return t
}

// NewTrackerWithPostBid creates a Tracker with post-bid fraud checking.
func NewTrackerWithPostBid(freqCtrl *freq.Controller, d *data.Data, checker domainFraud.PostBidChecker) *Tracker {
	t := &Tracker{
		freqCtrl:       freqCtrl,
		data:           d,
		eventBuf:       make(chan biz.StatEvent, 10000),
		postBidChecker: checker,
	}
	go t.flushLoop()
	return t
}

func (t *Tracker) flushLoop() {
	batch := make([]biz.StatEvent, 0, 100)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case e := <-t.eventBuf:
			batch = append(batch, e)
			if len(batch) >= 100 {
				t.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				t.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

func (t *Tracker) flush(batch []biz.StatEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	repo := data.NewReportRepo(t.data)
	repo.InsertEvents(ctx, batch)
}

// HandleImpression processes an impression tracking request.
// It confirms the token bucket reservation (if present) and logs the event.
func (t *Tracker) HandleImpression(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	isTest := q.Get("is_test") == "1"
	adgroupID, _ := strconv.ParseInt(q.Get("adgroup_id"), 10, 64)
	creativeID, _ := strconv.ParseInt(q.Get("creative_id"), 10, 64)
	price, _ := strconv.ParseFloat(q.Get("price"), 64)
	clickID := q.Get("click_id")
	reservationID := q.Get("reservation_id")
	campaignID, _ := strconv.ParseInt(q.Get("campaign_id"), 10, 64)
	advertiserID, _ := strconv.ParseInt(q.Get("advertiser_id"), 10, 64)

	// Confirm token bucket reservation if present
	if t.tokenBucket != nil && reservationID != "" && campaignID > 0 && !isTest {
		ok, _, err := t.tokenBucket.Confirm(r.Context(), reservationID, campaignID, adgroupID)
		if err != nil {
			log.Printf("budget confirm error: reservation=%s err=%v", reservationID, err)
		} else if !ok {
			log.Printf("budget confirm failed: reservation=%s (may have expired)", reservationID)
		}
	}

	// Existing frequency/budget check
	result, _ := t.freqCtrl.Check(r.Context(), freq.CheckParams{
		AdGroupID: adgroupID,
		UserID:    q.Get("uid"),
		BidPrice:  price,
	})

	if !isTest {
		t.eventBuf <- biz.StatEvent{
			EventType:    biz.EventImpression,
			AdGroupID:    adgroupID,
			CreativeID:   creativeID,
			CampaignID:   campaignID,
			AdvertiserID: advertiserID,
			Price:        price,
			FreqResult:   result.Reason,
			ClickID:      clickID,
			IP:           r.RemoteAddr,
			UA:           r.Header.Get("User-Agent"),
			EventTime:    time.Now(),
		}
	}

	if t.postBidChecker != nil && !isTest {
		go func() {
			t.postBidChecker.CheckImpression(context.Background(), domainFraud.ImpressionEvent{
				RequestID: clickID,
				IP:        r.RemoteAddr,
				UserAgent: r.Header.Get("User-Agent"),
			})
		}()
	}

	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(pixelGIF)
}

// HandleClick processes a click tracking request.
func (t *Tracker) HandleClick(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	isTest := q.Get("is_test") == "1"
	adgroupID, _ := strconv.ParseInt(q.Get("adgroup_id"), 10, 64)
	creativeID, _ := strconv.ParseInt(q.Get("creative_id"), 10, 64)
	clickID := q.Get("click_id")
	campaignID, _ := strconv.ParseInt(q.Get("campaign_id"), 10, 64)
	advertiserID, _ := strconv.ParseInt(q.Get("advertiser_id"), 10, 64)

	if !isTest {
		t.eventBuf <- biz.StatEvent{
			EventType:    biz.EventClick,
			AdGroupID:    adgroupID,
			CreativeID:   creativeID,
			CampaignID:   campaignID,
			AdvertiserID: advertiserID,
			ClickID:      clickID,
			IP:           r.RemoteAddr,
			UA:           r.Header.Get("User-Agent"),
			EventTime:    time.Now(),
		}
	}

	if t.postBidChecker != nil && !isTest {
		go func() {
			t.postBidChecker.CheckClick(context.Background(), domainFraud.ClickEvent{
				RequestID: clickID,
				IP:        r.RemoteAddr,
				UserAgent: r.Header.Get("User-Agent"),
			})
		}()
	}

	landingURL := q.Get("url")
	if landingURL == "" {
		landingURL = "https://opendsp.io"
	}

	w.Header().Set("Location", landingURL)
	w.WriteHeader(http.StatusFound)
	fmt.Fprintf(w, `<html><body><a href="%s">%s</a></body></html>`, landingURL, landingURL)
}

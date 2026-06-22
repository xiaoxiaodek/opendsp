package adserver

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data"
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

type Tracker struct {
	freqCtrl *freq.Controller
	data     *data.Data
	eventBuf chan biz.StatEvent
	mu       sync.Mutex
}

func NewTracker(freqCtrl *freq.Controller, d *data.Data) *Tracker {
	t := &Tracker{
		freqCtrl: freqCtrl,
		data:     d,
		eventBuf: make(chan biz.StatEvent, 10000),
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

func (t *Tracker) HandleImpression(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	adgroupID, _ := strconv.ParseInt(q.Get("adgroup_id"), 10, 64)
	creativeID, _ := strconv.ParseInt(q.Get("creative_id"), 10, 64)
	price, _ := strconv.ParseFloat(q.Get("price"), 64)
	clickID := q.Get("click_id")

	result, _ := t.freqCtrl.Check(r.Context(), freq.CheckParams{
		AdGroupID: adgroupID,
		UserID:    q.Get("uid"),
		BidPrice:  price,
	})

	t.eventBuf <- biz.StatEvent{
		EventType:  biz.EventImpression,
		AdGroupID:  adgroupID,
		CreativeID: creativeID,
		Price:      price,
		FreqResult: result.Reason,
		ClickID:    clickID,
		IP:         r.RemoteAddr,
		UA:         r.Header.Get("User-Agent"),
		EventTime:  time.Now(),
	}

	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(pixelGIF)
}

func (t *Tracker) HandleClick(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	adgroupID, _ := strconv.ParseInt(q.Get("adgroup_id"), 10, 64)
	creativeID, _ := strconv.ParseInt(q.Get("creative_id"), 10, 64)
	clickID := q.Get("click_id")

	t.eventBuf <- biz.StatEvent{
		EventType:  biz.EventClick,
		AdGroupID:  adgroupID,
		CreativeID: creativeID,
		ClickID:    clickID,
		IP:         r.RemoteAddr,
		UA:         r.Header.Get("User-Agent"),
		EventTime:  time.Now(),
	}

	landingURL := q.Get("url")
	if landingURL == "" {
		landingURL = "https://opendsp.io"
	}

	w.Header().Set("Location", landingURL)
	w.WriteHeader(http.StatusFound)
	fmt.Fprintf(w, `<html><body><a href="%s">%s</a></body></html>`, landingURL, landingURL)
}

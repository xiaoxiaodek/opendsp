package adserver

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/opendsp/opendsp/internal/service/adserver/adapter"
	"github.com/opendsp/opendsp/pkg/vast"
)

type Server struct {
	engine   *Engine
	tracker  *Tracker
	registry *adapter.Registry
}

func NewServer(engine *Engine, tracker *Tracker) *Server {
	s := &Server{
		engine:   engine,
		tracker:  tracker,
		registry: adapter.NewRegistry(),
	}

	s.registry.Register(&adapter.OpenRTBAdapter{}, "/rtb/openrtb")
	s.registry.Register(&adapter.IqiyiAdapter{}, "/rtb/iqiyi")

	return s
}

func (s *Server) RegisterAdapter(adp adapter.BidAdapter, path string) {
	s.registry.Register(adp, path)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/track/impression":
		s.tracker.HandleImpression(w, r)
		return
	case "/track/click":
		s.tracker.HandleClick(w, r)
		return
	case "/ready":
		if s.engine.index.IsReady() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("not ready"))
		}
		return
	case "/health":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return
	case "/cookie/sync":
		redirectURL := r.URL.Query().Get("redirect_url")
		if redirectURL == "" {
			http.Error(w, "missing redirect_url", http.StatusBadRequest)
			return
		}
		dspHost := os.Getenv("DSP_HOST")
		if dspHost == "" {
			dspHost = r.Host
		}
		qiyiURL := fmt.Sprintf("https://ckm.iqiyi.com/pixel?redirect=%s",
			url.QueryEscape(fmt.Sprintf("https://%s/cookie/map?redirect=%s",
				dspHost, url.QueryEscape(redirectURL))))
		http.Redirect(w, r, qiyiURL, http.StatusFound)
		return
	case "/cookie/map":
		qiyiUID := r.URL.Query().Get("qiyi_uid")
		redirectURL := r.URL.Query().Get("redirect")
		if qiyiUID == "" || redirectURL == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		log.Printf("cookie mapping: qiyi_uid=%s redirect=%s", qiyiUID, redirectURL)
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	case "/win/iqi":
		encoded := r.URL.Query().Get("settlement")
		bidID := r.URL.Query().Get("bid_id")
		token := os.Getenv("IQIYI_SETTLEMENT_TOKEN")
		if encoded != "" && bidID != "" && token != "" {
			price, err := DecodeSettlementPrice(encoded, token, bidID)
			if err != nil {
				log.Printf("settlement decode error: %v", err)
			} else {
				log.Printf("settlement decoded: bid=%s price=%d (分)", bidID, price)
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	adp, ok := s.registry.Match(r.URL.Path)
	if !ok {
		http.Error(w, "unknown endpoint", http.StatusNotFound)
		return
	}

	if r.Header.Get("Content-Type") != adp.ContentType() {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	if len(raw) > 0 && r.Header.Get("Content-Encoding") == "gzip" {
		gr, gzErr := gzip.NewReader(strings.NewReader(string(raw)))
		if gzErr != nil {
			http.Error(w, "decompress failed", http.StatusBadRequest)
			return
		}
		decompressed, gzErr := io.ReadAll(gr)
		gr.Close()
		if gzErr != nil {
			http.Error(w, "decompress read failed", http.StatusBadRequest)
			return
		}
		raw = decompressed
	}

	req, err := adp.ParseRequest(raw)
	if err != nil {
		http.Error(w, "parse request failed", http.StatusBadRequest)
		return
	}

	if req.IsPing {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	s.engine.RefreshExclusions(r.Context(), req.User.UserID)

	var seatBids []adapter.UnifiedSeatBid
	for _, imp := range req.Imps {
		result := s.engine.Bid(r.Context(), BidParams{
			MediaID:      req.MediaID,
			PositionType: imp.PositionType,
			GeoCity:      req.Device.GeoCity,
			OS:           req.Device.OS,
			DeviceType:   req.Device.DeviceType,
			ContentID:    req.Content.ContentID,
			Category:     req.Content.Category,
			Width:        imp.Width,
			Height:       imp.Height,
			MinDuration:  imp.MinDuration,
			MaxDuration:  imp.MaxDuration,
			UserID:       req.User.UserID,
			AudienceID:   extractAudienceID(req),
			IsTest:       req.IsTest,
		})

		if result == nil {
			continue
		}

		// Enrich impression tracker URLs with reservation_id, campaign_id and advertiser_id
		enrichedTrackers := make([]string, len(result.ImpTrackers))
		for i, tracker := range result.ImpTrackers {
			enrichedTrackers[i] = appendReservationParams(tracker, result.ReservationID, result.CampaignID, result.AdvertiserID)
		}

		enrichedClickTrackers := make([]string, len(result.ClickTrackers))
		for i, tracker := range result.ClickTrackers {
			enrichedClickTrackers[i] = appendIDParams(tracker, result.CampaignID, result.AdvertiserID)
		}

		vastXML, _ := vast.BuildVast(vast.AdParams{
			AdID:           result.PlatformCrID,
			Title:          result.Creative.Title,
			Duration:       result.Duration,
			AssetURL:       result.AssetURL,
			AssetMime:      result.Creative.AssetMime,
			Width:          result.Width,
			Height:         result.Height,
			ImpTrackers:    enrichedTrackers,
			ClickID:        result.ClickID,
			ClickType:      result.ClickType,
			ClickTrackers:  result.ClickTrackers,
			ClickThroughURL: result.ClickThroughURL,
			TrackingEvents: result.TrackingEvents,
			DeeplinkApp:    result.DeeplinkApp,
			IconURL:        result.IconURL,
		})

		seatBids = append(seatBids, adapter.UnifiedSeatBid{
			Bids: []adapter.UnifiedBid{{
				ImpID:          imp.ImpID,
				Price:          result.Price,
				AdMarkup:       vastXML,
				CreativeID:     strconv.FormatInt(result.Creative.ID, 10),
				PlatformCrID:   result.PlatformCrID,
				LandingURL:     result.LandingURL,
				DeeplinkURL:    result.DeeplinkURL,
				ImpTrackers:    result.ImpTrackers,
				ClickTrackers:  enrichedClickTrackers,
				Width:          result.Width,
				Height:         result.Height,
				Duration:       result.Duration,
				DeeplinkApp:    result.DeeplinkApp,
				ClickType:      result.ClickType,
				ClickThroughURL: result.ClickThroughURL,
				TrackingEvents: result.TrackingEvents,
				IconURL:        result.IconURL,
			}},
		})
	}

	if len(seatBids) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := &adapter.UnifiedBidResponse{
		RequestID: req.RequestID,
		SeatBids:  seatBids,
	}

	out, err := adp.BuildResponse(req, resp)
	if err != nil {
		http.Error(w, "build response failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", adp.ResponseContentType())

	// Gzip compression support
	acceptEncoding := r.Header.Get("Accept-Encoding")
	if strings.Contains(acceptEncoding, "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gw := gzip.NewWriter(w)
		defer gw.Close()
		gw.Write(out)
	} else {
		w.Write(out)
	}
}

// appendReservationParams appends reservation_id, campaign_id and advertiser_id as query params to a tracker URL.
func appendReservationParams(trackerURL, reservationID string, campaignID, advertiserID int64) string {
	if reservationID == "" {
		return trackerURL
	}
	sep := "&"
	if !strings.Contains(trackerURL, "?") {
		sep = "?"
	}
	return fmt.Sprintf("%s%sreservation_id=%s&campaign_id=%d&advertiser_id=%d",
		trackerURL, sep, url.QueryEscape(reservationID), campaignID, advertiserID)
}

func appendIDParams(trackerURL string, campaignID, advertiserID int64) string {
	if campaignID == 0 && advertiserID == 0 {
		return trackerURL
	}
	sep := "&"
	if !strings.Contains(trackerURL, "?") {
		sep = "?"
	}
	return fmt.Sprintf("%s%scampaign_id=%d&advertiser_id=%d",
		trackerURL, sep, campaignID, advertiserID)
}

func extractAudienceID(req *adapter.UnifiedBidRequest) int64 {
	for _, id := range req.UserDMPIDs {
		if audienceID, err := strconv.ParseInt(id, 10, 64); err == nil {
			return audienceID
		}
	}
	return 0
}

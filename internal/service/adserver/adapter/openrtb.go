package adapter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type OpenRTBAdapter struct{}

func (a *OpenRTBAdapter) Name() string               { return "openrtb" }
func (a *OpenRTBAdapter) ContentType() string         { return "application/json" }
func (a *OpenRTBAdapter) ResponseContentType() string { return "application/json" }

type openrtbBidRequest struct {
	ID     string         `json:"id"`
	Imp    []openrtbImp   `json:"imp"`
	Device *openrtbDevice `json:"device,omitempty"`
	User   *openrtbUser   `json:"user,omitempty"`
	Site   *openrtbSite   `json:"site,omitempty"`
	IsTest int            `json:"is_test,omitempty"`
}

type openrtbImp struct {
	ID       string         `json:"id"`
	Banner   *openrtbBanner `json:"banner,omitempty"`
	Video    *openrtbVideo  `json:"video,omitempty"`
	BidFloor float64        `json:"bidfloor"`
	Secure   int            `json:"secure,omitempty"`
}

type openrtbBanner struct {
	W   int32 `json:"w"`
	H   int32 `json:"h"`
	Pos int32 `json:"pos,omitempty"`
}

type openrtbVideo struct {
	W           int32    `json:"w"`
	H           int32    `json:"h"`
	MinDuration int32    `json:"minduration"`
	MaxDuration int32    `json:"maxduration"`
	MIMEs       []string `json:"mimes,omitempty"`
}

type openrtbDevice struct {
	UA         string      `json:"ua"`
	IP         string      `json:"ip"`
	OS         string      `json:"os"`
	DeviceType int32       `json:"devicetype,omitempty"`
	Make       string      `json:"make,omitempty"`
	Model      string      `json:"model,omitempty"`
	IFA        string      `json:"ifa,omitempty"`
	Geo        *openrtbGeo `json:"geo,omitempty"`
}

type openrtbGeo struct {
	Country string `json:"country,omitempty"`
	City    string `json:"city,omitempty"`
}

type openrtbUser struct {
	ID       string `json:"id,omitempty"`
	BuyerUID string `json:"buyeruid,omitempty"`
}

type openrtbSite struct {
	ID      string          `json:"id,omitempty"`
	Domain  string          `json:"domain,omitempty"`
	Content *openrtbContent `json:"content,omitempty"`
}

type openrtbContent struct {
	ID       string   `json:"id,omitempty"`
	Title    string   `json:"title,omitempty"`
	Cat      []string `json:"cat,omitempty"`
	Keywords string   `json:"keywords,omitempty"`
}

type bid struct {
	ID    string  `json:"id"`
	ImpID string  `json:"impid"`
	Price float64 `json:"price"`
	Adm   string  `json:"adm,omitempty"`
	CrID  string  `json:"crid,omitempty"`
}

type seatbid struct {
	Bid []bid `json:"bid"`
}

type bidResponse struct {
	ID      string    `json:"id"`
	SeatBid []seatbid `json:"seatbid"`
}

func (a *OpenRTBAdapter) ParseRequest(raw []byte) (*UnifiedBidRequest, error) {
	var req openrtbBidRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("parse openrtb: %w", err)
	}

	unified := &UnifiedBidRequest{
		RequestID: req.ID,
		IsTest:    req.IsTest == 1,
	}

	if req.Device != nil {
		deviceType := "mobile"
		switch req.Device.DeviceType {
		case 1:
			deviceType = "mobile"
		case 2:
			deviceType = "pc"
		case 3:
			deviceType = "ott"
		}
		unified.Device = UnifiedDevice{
			OS:         req.Device.OS,
			DeviceType: deviceType,
			IP:         req.Device.IP,
			UA:         req.Device.UA,
			DeviceID:   req.Device.IFA,
			Make:       req.Device.Make,
			Model:      req.Device.Model,
		}
		if req.Device.Geo != nil {
			unified.Device.GeoCity = req.Device.Geo.City
			unified.Device.GeoCountry = req.Device.Geo.Country
		}
	}

	if req.User != nil {
		unified.User = UnifiedUser{
			UserID: req.User.BuyerUID,
		}
	}

	if req.Site != nil {
		unified.MediaID = req.Site.Domain
		if req.Site.Content != nil {
			unified.Content = UnifiedContent{
				ContentID: req.Site.Content.ID,
				Title:     req.Site.Content.Title,
				Category:  firstOrEmpty(req.Site.Content.Cat),
				Keywords:  splitKeywords(req.Site.Content.Keywords),
			}
		}
	}

	for _, imp := range req.Imp {
		ui := UnifiedImpression{ImpID: imp.ID, BidFloor: imp.BidFloor}
		if imp.Banner != nil {
			ui.Width = imp.Banner.W
			ui.Height = imp.Banner.H
			ui.PositionType = imp.Banner.Pos
		}
		if imp.Video != nil {
			ui.Width = imp.Video.W
			ui.Height = imp.Video.H
			ui.MinDuration = imp.Video.MinDuration
			ui.MaxDuration = imp.Video.MaxDuration
			if ui.PositionType == 0 {
				ui.PositionType = 1
			}
		}
		unified.Imps = append(unified.Imps, ui)
	}

	return unified, nil
}

func (a *OpenRTBAdapter) BuildResponse(req *UnifiedBidRequest, resp *UnifiedBidResponse) ([]byte, error) {
	br := bidResponse{ID: resp.RequestID}
	for _, seat := range resp.SeatBids {
		sb := seatbid{}
		for _, b := range seat.Bids {
			sb.Bid = append(sb.Bid, bid{
				ID:    uuid.New().String(),
				ImpID: b.ImpID,
				Price: b.Price,
				Adm:   b.AdMarkup,
				CrID:  b.CreativeID,
			})
		}
		br.SeatBid = append(br.SeatBid, sb)
	}

	return json.Marshal(br)
}

func firstOrEmpty(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

func splitKeywords(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, kw := range strings.Split(s, ",") {
		result = append(result, strings.TrimSpace(kw))
	}
	return result
}

package encoder

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"

	"github.com/opendsp/opendsp/internal/mockadx/generator"
)

type OpenRTBEncoder struct {
	gzip bool
}

func NewOpenRTBEncoder(gzip bool) *OpenRTBEncoder {
	return &OpenRTBEncoder{gzip: gzip}
}

func (e *OpenRTBEncoder) ContentType() string {
	return "application/json"
}

type openrtbReq struct {
	ID     string       `json:"id"`
	Imp    []openrtbImp `json:"imp"`
	Device openrtbDev   `json:"device"`
	User   openrtbUsr   `json:"user"`
	Site   openrtbSite  `json:"site"`
	IsTest int          `json:"is_test"`
}

type openrtbImp struct {
	ID       string      `json:"id"`
	Video    *openrtbVid `json:"video,omitempty"`
	BidFloor float64     `json:"bidfloor"`
}

type openrtbVid struct {
	W           int32    `json:"w"`
	H           int32    `json:"h"`
	MinDuration int32    `json:"minduration"`
	MaxDuration int32    `json:"maxduration"`
	MIMEs       []string `json:"mimes"`
}

type openrtbDev struct {
	UA         string     `json:"ua"`
	IP         string     `json:"ip"`
	OS         string     `json:"os"`
	DeviceType int32      `json:"devicetype"`
	Geo        *openrtbGeo `json:"geo,omitempty"`
}

type openrtbGeo struct {
	Country string `json:"country,omitempty"`
	City    string `json:"city,omitempty"`
}

type openrtbUsr struct {
	ID string `json:"id"`
}

type openrtbSite struct {
	ID      string       `json:"id"`
	Content *openrtbCont `json:"content,omitempty"`
}

type openrtbCont struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type openrtbResp struct {
	ID      string        `json:"id"`
	SeatBid []openrtbSeat `json:"seatbid"`
}

type openrtbSeat struct {
	Bid []openrtbBid `json:"bid"`
}

type openrtbBid struct {
	ID    string  `json:"id"`
	ImpID string  `json:"impid"`
	Price float64 `json:"price"`
	CrID  string  `json:"crid"`
	Adm   string  `json:"adm,omitempty"`
}

func (e *OpenRTBEncoder) Encode(ctx context.Context, spec *generator.BidRequestSpec) ([]byte, error) {
	isTest := 0
	if spec.IsTest {
		isTest = 1
	}

	req := openrtbReq{
		ID:     spec.RequestID,
		IsTest: isTest,
		Imp: []openrtbImp{{
			ID: spec.Imp.ImpID,
			Video: &openrtbVid{
				W:           spec.Imp.Width,
				H:           spec.Imp.Height,
				MinDuration: spec.Imp.MinDuration,
				MaxDuration: spec.Imp.MaxDuration,
				MIMEs:       []string{"video/mp4"},
			},
			BidFloor: spec.Imp.BidFloor,
		}},
		Device: openrtbDev{
			UA:         spec.Device.UA,
			IP:         spec.Device.IP,
			OS:         spec.Device.OS,
			DeviceType: deviceTypeToInt(spec.Device.DeviceType),
			Geo: &openrtbGeo{
				Country: spec.Device.GeoCountry,
				City:    spec.Device.GeoCity,
			},
		},
		User: openrtbUsr{ID: spec.UserID},
		Site: openrtbSite{
			ID: "1",
			Content: &openrtbCont{
				ID:    spec.Content.ContentID,
				Title: spec.Content.Title,
			},
		},
	}

	raw, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal openrtb: %w", err)
	}

	if e.gzip {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		if _, err := gw.Write(raw); err != nil {
			return nil, err
		}
		gw.Close()
		return buf.Bytes(), nil
	}

	return raw, nil
}

func (e *OpenRTBEncoder) Decode(raw []byte) (*BidResponseSpec, error) {
	var resp openrtbResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal openrtb response: %w", err)
	}

	spec := &BidResponseSpec{RequestID: resp.ID}
	for _, seat := range resp.SeatBid {
		for _, bid := range seat.Bid {
			spec.BidID = bid.ID
			spec.ImpID = bid.ImpID
			spec.Price = bid.Price
			spec.CrID = bid.CrID
			spec.Adm = bid.Adm
			break
		}
		break
	}
	return spec, nil
}

func deviceTypeToInt(dt string) int32 {
	switch dt {
	case "mobile":
		return 1
	case "pc":
		return 2
	case "ott":
		return 3
	default:
		return 1
	}
}
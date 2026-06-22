package adapter

import (
	"fmt"

	"github.com/google/uuid"
	iqiyipb "github.com/opendsp/opendsp/gen/platform/iqiyi"
	"google.golang.org/protobuf/proto"
)

type IqiyiAdapter struct{}

func (a *IqiyiAdapter) Name() string               { return "iqiyi" }
func (a *IqiyiAdapter) ContentType() string         { return "application/x-protobuf" }
func (a *IqiyiAdapter) ResponseContentType() string { return "application/x-protobuf" }

func (a *IqiyiAdapter) ParseRequest(raw []byte) (*UnifiedBidRequest, error) {
	iqiyiReq := &iqiyipb.BidRequest{}
	if err := proto.Unmarshal(raw, iqiyiReq); err != nil {
		return nil, fmt.Errorf("parse iqiyi protobuf: %w", err)
	}

	unified := &UnifiedBidRequest{
		RequestID: iqiyiReq.GetId(),
		MediaID:   "iqiyi",
		IsTest:    iqiyiReq.GetIsTest(),
		IsPing:    iqiyiReq.GetIsPing(),
	}

	if dev := iqiyiReq.GetDevice(); dev != nil {
		deviceType := "mobile"
		if pid := dev.GetPlatformId(); pid != "" && len(pid) > 0 {
			switch pid[0] {
			case '1':
				deviceType = "pc"
			case '5':
				deviceType = "ott"
			}
		}

		connType := dev.GetConnectionType()
		connName := ""
		switch connType {
		case 1:
			connName = "wifi"
		case 2, 3, 4:
			connName = "cellular"
		case 5:
			connName = "ethernet"
		}

		unified.Device = UnifiedDevice{
			OS:             dev.GetOs(),
			DeviceType:     deviceType,
			IP:             dev.GetIp(),
			UA:             dev.GetUa(),
			DeviceID:       coalesce(dev.GetIdfa(), dev.GetImei(), dev.GetOaid()),
			Make:           dev.GetModel(),
			Model:          dev.GetModel(),
			ScreenWidth:    dev.GetScreenWidth(),
			ScreenHeight:   dev.GetScreenHeight(),
			ConnectionType: connType,
			PlatformID:     dev.GetPlatformId(),
			OSVersion:      dev.GetOsVersion(),
		}
		if geo := dev.GetGeo(); geo != nil {
			unified.Device.GeoCity = geo.GetCity()
			unified.Device.GeoCountry = geo.GetCountry()
		}

		if connName != "" {
			unified.ConnectionType = connName
		}
	}

	if user := iqiyiReq.GetUser(); user != nil {
		unified.User = UnifiedUser{
			UserID:  user.GetId(),
			DMPIDs:  user.GetDmpId(),
			Feature: user.GetFeature(),
			Session: user.GetSession(),
		}
		unified.UserDMPIDs = user.GetDmpId()
		unified.UserFeature = user.GetFeature()
	}

	if site := iqiyiReq.GetSite(); site != nil {
		if content := site.GetContent(); content != nil {
			unified.Content = UnifiedContent{
				ContentID: coalesce(content.GetVideoClipId(), content.GetAlbumId()),
				Title:     content.GetTitle(),
				Category:  content.GetChannelId(),
				Tags:      content.GetTag(),
				Keywords:  content.GetKeyword(),
				Duration:  content.GetLen(),
				URL:       content.GetUrl(),
			}
		}
	}

	for _, imp := range iqiyiReq.GetImp() {
		ui := UnifiedImpression{
			ImpID:       imp.GetId(),
			BidFloor:    imp.GetBidfloor(),
			IsPMP:       imp.GetIsPmp(),
			BlockedAdTag: imp.GetBlockedAdTag(),
		}
		if cid := imp.GetCampaignId(); cid != "" {
			ui.CampaignID = cid
			if unified.CampaignID == "" {
				unified.CampaignID = cid
			}
		}

		if video := imp.GetVideo(); video != nil {
			ui.PositionType = a.convertAdType(video.GetAdType())
			ui.Width = video.GetW()
			ui.Height = video.GetH()
			ui.MinDuration = video.GetMinduration()
			ui.MaxDuration = video.GetMaxduration()
			ui.AdPositionID = video.GetAdZoneId()
		}
		if banner := imp.GetBanner(); banner != nil {
			if ui.AdPositionID == "" {
				ui.AdPositionID = banner.GetAdZoneId()
			}
			ui.CreativeTemplates = banner.GetCreativeTemplate()
		}
		unified.Imps = append(unified.Imps, ui)
	}

	return unified, nil
}

func (a *IqiyiAdapter) BuildResponse(req *UnifiedBidRequest, resp *UnifiedBidResponse) ([]byte, error) {
	iqiyiResp := &iqiyipb.BidResponse{
		Id:               proto.String(resp.RequestID),
		ProcessingTimeMs: proto.Int32(resp.ProcessingTimeMs),
	}

	for _, seat := range resp.SeatBids {
		seatBid := &iqiyipb.Seatbid{}
		for _, b := range seat.Bids {
			priceCents := int32(b.Price * 100)
			bid := &iqiyipb.Bid{
				Id:                     proto.String(uuid.New().String()),
				Impid:                  proto.String(b.ImpID),
				Price:                  proto.Int32(priceCents),
				Adm:                    proto.String(b.AdMarkup),
				IsPrecisionAdvertising: proto.Bool(true),
			}

			crID := b.PlatformCrID
			if crID == "" {
				crID = b.CreativeID
			}
			bid.Crid = proto.String(crID)

			if b.DeeplinkURL != "" {
				bid.DeeplinkUrl = proto.String(b.DeeplinkURL)
			}
			if b.DeeplinkApp != "" {
				bid.DeeplinkApp = proto.String(b.DeeplinkApp)
			}
			if b.StartDelay > 0 {
				bid.Startdelay = proto.Int32(b.StartDelay)
			}

			seatBid.Bid = append(seatBid.Bid, bid)
		}
		iqiyiResp.Seatbid = append(iqiyiResp.Seatbid, seatBid)
	}

	return proto.Marshal(iqiyiResp)
}

func (a *IqiyiAdapter) convertAdType(adType int32) int32 {
	switch adType {
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 3
	case 6:
		return 4
	case 10:
		return 5
	default:
		return 0
	}
}

func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

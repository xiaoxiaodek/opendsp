package encoder

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"

	iqiyipb "github.com/opendsp/opendsp/gen/platform/iqiyi"
	"github.com/opendsp/opendsp/internal/mockadx/generator"
	"google.golang.org/protobuf/proto"
)

type IqiyiEncoder struct {
	gzip bool
}

func NewIqiyiEncoder(gzip bool) *IqiyiEncoder {
	return &IqiyiEncoder{gzip: gzip}
}

func (e *IqiyiEncoder) ContentType() string {
	return "application/x-protobuf"
}

func (e *IqiyiEncoder) Encode(ctx context.Context, spec *generator.BidRequestSpec) ([]byte, error) {
	req := &iqiyipb.BidRequest{
		Id:     proto.String(spec.RequestID),
		IsTest: proto.Bool(spec.IsTest),
		IsPing: proto.Bool(false),
		Device: &iqiyipb.Device{
			Os:             proto.String(spec.Device.OS),
			Ip:             proto.String(spec.Device.IP),
			Ua:             proto.String(spec.Device.UA),
			Model:          proto.String(spec.Device.Make),
			ConnectionType: proto.Int32(1),
			PlatformId:     proto.String("33"),
			Geo: &iqiyipb.Geo{
				City:    proto.String(spec.Device.GeoCity),
				Country: proto.String("86"),
			},
		},
		User: &iqiyipb.User{
			Id: proto.String(spec.UserID),
		},
		Site: &iqiyipb.Site{
			Id: proto.String("1"),
			Content: &iqiyipb.Content{
				Title:     proto.String(spec.Content.Title),
				ChannelId: proto.String(spec.Content.Category),
				Len:       proto.Int32(spec.Content.Duration),
				Url:       proto.String(spec.Content.URL),
			},
		},
		Imp: []*iqiyipb.Impression{{
			Id:       proto.String(spec.Imp.ImpID),
			Bidfloor: proto.Float64(spec.Imp.BidFloor),
			Video: &iqiyipb.Video{
				AdType:      proto.Int32(convertPositionType(spec.Imp.PositionType)),
				Minduration: proto.Int32(spec.Imp.MinDuration),
				Maxduration: proto.Int32(spec.Imp.MaxDuration),
				W:           proto.Int32(spec.Imp.Width),
				H:           proto.Int32(spec.Imp.Height),
			},
		}},
	}

	raw, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal iqiyi: %w", err)
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

func (e *IqiyiEncoder) Decode(raw []byte) (*BidResponseSpec, error) {
	resp := &iqiyipb.BidResponse{}
	if err := proto.Unmarshal(raw, resp); err != nil {
		return nil, fmt.Errorf("unmarshal iqiyi response: %w", err)
	}

	spec := &BidResponseSpec{RequestID: resp.GetId()}
	for _, seat := range resp.GetSeatbid() {
		for _, bid := range seat.GetBid() {
			spec.BidID = bid.GetId()
			spec.ImpID = bid.GetImpid()
			spec.Price = float64(bid.GetPrice())
			spec.CrID = bid.GetCrid()
			spec.Adm = bid.GetAdm()
			break
		}
		break
	}
	return spec, nil
}

func convertPositionType(pt int32) int32 {
	switch pt {
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 10
	case 4:
		return 6
	case 5:
		return 3
	default:
		return 1
	}
}
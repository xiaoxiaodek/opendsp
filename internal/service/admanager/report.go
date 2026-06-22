package admanager

import (
	"context"
	"time"

	pb "github.com/opendsp/opendsp/gen/admanager/v1"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *AdManagerService) GetReport(ctx context.Context, req *pb.GetReportReq) (*pb.GetReportResp, error) {
	var campaignID, adGroupID *int64
	if req.CampaignId != nil {
		cid := *req.CampaignId
		campaignID = &cid
	}
	if req.AdGroupId != nil {
		agid := *req.AdGroupId
		adGroupID = &agid
	}

	reports, err := s.reportUC.Query(ctx, req.AdvertiserId, campaignID, adGroupID, req.StartTime.AsTime(), req.EndTime.AsTime())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query report: %v", err)
	}

	var pbReports []*pb.ReportHourly
	for _, r := range reports {
		ctr := 0.0
		if r.Impressions > 0 {
			ctr = float64(r.Clicks) / float64(r.Impressions) * 100
		}
		cpm := 0.0
		if r.Impressions > 0 {
			cpm = r.Cost / float64(r.Impressions) * 1000
		}
		pbReports = append(pbReports, &pb.ReportHourly{
			Hour:        timestamppb.New(r.Hour),
			Impressions: r.Impressions,
			Clicks:      r.Clicks,
			Conversions: r.Conversions,
			Cost:        r.Cost,
			Ctr:         ctr,
			Cpm:         cpm,
		})
	}

	return &pb.GetReportResp{Reports: pbReports}, nil
}

func (s *AdManagerService) GetDashboard(ctx context.Context, req *pb.GetDashboardReq) (*pb.Dashboard, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	reports, err := s.reportUC.Query(ctx, req.AdvertiserId, nil, nil, todayStart, now)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query dashboard: %v", err)
	}

	d := &pb.Dashboard{}
	for _, r := range reports {
		d.TodayImpressions += r.Impressions
		d.TodayClicks += r.Clicks
		d.TodayCost += r.Cost
	}
	if d.TodayImpressions > 0 {
		d.TodayCtr = float64(d.TodayClicks) / float64(d.TodayImpressions) * 100
	}

	var activeCampaigns, activeAdGroups int64
	s.data.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM campaign WHERE advertiser_id=$1 AND status=1`, req.AdvertiserId,
	).Scan(&activeCampaigns)
	s.data.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ad_group ag JOIN campaign c ON ag.campaign_id=c.id WHERE c.advertiser_id=$1 AND ag.status=1`, req.AdvertiserId,
	).Scan(&activeAdGroups)
	d.ActiveCampaigns = activeCampaigns
	d.ActiveAdGroups = activeAdGroups

	return d, nil
}

func (s *AdManagerService) ListMedia(ctx context.Context, _ *emptypb.Empty) (*pb.ListMediaResp, error) {
	rows, err := s.data.Queries.ListAllMedia(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list media: %v", err)
	}
	var media []*pb.Media
	for _, row := range rows {
		d := ""
		if row.Domain != nil {
			d = *row.Domain
		}
		media = append(media, &pb.Media{
			Id:     row.ID,
			Name:   row.Name,
			Code:   row.Code,
			Domain: d,
		})
	}
	return &pb.ListMediaResp{Media: media}, nil
}

func (s *AdManagerService) ListAdPositions(ctx context.Context, req *pb.ListAdPositionsReq) (*pb.ListAdPositionsResp, error) {
	rows, err := s.data.Queries.ListAdPositionsByMedia(ctx, req.MediaId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list ad positions: %v", err)
	}
	var positions []*pb.AdPosition
	for _, row := range rows {
		positions = append(positions, &pb.AdPosition{
			Id:           row.ID,
			MediaId:      row.MediaID,
			Name:         row.Name,
			PositionType: int32(row.PositionType),
			AdFormat:     int32(row.AdFormat),
			Width:        valInt32(row.Width),
			Height:       valInt32(row.Height),
			MaxSize:      valInt32(row.MaxSize),
			DurationMin:  valInt32(row.DurationMin),
			DurationMax:  valInt32(row.DurationMax),
		})
	}
	return &pb.ListAdPositionsResp{Positions: positions}, nil
}

func valInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func (s *AdManagerService) GetDashboardBreakdown(ctx context.Context, req *pb.GetDashboardBreakdownReq) (*pb.GetDashboardBreakdownResp, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	topN := req.TopN
	if topN <= 0 {
		topN = 10
	}
	if topN > 100 {
		topN = 100
	}

	rows, err := s.data.Queries.QueryDashboardBreakdown(ctx, &dbsqlc.QueryDashboardBreakdownParams{
		AdvertiserID: req.AdvertiserId,
		Hour:         todayStart,
		Hour_2:       now,
		Limit:        topN,
		Dimension:    req.Dimension,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query breakdown: %v", err)
	}

	var items []*pb.DimensionItem
	for _, row := range rows {
		id := int64(0)
		name := ""
		if v, ok := row.ID.(int64); ok {
			id = v
		}
		if v, ok := row.Name.(string); ok {
			name = v
		}
		ctr := 0.0
		if row.Impressions > 0 {
			ctr = float64(row.Clicks) / float64(row.Impressions) * 100
		}
		cpm := 0.0
		if row.Impressions > 0 {
			cpm = row.Cost / float64(row.Impressions) * 1000
		}
		items = append(items, &pb.DimensionItem{
			Id:          id,
			Name:        name,
			Impressions: row.Impressions,
			Clicks:      row.Clicks,
			Ctr:         ctr,
			Cost:        row.Cost,
			Cpm:         cpm,
		})
	}

	return &pb.GetDashboardBreakdownResp{Items: items}, nil
}

func (s *AdManagerService) GetEntityReport(ctx context.Context, req *pb.GetEntityReportReq) (*pb.GetEntityReportResp, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	startTime := req.StartTime.AsTime()
	endTime := req.EndTime.AsTime()

	todayRows, err := s.data.Queries.QueryEntityReport(ctx, &dbsqlc.QueryEntityReportParams{
		AdvertiserID: req.AdvertiserId,
		CampaignID:   req.EntityId,
		Dimension:    req.EntityType,
		Hour:         todayStart,
		Hour_2:       now,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query entity today: %v", err)
	}

	var todayImp, todayClicks int64
	var todayCost float64
	for _, row := range todayRows {
		todayImp += row.Impressions
		todayClicks += row.Clicks
		todayCost += row.Cost
	}
	todayCtr := 0.0
	if todayImp > 0 {
		todayCtr = float64(todayClicks) / float64(todayImp) * 100
	}

	hourlyRows, err := s.data.Queries.QueryEntityReport(ctx, &dbsqlc.QueryEntityReportParams{
		AdvertiserID: req.AdvertiserId,
		CampaignID:   req.EntityId,
		Dimension:    req.EntityType,
		Hour:         startTime,
		Hour_2:       endTime,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query entity hourly: %v", err)
	}

	var hourly []*pb.ReportHourly
	for _, row := range hourlyRows {
		ctr := 0.0
		if row.Impressions > 0 {
			ctr = float64(row.Clicks) / float64(row.Impressions) * 100
		}
		cpm := 0.0
		if row.Impressions > 0 {
			cpm = row.Cost / float64(row.Impressions) * 1000
		}
		hourly = append(hourly, &pb.ReportHourly{
			Hour:        timestamppb.New(row.Hour),
			Impressions: row.Impressions,
			Clicks:      row.Clicks,
			Cost:        row.Cost,
			Ctr:         ctr,
			Cpm:         cpm,
		})
	}

	var subItems []*pb.DimensionItem
	if req.EntityType == "campaign" || req.EntityType == "adgroup" {
		subRows, err := s.data.Queries.QueryEntitySubItems(ctx, &dbsqlc.QueryEntitySubItemsParams{
			AdvertiserID: req.AdvertiserId,
			CampaignID:   req.EntityId,
			Dimension:    req.EntityType,
			Hour:         todayStart,
			Hour_2:       now,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "query sub items: %v", err)
		}
		for _, row := range subRows {
			id := int64(0)
			name := ""
			if v, ok := row.ID.(int64); ok {
				id = v
			}
			if v, ok := row.Name.(string); ok {
				name = v
			}
			ctr := 0.0
			if row.Impressions > 0 {
				ctr = float64(row.Clicks) / float64(row.Impressions) * 100
			}
			cpm := 0.0
			if row.Impressions > 0 {
				cpm = row.Cost / float64(row.Impressions) * 1000
			}
			subItems = append(subItems, &pb.DimensionItem{
				Id:          id,
				Name:        name,
				Impressions: row.Impressions,
				Clicks:      row.Clicks,
				Ctr:         ctr,
				Cost:        row.Cost,
				Cpm:         cpm,
			})
		}
	}

	return &pb.GetEntityReportResp{
		TodayCost:        todayCost,
		TodayImpressions: todayImp,
		TodayClicks:      todayClicks,
		TodayCtr:         todayCtr,
		Hourly:           hourly,
		SubItems:         subItems,
	}, nil
}

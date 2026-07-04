package admanager

import (
	"context"

	"github.com/opendsp/opendsp/internal/biz"
	pb "github.com/opendsp/opendsp/gen/admanager/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *AdManagerService) CreateAdGroup(ctx context.Context, req *pb.CreateAdGroupReq) (*pb.AdGroup, error) {
	targeting := []byte(req.Targeting)
	if len(targeting) == 0 {
		targeting = []byte("{}")
	}
	ag := &biz.AdGroup{
		CampaignID:  req.CampaignId,
		Name:        req.Name,
		BidType:     int16(req.BidType),
		BidPrice:    req.BidPrice,
		DailyBudget: req.DailyBudget,
		FreqPeriod:  req.FreqPeriod,
		Targeting:   targeting,
	}
	if req.FreqCap != nil {
		fc := *req.FreqCap
		ag.FreqCap = &fc
	}

	if err := s.adGroupUC.Create(ctx, ag); err != nil {
		return nil, status.Errorf(codes.Internal, "create adgroup: %v", err)
	}
	return adGroupToProto(ag), nil
}

func (s *AdManagerService) UpdateAdGroup(ctx context.Context, req *pb.UpdateAdGroupReq) (*pb.AdGroup, error) {
	ag, err := s.adGroupUC.Get(ctx, req.Id)
	if err != nil || ag == nil {
		return nil, status.Errorf(codes.NotFound, "adgroup not found")
	}
	if req.Name != nil {
		ag.Name = *req.Name
	}
	if req.BidPrice != nil {
		ag.BidPrice = *req.BidPrice
	}
	if req.DailyBudget != nil {
		ag.DailyBudget = req.DailyBudget
	}
	if req.FreqCap != nil {
		fc := *req.FreqCap
		ag.FreqCap = &fc
	}
	if req.Targeting != nil {
		ag.Targeting = []byte(*req.Targeting)
	}

	if err := s.adGroupUC.Update(ctx, ag); err != nil {
		return nil, status.Errorf(codes.Internal, "update adgroup: %v", err)
	}
	return adGroupToProto(ag), nil
}

func (s *AdManagerService) GetAdGroup(ctx context.Context, req *pb.GetAdGroupReq) (*pb.AdGroup, error) {
	ag, err := s.adGroupUC.Get(ctx, req.Id)
	if err != nil || ag == nil {
		return nil, status.Errorf(codes.NotFound, "adgroup not found")
	}
	return adGroupToProto(ag), nil
}

func (s *AdManagerService) ListAdGroups(ctx context.Context, req *pb.ListAdGroupsReq) (*pb.ListAdGroupsResp, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	var statusFilter *int16
	if req.Status != nil {
		s := int16(*req.Status)
		statusFilter = &s
	}

	groups, total, err := s.adGroupUC.List(ctx, req.CampaignId, statusFilter, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list adgroups: %v", err)
	}

	var pbGroups []*pb.AdGroup
	for i := range groups {
		pbGroups = append(pbGroups, adGroupToProto(&groups[i]))
	}

	return &pb.ListAdGroupsResp{AdGroups: pbGroups, Total: total}, nil
}

func (s *AdManagerService) UpdateAdGroupStatus(ctx context.Context, req *pb.UpdateAdGroupStatusReq) (*pb.AdGroup, error) {
	var err error
	switch req.Status {
	case int32(biz.CampaignStatusActive):
		err = s.adGroupUC.Activate(ctx, req.Id)
	case int32(biz.CampaignStatusPaused):
		err = s.adGroupUC.Pause(ctx, req.Id)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid status: %d", req.Status)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update adgroup status: %v", err)
	}
	ag, _ := s.adGroupUC.Get(ctx, req.Id)
	return adGroupToProto(ag), nil
}

func adGroupToProto(ag *biz.AdGroup) *pb.AdGroup {
	p := &pb.AdGroup{
		Id:          ag.ID,
		CampaignId:  ag.CampaignID,
		Name:        ag.Name,
		BidType:     int32(ag.BidType),
		BidPrice:    ag.BidPrice,
		DailyBudget: ag.DailyBudget,
		FreqPeriod:  ag.FreqPeriod,
		Targeting:   string(ag.Targeting),
		Status:      int32(ag.Status),
		CreatedAt:   timestamppb.New(ag.CreatedAt),
		UpdatedAt:   timestamppb.New(ag.UpdatedAt),
	}
	if ag.FreqCap != nil {
		fc := *ag.FreqCap
		p.FreqCap = &fc
	}
	return p
}

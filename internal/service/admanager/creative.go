package admanager

import (
	"context"

	"github.com/opendsp/opendsp/internal/biz"
	pb "github.com/opendsp/opendsp/gen/admanager/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *AdManagerService) CreateCreative(ctx context.Context, req *pb.CreateCreativeReq) (*pb.Creative, error) {
	cr := &biz.Creative{
		AdGroupID:     req.AdGroupId,
		Name:          req.Name,
		CreativeType:  int16(req.CreativeType),
		AssetURL:      req.AssetUrl,
		AssetSize:     req.AssetSize,
		AssetDuration: req.AssetDuration,
		AssetWidth:    req.AssetWidth,
		AssetHeight:   req.AssetHeight,
		AssetMime:     req.AssetMime,
		Title:         req.Title,
		Description:   req.Description,
		CTAText:       req.CtaText,
		BrandName:     req.BrandName,
		BrandLogo:     req.BrandLogo,
		LandingURL:    req.LandingUrl,
		DeeplinkURL:   req.DeeplinkUrl,
		ImpTracker:    req.ImpTracker,
		ClickTracker:  req.ClickTracker,
	}

	if err := s.CreateCreativeWithAudit(ctx, cr); err != nil {
		return nil, status.Errorf(codes.Internal, "create creative: %v", err)
	}
	return creativeToProto(cr), nil
}

func (s *AdManagerService) ListCreatives(ctx context.Context, req *pb.ListCreativesReq) (*pb.ListCreativesResp, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	creatives, total, err := s.creativeUC.List(ctx, req.AdGroupId, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list creatives: %v", err)
	}

	var pbCreatives []*pb.Creative
	for i := range creatives {
		pbCreatives = append(pbCreatives, creativeToProto(&creatives[i]))
	}

	return &pb.ListCreativesResp{Creatives: pbCreatives, Total: total}, nil
}

func (s *AdManagerService) SubmitAudit(ctx context.Context, req *pb.SubmitAuditReq) (*pb.Creative, error) {
	if err := s.creativeUC.SubmitAudit(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "submit audit: %v", err)
	}
	return &pb.Creative{Id: req.Id, AuditStatus: int32(biz.AuditStatusPending)}, nil
}

func (s *AdManagerService) UpdateCreative(ctx context.Context, req *pb.UpdateCreativeReq) (*pb.Creative, error) {
	cr, err := s.creativeUC.Get(ctx, req.Id)
	if err != nil || cr == nil {
		return nil, status.Errorf(codes.NotFound, "creative not found")
	}
	if req.Name != nil {
		cr.Name = *req.Name
	}
	if req.AssetUrl != nil {
		cr.AssetURL = *req.AssetUrl
	}
	if req.AssetWidth != nil {
		cr.AssetWidth = *req.AssetWidth
	}
	if req.AssetHeight != nil {
		cr.AssetHeight = *req.AssetHeight
	}
	if req.AssetDuration != nil {
		cr.AssetDuration = *req.AssetDuration
	}
	if req.Title != nil {
		cr.Title = *req.Title
	}
	if req.Description != nil {
		cr.Description = *req.Description
	}
	if req.LandingUrl != nil {
		cr.LandingURL = *req.LandingUrl
	}
	if req.ImpTracker != nil {
		cr.ImpTracker = *req.ImpTracker
	}
	if req.ClickTracker != nil {
		cr.ClickTracker = *req.ClickTracker
	}

	if err := s.creativeUC.Update(ctx, cr); err != nil {
		return nil, status.Errorf(codes.Internal, "update creative: %v", err)
	}
	return creativeToProto(cr), nil
}

func creativeToProto(cr *biz.Creative) *pb.Creative {
	return &pb.Creative{
		Id:            cr.ID,
		AdGroupId:     cr.AdGroupID,
		Name:          cr.Name,
		CreativeType:  int32(cr.CreativeType),
		AssetUrl:      cr.AssetURL,
		AssetSize:     cr.AssetSize,
		AssetDuration: cr.AssetDuration,
		AssetWidth:    cr.AssetWidth,
		AssetHeight:   cr.AssetHeight,
		AssetMime:     cr.AssetMime,
		Title:         cr.Title,
		Description:   cr.Description,
		CtaText:       cr.CTAText,
		BrandName:     cr.BrandName,
		BrandLogo:     cr.BrandLogo,
		LandingUrl:    cr.LandingURL,
		DeeplinkUrl:   cr.DeeplinkURL,
		ImpTracker:    cr.ImpTracker,
		ClickTracker:  cr.ClickTracker,
		AuditStatus:   int32(cr.AuditStatus),
		AuditReason:   cr.AuditReason,
		CreatedAt:     timestamppb.New(cr.CreatedAt),
	}
}

package admanager

import (
	"context"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data"
	"github.com/opendsp/opendsp/internal/dmp"
	pb "github.com/opendsp/opendsp/gen/admanager/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AdManagerService struct {
	pb.UnimplementedAdManagerServer
	campaignUC      *biz.CampaignUseCase
	adGroupUC       *biz.AdGroupUseCase
	creativeUC      *biz.CreativeUseCase
	reportUC        *biz.ReportUseCase
	advertiserUC    *biz.AdvertiserUseCase
	proofMaterialUC *biz.ProofMaterialUseCase
	balanceUC       *biz.BalanceUseCase
	mediaUC         *biz.MediaUseCase
	adPositionUC    *biz.AdPositionUseCase
	adminUC         *biz.AdminUseCase
	dmpRepo         biz.DmpRepo
	tagStore        *dmp.TagStore
	resolver        *dmp.AudienceResolver
	lookalike       *dmp.LookalikeEngine
	data            *data.Data
}

func NewAdManagerService(
	campaignUC *biz.CampaignUseCase,
	adGroupUC *biz.AdGroupUseCase,
	creativeUC *biz.CreativeUseCase,
	reportUC *biz.ReportUseCase,
	d *data.Data,
) *AdManagerService {
	return &AdManagerService{
		campaignUC: campaignUC,
		adGroupUC:  adGroupUC,
		creativeUC: creativeUC,
		reportUC:   reportUC,
		data:       d,
	}
}

func NewAdManagerServiceFull(
	campaignUC *biz.CampaignUseCase,
	adGroupUC *biz.AdGroupUseCase,
	creativeUC *biz.CreativeUseCase,
	reportUC *biz.ReportUseCase,
	advertiserUC *biz.AdvertiserUseCase,
	proofMaterialUC *biz.ProofMaterialUseCase,
	balanceUC *biz.BalanceUseCase,
	mediaUC *biz.MediaUseCase,
	adPositionUC *biz.AdPositionUseCase,
	adminUC *biz.AdminUseCase,
	dmpRepo biz.DmpRepo,
	tagStore *dmp.TagStore,
	resolver *dmp.AudienceResolver,
	lookalike *dmp.LookalikeEngine,
	d *data.Data,
) *AdManagerService {
	return &AdManagerService{
		campaignUC:      campaignUC,
		adGroupUC:       adGroupUC,
		creativeUC:      creativeUC,
		reportUC:        reportUC,
		advertiserUC:    advertiserUC,
		proofMaterialUC: proofMaterialUC,
		balanceUC:       balanceUC,
		mediaUC:         mediaUC,
		adPositionUC:    adPositionUC,
		adminUC:         adminUC,
		dmpRepo:         dmpRepo,
		tagStore:        tagStore,
		resolver:        resolver,
		lookalike:       lookalike,
		data:            d,
	}
}

func (s *AdManagerService) CreateCampaign(ctx context.Context, req *pb.CreateCampaignReq) (*pb.Campaign, error) {
	c := &biz.Campaign{
		AdvertiserID: req.AdvertiserId,
		Name:         req.Name,
		Budget:       req.Budget,
		DailyBudget:  req.DailyBudget,
		Pacing:       int16(req.Pacing),
	}
	if req.StartTime != nil {
		t := req.StartTime.AsTime()
		c.StartTime = &t
	}
	if req.EndTime != nil {
		t := req.EndTime.AsTime()
		c.EndTime = &t
	}

	if err := s.campaignUC.Create(ctx, c); err != nil {
		return nil, status.Errorf(codes.Internal, "create campaign: %v", err)
	}
	return campaignToProto(c), nil
}

func (s *AdManagerService) UpdateCampaign(ctx context.Context, req *pb.UpdateCampaignReq) (*pb.Campaign, error) {
	c, err := s.campaignUC.Get(ctx, req.Id)
	if err != nil || c == nil {
		return nil, status.Errorf(codes.NotFound, "campaign not found")
	}
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.Budget != nil {
		c.Budget = req.Budget
	}
	if req.DailyBudget != nil {
		c.DailyBudget = req.DailyBudget
	}
	if req.StartTime != nil {
		t := req.StartTime.AsTime()
		c.StartTime = &t
	}
	if req.EndTime != nil {
		t := req.EndTime.AsTime()
		c.EndTime = &t
	}
	if req.Pacing != nil {
		c.Pacing = int16(*req.Pacing)
	}

	if err := s.campaignUC.Update(ctx, c); err != nil {
		return nil, status.Errorf(codes.Internal, "update campaign: %v", err)
	}
	return campaignToProto(c), nil
}

func (s *AdManagerService) GetCampaign(ctx context.Context, req *pb.GetCampaignReq) (*pb.Campaign, error) {
	c, err := s.campaignUC.Get(ctx, req.Id)
	if err != nil || c == nil {
		return nil, status.Errorf(codes.NotFound, "campaign not found")
	}
	return campaignToProto(c), nil
}

func (s *AdManagerService) ListCampaigns(ctx context.Context, req *pb.ListCampaignsReq) (*pb.ListCampaignsResp, error) {
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

	campaigns, total, err := s.campaignUC.List(ctx, req.AdvertiserId, statusFilter, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list campaigns: %v", err)
	}

	var pbCampaigns []*pb.Campaign
	for i := range campaigns {
		pbCampaigns = append(pbCampaigns, campaignToProto(&campaigns[i]))
	}

	return &pb.ListCampaignsResp{Campaigns: pbCampaigns, Total: total}, nil
}

func (s *AdManagerService) UpdateCampaignStatus(ctx context.Context, req *pb.UpdateCampaignStatusReq) (*pb.Campaign, error) {
	var err error
	switch req.Status {
	case int32(biz.CampaignStatusActive):
		err = s.campaignUC.Activate(ctx, req.Id)
	case int32(biz.CampaignStatusPaused):
		err = s.campaignUC.Pause(ctx, req.Id)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid status: %d", req.Status)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update campaign status: %v", err)
	}
	c, _ := s.campaignUC.Get(ctx, req.Id)
	return campaignToProto(c), nil
}

func campaignToProto(c *biz.Campaign) *pb.Campaign {
	p := &pb.Campaign{
		Id:           c.ID,
		AdvertiserId: c.AdvertiserID,
		Name:         c.Name,
		Budget:       c.Budget,
		DailyBudget:  c.DailyBudget,
		Pacing:       int32(c.Pacing),
		Status:       int32(c.Status),
		CreatedAt:    timestamppb.New(c.CreatedAt),
		UpdatedAt:    timestamppb.New(c.UpdatedAt),
	}
	if c.StartTime != nil {
		p.StartTime = timestamppb.New(*c.StartTime)
	}
	if c.EndTime != nil {
		p.EndTime = timestamppb.New(*c.EndTime)
	}
	return p
}

package admanager

import (
	"context"
	"encoding/json"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/dmp"
	pb "github.com/opendsp/opendsp/gen/admanager/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *AdManagerService) CreateTag(ctx context.Context, req *pb.CreateTagReq) (*pb.Tag, error) {
	advertiserID := getAdvertiserID(ctx)

	tag := &biz.DmpTag{
		AdvertiserID: advertiserID,
		Name:         req.Name,
		TagType:      int16(req.TagType),
		Source:       "upload",
		Status:       biz.TagStatusComputing,
	}

	id, err := s.dmpRepo.CreateTag(ctx, tag)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create tag: %v", err)
	}

	if len(req.DeviceIds) > 0 {
		go func() {
			deviceIDs := make([]uint32, len(req.DeviceIds))
			for i, did := range req.DeviceIds {
				deviceIDs[i] = dmp.HashDeviceID(did)
			}
			if err := s.tagStore.AddDevices(id, deviceIDs); err == nil {
				count := int64(len(deviceIDs))
				s.dmpRepo.UpdateTagDeviceCount(context.Background(), id, count, biz.TagStatusReady)
			}
		}()
	}

	return &pb.Tag{
		Id:           id,
		AdvertiserId: advertiserID,
		Name:         req.Name,
		TagType:      req.TagType,
		Status:       int32(biz.TagStatusComputing),
		CreatedAt:    timestamppb.Now(),
	}, nil
}

func (s *AdManagerService) ListTags(ctx context.Context, req *pb.ListTagsReq) (*pb.ListTagsResp, error) {
	advertiserID := getAdvertiserID(ctx)
	var tagType *int16
	if req.TagType != nil {
		v := int16(*req.TagType)
		tagType = &v
	}

	tags, err := s.dmpRepo.ListTags(ctx, advertiserID, tagType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list tags: %v", err)
	}

	var pbTags []*pb.Tag
	for _, t := range tags {
		pbTags = append(pbTags, &pb.Tag{
			Id:           t.ID,
			AdvertiserId: t.AdvertiserID,
			Name:         t.Name,
			TagType:      int32(t.TagType),
			DeviceCount:  t.DeviceCount,
			Source:       t.Source,
			Status:       int32(t.Status),
			CreatedAt:    timestamppb.New(t.CreatedAt),
		})
	}
	return &pb.ListTagsResp{Tags: pbTags}, nil
}

func (s *AdManagerService) DeleteTag(ctx context.Context, req *pb.DeleteTagReq) (*emptypb.Empty, error) {
	if err := s.dmpRepo.DeleteTag(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "delete tag: %v", err)
	}
	s.tagStore.Invalidate(req.Id)
	return &emptypb.Empty{}, nil
}

func (s *AdManagerService) CreateAudience(ctx context.Context, req *pb.CreateAudienceReq) (*pb.Audience, error) {
	advertiserID := getAdvertiserID(ctx)

	audience := &biz.DmpAudience{
		AdvertiserID: advertiserID,
		Name:         req.Name,
		AudienceType: int16(req.AudienceType),
		Rules:        json.RawMessage(req.Rules),
		Status:       biz.AudienceStatusComputing,
	}

	id, err := s.dmpRepo.CreateAudience(ctx, audience)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create audience: %v", err)
	}

	go func() {
		bm, err := s.resolver.Resolve(id, json.RawMessage(req.Rules))
		if err != nil {
			s.dmpRepo.UpdateAudienceDeviceCount(context.Background(), id, 0, biz.AudienceStatusInvalid)
			return
		}
		s.dmpRepo.UpdateAudienceDeviceCount(context.Background(), id, int64(bm.GetCardinality()), biz.AudienceStatusReady)
	}()

	return &pb.Audience{
		Id:           id,
		AdvertiserId: advertiserID,
		Name:         req.Name,
		AudienceType: req.AudienceType,
		Rules:        req.Rules,
		Status:       int32(biz.AudienceStatusComputing),
		CreatedAt:    timestamppb.Now(),
	}, nil
}

func (s *AdManagerService) ListAudiences(ctx context.Context, req *pb.ListAudiencesReq) (*pb.ListAudiencesResp, error) {
	advertiserID := getAdvertiserID(ctx)
	var audienceType *int16
	if req.AudienceType != nil {
		v := int16(*req.AudienceType)
		audienceType = &v
	}

	audiences, err := s.dmpRepo.ListAudiences(ctx, advertiserID, audienceType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list audiences: %v", err)
	}

	var pbAudiences []*pb.Audience
	for _, a := range audiences {
		pbAudiences = append(pbAudiences, &pb.Audience{
			Id:           a.ID,
			AdvertiserId: a.AdvertiserID,
			Name:         a.Name,
			AudienceType: int32(a.AudienceType),
			Rules:        string(a.Rules),
			DeviceCount:  a.DeviceCount,
			Status:       int32(a.Status),
			CreatedAt:    timestamppb.New(a.CreatedAt),
		})
	}
	return &pb.ListAudiencesResp{Audiences: pbAudiences}, nil
}

func (s *AdManagerService) GetAudience(ctx context.Context, req *pb.GetAudienceReq) (*pb.Audience, error) {
	a, err := s.dmpRepo.GetAudience(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "audience not found")
	}
	return &pb.Audience{
		Id:           a.ID,
		AdvertiserId: a.AdvertiserID,
		Name:         a.Name,
		AudienceType: int32(a.AudienceType),
		Rules:        string(a.Rules),
		DeviceCount:  a.DeviceCount,
		Status:       int32(a.Status),
		CreatedAt:    timestamppb.New(a.CreatedAt),
	}, nil
}

func (s *AdManagerService) DeleteAudience(ctx context.Context, req *pb.DeleteAudienceReq) (*emptypb.Empty, error) {
	if err := s.dmpRepo.DeleteAudience(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "delete audience: %v", err)
	}
	s.resolver.InvalidateCache(req.Id)
	return &emptypb.Empty{}, nil
}

func (s *AdManagerService) CreateLookalike(ctx context.Context, req *pb.CreateLookalikeReq) (*pb.LookalikeTask, error) {
	tagID, err := s.lookalike.Run(ctx, req.SeedAudienceId, req.ExpansionFactor)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create lookalike: %v", err)
	}
	return &pb.LookalikeTask{
		Id:             tagID,
		SeedAudienceId: req.SeedAudienceId,
		ResultTagId:    tagID,
		Status:         1,
		CreatedAt:      timestamppb.Now(),
	}, nil
}

func getAdvertiserID(ctx context.Context) int64 {
	if v, ok := ctx.Value("advertiser_id").(int64); ok {
		return v
	}
	return 0
}

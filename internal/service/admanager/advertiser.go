package admanager

import (
	"context"

	"github.com/opendsp/opendsp/internal/biz"
	pb "github.com/opendsp/opendsp/gen/admanager/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *AdManagerService) CreateAdvertiser(ctx context.Context, req *pb.CreateAdvertiserReq) (*pb.Advertiser, error) {
	a := &biz.Advertiser{
		Name:         req.Name,
		Industry:     req.Industry,
		ContactName:  req.ContactName,
		ContactEmail: req.ContactEmail,
		Address:      req.Address,
		Website:      req.Website,
		BrandNames:   req.BrandNames,
	}
	if err := s.advertiserUC.Create(ctx, a); err != nil {
		return nil, status.Errorf(codes.Internal, "create advertiser: %v", err)
	}
	return advertiserToProto(a), nil
}

func (s *AdManagerService) ListAdvertisers(ctx context.Context, req *pb.ListAdvertisersReq) (*pb.ListAdvertisersResp, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	var statusFilter, qualFilter *int16
	if req.Status != nil {
		s := int16(*req.Status)
		statusFilter = &s
	}
	if req.QualificationStatus != nil {
		q := int16(*req.QualificationStatus)
		qualFilter = &q
	}

	advertisers, total, err := s.advertiserUC.List(ctx, statusFilter, qualFilter, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list advertisers: %v", err)
	}

	var pbAdvs []*pb.Advertiser
	for i := range advertisers {
		pbAdvs = append(pbAdvs, advertiserToProto(&advertisers[i]))
	}
	return &pb.ListAdvertisersResp{Advertisers: pbAdvs, Total: total}, nil
}

func (s *AdManagerService) GetAdvertiser(ctx context.Context, req *pb.GetAdvertiserReq) (*pb.Advertiser, error) {
	a, err := s.advertiserUC.Get(ctx, req.Id)
	if err != nil || a == nil {
		return nil, status.Errorf(codes.NotFound, "advertiser not found")
	}
	return advertiserToProto(a), nil
}

func (s *AdManagerService) UpdateAdvertiser(ctx context.Context, req *pb.UpdateAdvertiserReq) (*pb.Advertiser, error) {
	a, err := s.advertiserUC.Get(ctx, req.Id)
	if err != nil || a == nil {
		return nil, status.Errorf(codes.NotFound, "advertiser not found")
	}
	if req.Name != nil {
		a.Name = *req.Name
	}
	if req.Industry != nil {
		a.Industry = req.Industry
	}
	if req.ContactName != nil {
		a.ContactName = req.ContactName
	}
	if req.ContactEmail != nil {
		a.ContactEmail = req.ContactEmail
	}
	if req.Address != nil {
		a.Address = req.Address
	}
	if req.Website != nil {
		a.Website = req.Website
	}
	if req.BrandNames != nil {
		a.BrandNames = req.BrandNames
	}
	if err := s.advertiserUC.Update(ctx, a); err != nil {
		return nil, status.Errorf(codes.Internal, "update advertiser: %v", err)
	}
	return advertiserToProto(a), nil
}

func (s *AdManagerService) SubmitQualification(ctx context.Context, req *pb.SubmitQualificationReq) (*pb.Advertiser, error) {
	if err := s.advertiserUC.SubmitQualification(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "submit qualification: %v", err)
	}
	a, _ := s.advertiserUC.Get(ctx, req.Id)
	return advertiserToProto(a), nil
}

func (s *AdManagerService) AuditAdvertiser(ctx context.Context, req *pb.AuditAdvertiserReq) (*pb.Advertiser, error) {
	if err := s.advertiserUC.Audit(ctx, req.Id, int16(req.QualificationStatus), req.QualificationReason); err != nil {
		return nil, status.Errorf(codes.Internal, "audit advertiser: %v", err)
	}
	a, _ := s.advertiserUC.Get(ctx, req.Id)
	return advertiserToProto(a), nil
}

func (s *AdManagerService) DeleteAdvertiser(ctx context.Context, req *pb.DeleteAdvertiserReq) (*emptypb.Empty, error) {
	if err := s.advertiserUC.Delete(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "delete advertiser: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *AdManagerService) UploadProofMaterial(ctx context.Context, req *pb.UploadProofMaterialReq) (*pb.ProofMaterial, error) {
	m := &biz.ProofMaterial{
		AdvertiserID: req.AdvertiserId,
		MaterialType: int16(req.MaterialType),
		FileURL:      req.FileUrl,
		FileName:     &req.FileName,
		FileSize:     &req.FileSize,
	}
	if err := s.proofMaterialUC.Upload(ctx, m); err != nil {
		return nil, status.Errorf(codes.Internal, "upload proof: %v", err)
	}
	return &pb.ProofMaterial{
		AdvertiserId: m.AdvertiserID,
		MaterialType: int32(m.MaterialType),
		FileUrl:      m.FileURL,
	}, nil
}

func (s *AdManagerService) ListProofMaterials(ctx context.Context, req *pb.ListProofMaterialsReq) (*pb.ListProofMaterialsResp, error) {
	materials, err := s.proofMaterialUC.List(ctx, req.AdvertiserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list proofs: %v", err)
	}
	var pbMats []*pb.ProofMaterial
	for i := range materials {
		m := &materials[i]
		pbMats = append(pbMats, &pb.ProofMaterial{
			Id:           m.ID,
			AdvertiserId: m.AdvertiserID,
			MaterialType: int32(m.MaterialType),
			FileUrl:      m.FileURL,
			FileName:     ptrStrPb(m.FileName),
			FileSize:     ptrInt32Pb(m.FileSize),
			AuditStatus:  int32(m.AuditStatus),
			AuditReason:  ptrStrPb(m.AuditReason),
			CreatedAt:    timestamppb.New(m.CreatedAt),
		})
	}
	return &pb.ListProofMaterialsResp{Materials: pbMats}, nil
}

func (s *AdManagerService) GetBalance(ctx context.Context, req *pb.GetBalanceReq) (*pb.Balance, error) {
	balance, creditLimit, err := s.balanceUC.Get(ctx, req.AdvertiserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get balance: %v", err)
	}
	return &pb.Balance{
		AdvertiserId: req.AdvertiserId,
		Balance:      balance,
		CreditLimit:  creditLimit,
	}, nil
}

func (s *AdManagerService) Recharge(ctx context.Context, req *pb.RechargeReq) (*pb.Balance, error) {
	_, err := s.balanceUC.Recharge(ctx, req.AdvertiserId, req.Amount, req.Description, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "recharge: %v", err)
	}
	balance, creditLimit, _ := s.balanceUC.Get(ctx, req.AdvertiserId)
	return &pb.Balance{
		AdvertiserId: req.AdvertiserId,
		Balance:      balance,
		CreditLimit:  creditLimit,
	}, nil
}

func (s *AdManagerService) ListTransactions(ctx context.Context, req *pb.ListTransactionsReq) (*pb.ListTransactionsResp, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	txs, total, err := s.balanceUC.ListTransactions(ctx, req.AdvertiserId, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list transactions: %v", err)
	}
	var pbTxs []*pb.BalanceTransaction
	for i := range txs {
		tx := &txs[i]
		pbTxs = append(pbTxs, &pb.BalanceTransaction{
			Id:            tx.ID,
			AdvertiserId:  tx.AdvertiserID,
			Amount:        tx.Amount,
			BalanceBefore: tx.BalanceBefore,
			BalanceAfter:  tx.BalanceAfter,
			TxType:        int32(tx.TxType),
			Description:   ptrStrPb(tx.Description),
			CreatedAt:     timestamppb.New(tx.CreatedAt),
		})
	}
	return &pb.ListTransactionsResp{Transactions: pbTxs, Total: total}, nil
}

func (s *AdManagerService) CreateMedia(ctx context.Context, req *pb.CreateMediaReq) (*pb.Media, error) {
	id, err := s.mediaUC.Create(ctx, req.Name, req.Code, req.Domain)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create media: %v", err)
	}
	return &pb.Media{Id: id, Name: req.Name, Code: req.Code, Domain: req.Domain}, nil
}

func (s *AdManagerService) UpdateMedia(ctx context.Context, req *pb.UpdateMediaReq) (*pb.Media, error) {
	if err := s.mediaUC.Update(ctx, req.Id, req.Name, req.Domain); err != nil {
		return nil, status.Errorf(codes.Internal, "update media: %v", err)
	}
	return &pb.Media{Id: req.Id}, nil
}

func (s *AdManagerService) UpdateMediaStatus(ctx context.Context, req *pb.UpdateMediaStatusReq) (*pb.Media, error) {
	if err := s.mediaUC.UpdateStatus(ctx, req.Id, int16(req.Status)); err != nil {
		return nil, status.Errorf(codes.Internal, "update media status: %v", err)
	}
	return &pb.Media{Id: req.Id}, nil
}

func (s *AdManagerService) CreateAdPosition(ctx context.Context, req *pb.CreateAdPositionReq) (*pb.AdPosition, error) {
	id, err := s.adPositionUC.Create(ctx, req.MediaId, req.Name, int16(req.PositionType), int16(req.AdFormat),
		req.Width, req.Height, req.MaxSize, req.DurationMin, req.DurationMax, req.MimeTypes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create ad position: %v", err)
	}
	return &pb.AdPosition{Id: id, MediaId: req.MediaId, Name: req.Name}, nil
}

func (s *AdManagerService) UpdateAdPosition(ctx context.Context, req *pb.UpdateAdPositionReq) (*pb.AdPosition, error) {
	if err := s.adPositionUC.Update(ctx, req.Id, req.Name, req.Width, req.Height, req.MaxSize, req.DurationMin, req.DurationMax); err != nil {
		return nil, status.Errorf(codes.Internal, "update ad position: %v", err)
	}
	return &pb.AdPosition{Id: req.Id}, nil
}

func (s *AdManagerService) ListUsers(ctx context.Context, req *pb.ListUsersReq) (*pb.ListUsersResp, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	users, total, err := s.adminUC.ListUsers(ctx, req.Role, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list users: %v", err)
	}
	var pbUsers []*pb.User
	for i := range users {
		u := &users[i]
		pbUsers = append(pbUsers, &pb.User{
			Id:    u.ID,
			Email: u.Email,
			Name:  ptrStrPb(u.Name),
			Role:  u.Role,
			CreatedAt: timestamppb.New(u.CreatedAt),
		})
	}
	return &pb.ListUsersResp{Users: pbUsers, Total: total}, nil
}

func (s *AdManagerService) UpdateUserRole(ctx context.Context, req *pb.UpdateUserRoleReq) (*pb.User, error) {
	if err := s.adminUC.UpdateUserRole(ctx, req.Id, req.Role); err != nil {
		return nil, status.Errorf(codes.Internal, "update user role: %v", err)
	}
	return &pb.User{Id: req.Id, Role: req.Role}, nil
}

func (s *AdManagerService) CreateUser(ctx context.Context, req *pb.CreateUserReq) (*pb.User, error) {
	if req.Email == "" || req.Password == "" || len(req.Password) < 6 {
		return nil, status.Errorf(codes.InvalidArgument, "email and password (min 6 chars) required")
	}
	id, err := s.adminUC.CreateUser(ctx, req.Email, hashPassword(req.Password), req.Name, req.AdvertiserId, req.Role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create user: %v", err)
	}
	return &pb.User{Id: id, Email: req.Email, Name: req.Name, Role: req.Role}, nil
}

func (s *AdManagerService) ListPendingAudits(ctx context.Context, req *pb.ListPendingAuditsReq) (*pb.ListPendingAuditsResp, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	audits, total, err := s.adminUC.ListPendingAudits(ctx, req.AuditType, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list pending audits: %v", err)
	}
	var pbAudits []*pb.PendingAudit
	for i := range audits {
		a := &audits[i]
		pbAudits = append(pbAudits, &pb.PendingAudit{
			Id:             a.ID,
			AuditType:      a.AuditType,
			Name:           a.Name,
			AdvertiserId:   a.AdvertiserID,
			AdvertiserName: a.AdvertiserName,
			Status:         int32(a.Status),
			Reason:         ptrStrPb(a.Reason),
			CreatedAt:      timestamppb.New(a.CreatedAt),
		})
	}
	return &pb.ListPendingAuditsResp{Audits: pbAudits, Total: total}, nil
}

func (s *AdManagerService) AuditCreative(ctx context.Context, req *pb.AuditCreativeReq) (*pb.Creative, error) {
	switch req.AuditStatus {
	case int32(biz.AuditStatusApproved):
		if err := s.creativeUC.Approve(ctx, req.Id); err != nil {
			return nil, status.Errorf(codes.Internal, "approve creative: %v", err)
		}
	case int32(biz.AuditStatusRejected):
		if err := s.creativeUC.Reject(ctx, req.Id, req.AuditReason); err != nil {
			return nil, status.Errorf(codes.Internal, "reject creative: %v", err)
		}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid audit status")
	}
	return &pb.Creative{Id: req.Id, AuditStatus: req.AuditStatus, AuditReason: req.AuditReason}, nil
}

func advertiserToProto(a *biz.Advertiser) *pb.Advertiser {
	return &pb.Advertiser{
		Id:                  a.ID,
		Name:                a.Name,
		Industry:            ptrStrPb(a.Industry),
		ContactName:         ptrStrPb(a.ContactName),
		ContactEmail:        ptrStrPb(a.ContactEmail),
		Balance:             a.Balance,
		Status:              int32(a.Status),
		QualificationStatus: int32(a.QualificationStatus),
		QualificationReason: ptrStrPb(a.QualificationReason),
		CreditLimit:         a.CreditLimit,
		Address:             ptrStrPb(a.Address),
		Website:             ptrStrPb(a.Website),
		BrandNames:          ptrStrPb(a.BrandNames),
		CreatedAt:           timestamppb.New(a.CreatedAt),
		UpdatedAt:           timestamppb.New(a.UpdatedAt),
	}
}

func ptrStrPb(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrInt32Pb(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

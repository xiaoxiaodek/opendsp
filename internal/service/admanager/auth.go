package admanager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	pb "github.com/opendsp/opendsp/gen/admanager/v1"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
	"github.com/opendsp/opendsp/internal/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func hashPassword(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}

func (s *AdManagerService) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginResp, error) {
	row, err := s.data.Queries.GetUserByEmail(ctx, req.Email)
	if err != nil || row.PasswordHash != hashPassword(req.Password) {
		return nil, status.Errorf(codes.Unauthenticated, "invalid email or password")
	}

	role := ""
	if row.Role != nil {
		role = *row.Role
	}
	name := ""
	if row.Name != nil {
		name = *row.Name
	}

	token, err := middleware.GenerateToken(row.ID, row.AdvertiserID, role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "token generation failed")
	}

	return &pb.LoginResp{
		Token:        token,
		UserId:       row.ID,
		Email:        row.Email,
		Name:         name,
		AdvertiserId: row.AdvertiserID,
		Role:         role,
	}, nil
}

func (s *AdManagerService) Register(ctx context.Context, req *pb.RegisterReq) (*pb.LoginResp, error) {
	if req.Email == "" || req.Password == "" || len(req.Password) < 6 {
		return nil, status.Errorf(codes.InvalidArgument, "email and password (min 6 chars) required")
	}

	// Check existing user
	existing, _ := s.data.Queries.GetUserByEmail(ctx, req.Email)
	if existing != nil && existing.ID > 0 {
		return nil, status.Errorf(codes.AlreadyExists, "email already registered")
	}

	// Create advertiser
	_ = s.data.Queries.CreateAdvertiserSimple(ctx, req.Name)

	advertiserID, _ := s.data.Queries.GetAdvertiserByName(ctx, req.Name)

	// Create user
	role := "admin"
	name := &req.Name
	id, err := s.data.Queries.CreateUser(ctx, &dbsqlc.CreateUserParams{
		Email:        req.Email,
		PasswordHash: hashPassword(req.Password),
		Name:         name,
		AdvertiserID: &advertiserID,
		Role:         &role,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create user failed")
	}

	token, err := middleware.GenerateToken(id, advertiserID, "admin")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "token generation failed")
	}

	return &pb.LoginResp{
		Token:        token,
		UserId:       id,
		Email:        req.Email,
		Name:         req.Name,
		AdvertiserId: advertiserID,
		Role:         "admin",
	}, nil
}

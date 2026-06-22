package admanager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/opendsp/opendsp/internal/middleware"
	pb "github.com/opendsp/opendsp/gen/admanager/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func hashPassword(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}

func (s *AdManagerService) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginResp, error) {
	var id, advertiserID int64
	var email, name, role, passwordHash string

	err := s.data.Pool.QueryRow(ctx,
		`SELECT id, email, name, COALESCE(advertiser_id, 0), role, password_hash FROM users WHERE email = $1`,
		req.Email,
	).Scan(&id, &email, &name, &advertiserID, &role, &passwordHash)

	if err != nil || passwordHash != hashPassword(req.Password) {
		return nil, status.Errorf(codes.Unauthenticated, "invalid email or password")
	}

	token, err := middleware.GenerateToken(id, advertiserID, role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "token generation failed")
	}

	return &pb.LoginResp{
		Token:        token,
		UserId:       id,
		Email:        email,
		Name:         name,
		AdvertiserId: advertiserID,
		Role:         role,
	}, nil
}

func (s *AdManagerService) Register(ctx context.Context, req *pb.RegisterReq) (*pb.LoginResp, error) {
	if req.Email == "" || req.Password == "" || len(req.Password) < 6 {
		return nil, status.Errorf(codes.InvalidArgument, "email and password (min 6 chars) required")
	}

	var existingID int64
	s.data.Pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, req.Email).Scan(&existingID)
	if existingID > 0 {
		return nil, status.Errorf(codes.AlreadyExists, "email already registered")
	}

	_, err := s.data.Pool.Exec(ctx, `INSERT INTO advertiser (name) VALUES ($1)`, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create advertiser failed")
	}

	var advertiserID int64
	s.data.Pool.QueryRow(ctx, `SELECT id FROM advertiser WHERE name = $1 ORDER BY id DESC LIMIT 1`, req.Name).Scan(&advertiserID)

	var id int64
	err = s.data.Pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, advertiser_id, role) VALUES ($1, $2, $3, $4, 'admin') RETURNING id`,
		req.Email, hashPassword(req.Password), req.Name, advertiserID,
	).Scan(&id)
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

package filegateway

import (
	"context"
	"crypto/rand"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/opendsp/opendsp/internal/data"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
	"github.com/opendsp/opendsp/internal/storage"
	pb "github.com/opendsp/opendsp/gen/filegateway/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	uploadURLTTL = 5 * time.Minute
	nanoidLen    = 20
)

var namespaceBuckets = map[string]string{
	"creative": "opendsp-creatives",
	"proof":    "opendsp-proofs",
	"asset":    "opendsp-assets",
}

var namespaceChars = map[string]string{
	"creative": "c",
	"proof":    "p",
	"asset":    "a",
}

type Service struct {
	pb.UnimplementedFileGatewayServer
	backend storage.StorageBackend
	data    *data.Data
}

func NewService(backend storage.StorageBackend, d *data.Data) *Service {
	return &Service{backend: backend, data: d}
}

func (s *Service) CreateUploadURL(ctx context.Context, req *pb.CreateUploadURLReq) (*pb.CreateUploadURLResp, error) {
	ns := req.Namespace
	bucket, ok := namespaceBuckets[ns]
	if !ok {
		return nil, fmt.Errorf("unknown namespace: %s", ns)
	}
	nsChar, ok := namespaceChars[ns]
	if !ok {
		return nil, fmt.Errorf("unknown namespace: %s", ns)
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	if ext == "" {
		ext = ".bin"
	}

	fileID := nsChar + generateNanoid(nanoidLen) + ext
	storageKey := fmt.Sprintf("%s/%s/%s", bucket, fileID[:2], fileID)

	status := int16(1)
	filename := req.Filename
	err := s.data.Queries.InsertFileRecord(ctx, &dbsqlc.InsertFileRecordParams{
		ID:          fileID,
		Namespace:   ns,
		StorageKey:  storageKey,
		Filename:    &filename,
		Size:        &req.FileSize,
		ContentType: &req.ContentType,
		Status:      &status,
	})
	if err != nil {
		return nil, fmt.Errorf("insert file record: %w", err)
	}

	uploadURL, err := s.backend.PresignedPutURL(ctx, bucket, storageKey, uploadURLTTL)
	if err != nil {
		return nil, fmt.Errorf("presigned URL: %w", err)
	}

	return &pb.CreateUploadURLResp{
		FileId:     fileID,
		UploadUrl:  uploadURL,
		PublicPath: fmt.Sprintf("/%s/%s", ns, fileID),
	}, nil
}

func (s *Service) GetFileInfo(ctx context.Context, req *pb.GetFileInfoReq) (*pb.FileInfo, error) {
	rec, err := s.data.Queries.GetFileRecord(ctx, req.FileId)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", req.FileId)
	}
	return &pb.FileInfo{
		FileId:      rec.ID,
		Namespace:   rec.Namespace,
		Filename:    ptrStr(rec.Filename),
		Size:        ptrInt64(rec.Size),
		ContentType: ptrStr(rec.ContentType),
		Md5:         ptrStr(rec.Md5),
		Status:      ptrInt16(rec.Status),
		CreatedAt:   timestamppb.New(rec.CreatedAt.Time),
	}, nil
}

func (s *Service) DeleteFile(ctx context.Context, req *pb.DeleteFileReq) (*pb.DeleteFileResp, error) {
	rec, err := s.data.Queries.GetFileRecord(ctx, req.FileId)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", req.FileId)
	}
	if err := s.data.Queries.DeleteFileRecord(ctx, req.FileId); err != nil {
		return nil, fmt.Errorf("delete file record: %w", err)
	}
	_ = s.backend.DeleteObject(ctx, namespaceBuckets[rec.Namespace], rec.StorageKey)
	return &pb.DeleteFileResp{Success: true}, nil
}

func generateNanoid(n int) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func ptrInt16(v *int16) int32 {
	if v == nil {
		return 0
	}
	return int32(*v)
}

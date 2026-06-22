package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalBackend struct {
	dataDir string
}

func NewLocalBackend(dataDir string) (*LocalBackend, error) {
	for _, dir := range []string{"creatives", "proofs", "assets"} {
		if err := os.MkdirAll(filepath.Join(dataDir, dir), 0755); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", dir, err)
		}
	}
	return &LocalBackend{dataDir: dataDir}, nil
}

func (b *LocalBackend) PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error {
	path := filepath.Join(b.dataDir, key)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdirall %s: %w", filepath.Dir(path), err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file %s: %w", path, err)
	}
	defer f.Close()
	if _, err = io.Copy(f, r); err != nil {
		return fmt.Errorf("copy to %s: %w", path, err)
	}
	return nil
}

func (b *LocalBackend) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, *ObjectInfo, error) {
	path := filepath.Join(b.dataDir, key)
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, fmt.Errorf("stat %s: %w", path, err)
	}
	return f, &ObjectInfo{
		ContentLength: stat.Size(),
		ContentType:   detectContentType(filepath.Ext(key)),
	}, nil
}

func (b *LocalBackend) DeleteObject(ctx context.Context, bucket, key string) error {
	return os.Remove(filepath.Join(b.dataDir, key))
}

func (b *LocalBackend) PresignedPutURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	return "", fmt.Errorf("presigned URLs not supported in local backend")
}

func detectContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".flv":
		return "video/x-flv"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

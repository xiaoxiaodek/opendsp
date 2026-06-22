package storage

import (
	"context"
	"io"
	"time"
)

type StorageBackend interface {
	PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, *ObjectInfo, error)
	DeleteObject(ctx context.Context, bucket, key string) error
	PresignedPutURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
}

type ObjectInfo struct {
	ContentType   string
	ContentLength int64
	ETag          string
}

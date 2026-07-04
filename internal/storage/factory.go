package storage

import (
	"context"
	"fmt"

	"github.com/opendsp/opendsp/internal/config"
)

func NewFromConfig(ctx context.Context, cfg config.StorageConfig) (StorageBackend, error) {
	switch cfg.Backend {
	case "s3":
		return NewS3Backend(ctx, cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, cfg.Region)
	case "local":
		return NewLocalBackend(cfg.LocalDir)
	default:
		return nil, fmt.Errorf("unknown storage backend: %s", cfg.Backend)
	}
}

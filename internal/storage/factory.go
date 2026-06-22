package storage

import (
	"context"
	"fmt"
	"os"
)

type Config struct {
	Backend   string
	Endpoint  string
	AccessKey string
	SecretKey string
	Region    string
	LocalDir  string
}

func NewFromConfig(ctx context.Context, cfg Config) (StorageBackend, error) {
	switch cfg.Backend {
	case "s3":
		return NewS3Backend(ctx, cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, cfg.Region)
	case "local":
		return NewLocalBackend(cfg.LocalDir)
	default:
		return nil, fmt.Errorf("unknown storage backend: %s", cfg.Backend)
	}
}

func ConfigFromEnv() Config {
	return Config{
		Backend:   envDefault("STORAGE_BACKEND", "s3"),
		Endpoint:  envDefault("STORAGE_ENDPOINT", "http://localhost:9000"),
		AccessKey: envDefault("STORAGE_ACCESS_KEY", "dummy"),
		SecretKey: envDefault("STORAGE_SECRET_KEY", "dummy"),
		Region:    os.Getenv("STORAGE_REGION"),
		LocalDir:  envDefault("STORAGE_LOCAL_DIR", "/data/files"),
	}
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

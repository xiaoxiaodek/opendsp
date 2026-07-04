package rta

import (
	"fmt"
	"os"

	"github.com/opendsp/opendsp/internal/config"
	"gopkg.in/yaml.v3"
)

type RTAEntry struct {
	AdvertiserID int64  `yaml:"id"`
	Endpoint     string `yaml:"endpoint"`
	TimeoutMs    int64  `yaml:"timeout_ms"`
}

type RegistryConfig struct {
	Advertisers []RTAEntry `yaml:"advertisers"`
}

type Registry struct {
	entries map[int64]RTAEntry
}

func NewRegistry(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("rta registry: %w", err)
	}
	var cfg RegistryConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("rta registry: %w", err)
	}

	entries := make(map[int64]RTAEntry)
	for _, e := range cfg.Advertisers {
		entries[e.AdvertiserID] = e
	}
	return &Registry{entries: entries}, nil
}

func NewRegistryFromConfig(cfg config.RTAConfig) *Registry {
	entries := make(map[int64]RTAEntry)
	for _, a := range cfg.Advertisers {
		entries[a.ID] = RTAEntry{
			AdvertiserID: a.ID,
			Endpoint:     a.Endpoint,
			TimeoutMs:    a.TimeoutMs,
		}
	}
	return &Registry{entries: entries}
}

func (r *Registry) Get(advertiserID int64) (RTAEntry, bool) {
	e, ok := r.entries[advertiserID]
	return e, ok
}

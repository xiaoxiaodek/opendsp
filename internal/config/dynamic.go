package config

import (
	"strconv"
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/knadh/koanf/v2"
)

type DynamicStore struct {
	mu     sync.RWMutex
	data   map[string]string
	static *koanf.Koanf
}

func NewDynamicStore(static *koanf.Koanf) *DynamicStore {
	return &DynamicStore{
		data:   make(map[string]string),
		static: static,
	}
}

func (d *DynamicStore) GetString(key, fallback string) string {
	d.mu.RLock()
	v, ok := d.data[key]
	d.mu.RUnlock()
	if ok {
		return v
	}
	return fallback
}

func (d *DynamicStore) GetFloat64(key string, fallback float64) float64 {
	d.mu.RLock()
	v, ok := d.data[key]
	d.mu.RUnlock()
	if ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func (d *DynamicStore) GetBool(key string, fallback bool) bool {
	d.mu.RLock()
	v, ok := d.data[key]
	d.mu.RUnlock()
	if ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func (d *DynamicStore) GetInt64(key string, fallback int64) int64 {
	d.mu.RLock()
	v, ok := d.data[key]
	d.mu.RUnlock()
	if ok {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

func (d *DynamicStore) Watch(cfg ConsulConfig) {
	if cfg.Address == "" {
		return
	}
	client, err := api.NewClient(&api.Config{Address: cfg.Address})
	if err != nil {
		return
	}

	prefix := cfg.Prefix
	kv := client.KV()

	pairs, _, err := kv.List(prefix, nil)
	if err == nil {
		d.mu.Lock()
		for _, pair := range pairs {
			d.data[pair.Key] = string(pair.Value)
		}
		d.mu.Unlock()
	}

	var lastIndex uint64
	go func() {
		for {
			pairs, meta, err := kv.List(prefix, &api.QueryOptions{WaitIndex: lastIndex})
			if err != nil {
				continue
			}
			lastIndex = meta.LastIndex
			d.mu.Lock()
			d.data = make(map[string]string)
			for _, pair := range pairs {
				d.data[pair.Key] = string(pair.Value)
			}
			d.mu.Unlock()
		}
	}()
}

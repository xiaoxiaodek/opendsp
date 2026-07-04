package config

import (
	"testing"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

func TestDynamicStore_GetString_Fallback(t *testing.T) {
	k := koanf.New(".")
	k.Load(confmap.Provider(map[string]interface{}{}, "."), nil)
	d := NewDynamicStore(k)

	v := d.GetString("nonexistent", "fallback-value")
	if v != "fallback-value" {
		t.Errorf("expected fallback-value, got %s", v)
	}
}

func TestDynamicStore_GetString_Dynamic(t *testing.T) {
	k := koanf.New(".")
	k.Load(confmap.Provider(map[string]interface{}{}, "."), nil)
	d := NewDynamicStore(k)

	d.mu.Lock()
	d.data["key1"] = "dynamic-value"
	d.mu.Unlock()

	v := d.GetString("key1", "fallback")
	if v != "dynamic-value" {
		t.Errorf("expected dynamic-value, got %s", v)
	}
}

func TestDynamicStore_GetFloat64(t *testing.T) {
	k := koanf.New(".")
	k.Load(confmap.Provider(map[string]interface{}{}, "."), nil)
	d := NewDynamicStore(k)

	d.mu.Lock()
	d.data["threshold"] = "0.75"
	d.mu.Unlock()

	v := d.GetFloat64("threshold", 0.8)
	if v != 0.75 {
		t.Errorf("expected 0.75, got %f", v)
	}

	v = d.GetFloat64("nonexistent", 0.8)
	if v != 0.8 {
		t.Errorf("expected fallback 0.8, got %f", v)
	}

	d.mu.Lock()
	d.data["bad"] = "not-a-float"
	d.mu.Unlock()
	v = d.GetFloat64("bad", 0.8)
	if v != 0.8 {
		t.Errorf("expected fallback 0.8 for invalid, got %f", v)
	}
}

func TestDynamicStore_GetBool(t *testing.T) {
	k := koanf.New(".")
	k.Load(confmap.Provider(map[string]interface{}{}, "."), nil)
	d := NewDynamicStore(k)

	d.mu.Lock()
	d.data["enabled"] = "true"
	d.mu.Unlock()

	if !d.GetBool("enabled", false) {
		t.Error("expected true")
	}
	if d.GetBool("nonexistent", false) {
		t.Error("expected false fallback")
	}
}

func TestDynamicStore_GetInt64(t *testing.T) {
	k := koanf.New(".")
	k.Load(confmap.Provider(map[string]interface{}{}, "."), nil)
	d := NewDynamicStore(k)

	d.mu.Lock()
	d.data["count"] = "42"
	d.mu.Unlock()

	v := d.GetInt64("count", 0)
	if v != 42 {
		t.Errorf("expected 42, got %d", v)
	}
}

func TestDynamicStore_DeleteRevertsToFallback(t *testing.T) {
	k := koanf.New(".")
	k.Load(confmap.Provider(map[string]interface{}{}, "."), nil)
	d := NewDynamicStore(k)

	d.mu.Lock()
	d.data["key1"] = "dynamic"
	d.mu.Unlock()

	if d.GetString("key1", "fallback") != "dynamic" {
		t.Error("expected dynamic value")
	}

	d.mu.Lock()
	delete(d.data, "key1")
	d.mu.Unlock()

	if d.GetString("key1", "fallback") != "fallback" {
		t.Error("expected fallback after delete")
	}
}

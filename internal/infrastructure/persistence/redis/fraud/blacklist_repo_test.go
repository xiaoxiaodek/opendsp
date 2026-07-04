package fraud

import (
	"testing"
)

func TestBlacklistKey(t *testing.T) {
	tests := []struct {
		listType string
		expected string
	}{
		{"ip", keyIPBlacklist},
		{"device", keyDeviceBlacklist},
		{"ua", keyUABlacklist},
		{"geo", keyGeoBlacklist},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := blacklistKey(tt.listType)
		if got != tt.expected {
			t.Errorf("blacklistKey(%q) = %q, want %q", tt.listType, got, tt.expected)
		}
	}
}

func TestContainsSubstring(t *testing.T) {
	tests := []struct {
		s       string
		pattern string
		want    bool
	}{
		{"Mozilla/5.0 badbot", "badbot", true},
		{"Mozilla/5.0", "badbot", false},
		{"hello", "hello", true},
		{"short", "longerpattern", false},
		{"", "anything", false},
		{"anything", "", false},
	}

	for _, tt := range tests {
		got := containsSubstring(tt.s, tt.pattern)
		if got != tt.want {
			t.Errorf("containsSubstring(%q, %q) = %v, want %v", tt.s, tt.pattern, got, tt.want)
		}
	}
}

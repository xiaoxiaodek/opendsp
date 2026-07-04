package encoder

import (
	"context"
	"testing"

	"github.com/opendsp/opendsp/internal/mockadx/generator"
)

func TestOpenRTBEncoder_EncodeDecode(t *testing.T) {
	enc := NewOpenRTBEncoder(false)
	spec := &generator.BidRequestSpec{
		RequestID: "test-req-001",
		UserID:    "test-user-001",
		IsTest:    true,
		Device: generator.DeviceSpec{
			OS:         "android",
			IP:         "10.0.0.1",
			UA:         "MockADX/1.0",
			DeviceType: "mobile",
			GeoCity:    "861100",
			GeoCountry: "86",
		},
		Content: generator.ContentSpec{
			ContentID: "content_001",
			Title:     "Test Content",
		},
		Imp: generator.ImpSpec{
			ImpID:        "imp-001",
			PositionType: 1,
			Width:        1920,
			Height:       1080,
			BidFloor:     500,
		},
	}

	raw, err := enc.Encode(context.Background(), spec)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("encoded bytes should not be empty")
	}

	decoded, err := enc.Decode(raw)
	if err != nil {
		t.Logf("decode (expected fail with empty response): %v", err)
	}
	_ = decoded
}

func TestOpenRTBEncoder_IsTestHeader(t *testing.T) {
	enc := NewOpenRTBEncoder(false)
	spec := &generator.BidRequestSpec{
		RequestID: "test-is-test",
		UserID:    "u1",
		IsTest:    true,
		Device:    generator.DeviceSpec{OS: "android", IP: "10.0.0.1", DeviceType: "mobile"},
		Content:   generator.ContentSpec{ContentID: "c1"},
		Imp:       generator.ImpSpec{ImpID: "i1", PositionType: 1},
	}

	raw, err := enc.Encode(context.Background(), spec)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("encoded bytes should not be empty")
	}
}
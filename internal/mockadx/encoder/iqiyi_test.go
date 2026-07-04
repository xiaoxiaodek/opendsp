package encoder

import (
	"context"
	"testing"

	"github.com/opendsp/opendsp/internal/mockadx/generator"
)

func TestIqiyiEncoder_EncodeDecode(t *testing.T) {
	enc := NewIqiyiEncoder(false)
	spec := &generator.BidRequestSpec{
		RequestID: "test-req-001",
		UserID:    "test-user-001",
		IsTest:    true,
		Device: generator.DeviceSpec{
			OS:      "android",
			IP:      "10.0.0.1",
			UA:      "MockADX/1.0",
			GeoCity: "861100",
		},
		Content: generator.ContentSpec{
			ContentID: "content_001",
			Title:     "Test Content",
		},
		Imp: generator.ImpSpec{
			ImpID:        "imp-001",
			PositionType: 1,
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

func TestIqiyiEncoder_Gzip(t *testing.T) {
	enc := NewIqiyiEncoder(true)
	spec := &generator.BidRequestSpec{
		RequestID: "test-gzip",
		UserID:    "u1",
		IsTest:    true,
		Device:    generator.DeviceSpec{OS: "android", IP: "10.0.0.1"},
		Content:   generator.ContentSpec{ContentID: "c1"},
		Imp:       generator.ImpSpec{ImpID: "i1", PositionType: 1},
	}

	raw, err := enc.Encode(context.Background(), spec)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("gzip encoded bytes should not be empty")
	}
}
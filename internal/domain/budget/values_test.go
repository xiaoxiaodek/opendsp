package budget

import (
	"testing"
)

func TestMoney_Add(t *testing.T) {
	a := Money{AmountMicros: 1_000_000}
	b := Money{AmountMicros: 500_000}
	result := a.Add(b)
	if result.AmountMicros != 1_500_000 {
		t.Errorf("expected 1_500_000, got %d", result.AmountMicros)
	}
}

func TestMoney_Sub(t *testing.T) {
	a := Money{AmountMicros: 1_000_000}
	b := Money{AmountMicros: 300_000}
	result := a.Sub(b)
	if result.AmountMicros != 700_000 {
		t.Errorf("expected 700_000, got %d", result.AmountMicros)
	}
}

func TestMoney_IsNegative(t *testing.T) {
	pos := Money{AmountMicros: 100}
	if pos.IsNegative() {
		t.Error("positive money should not be negative")
	}

	neg := Money{AmountMicros: -100}
	if !neg.IsNegative() {
		t.Error("negative money should be negative")
	}

	zero := Money{AmountMicros: 0}
	if zero.IsNegative() {
		t.Error("zero should not be negative")
	}
}

func TestMoney_ToFloat64(t *testing.T) {
	m := Money{AmountMicros: 1_500_000}
	if m.ToFloat64() != 1.5 {
		t.Errorf("expected 1.5, got %f", m.ToFloat64())
	}
}

func TestNewMoneyFromFloat64(t *testing.T) {
	m := NewMoneyFromFloat64(2.5)
	if m.AmountMicros != 2_500_000 {
		t.Errorf("expected 2_500_000 micros, got %d", m.AmountMicros)
	}
}

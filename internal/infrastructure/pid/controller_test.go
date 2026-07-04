package pid

import (
	"math"
	"testing"
)

func TestPIDController_Converges(t *testing.T) {
	c := NewController(0.3, 0.05, 0.1, 2.0)

	actual := 1.0
	for i := 0; i < 50; i++ {
		adj := c.Adjust(actual)
		actual += adj * 0.5
	}
	if math.Abs(actual-2.0) > 0.1 {
		t.Errorf("expected actual near 2.0, got %f", actual)
	}
}

func TestPIDController_FirstCallReturnsZero(t *testing.T) {
	c := NewController(0.3, 0.05, 0.1, 2.0)
	adj := c.Adjust(1.0)
	if adj != 0 {
		t.Errorf("first adjustment should be 0, got %f", adj)
	}
}

func TestPIDController_Reset(t *testing.T) {
	c := NewController(0.3, 0.05, 0.1, 2.0)
	c.Adjust(1.0)
	c.Adjust(1.0)
	c.Reset()
	adj := c.Adjust(1.0)
	if adj != 0 {
		t.Errorf("after reset, first adjustment should be 0, got %f", adj)
	}
}

func TestPIDController_NegativeAdjustment(t *testing.T) {
	c := NewController(0.5, 0.0, 0.0, 2.0)
	c.Adjust(1.0) // first call returns 0
	adj := c.Adjust(3.0) // actual > target
	if adj >= 0 {
		t.Errorf("expected negative adjustment when actual > target, got %f", adj)
	}
}

func TestPIDController_PositiveAdjustment(t *testing.T) {
	c := NewController(0.5, 0.0, 0.0, 2.0)
	c.Adjust(1.0) // first call
	adj := c.Adjust(0.5) // actual < target
	if adj <= 0 {
		t.Errorf("expected positive adjustment when actual < target, got %f", adj)
	}
}

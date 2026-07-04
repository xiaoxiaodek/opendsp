package budget

import "time"

// Money represents a monetary amount in micros (1/1,000,000 of currency unit).
// Using int64 avoids floating-point precision issues in financial calculations.
type Money struct {
	AmountMicros int64
}

// Add returns a new Money with the sum.
func (m Money) Add(other Money) Money {
	return Money{AmountMicros: m.AmountMicros + other.AmountMicros}
}

// Sub returns a new Money with the difference.
func (m Money) Sub(other Money) Money {
	return Money{AmountMicros: m.AmountMicros - other.AmountMicros}
}

// IsNegative returns true if the amount is less than zero.
func (m Money) IsNegative() bool {
	return m.AmountMicros < 0
}

// ToFloat64 converts micros to a float64 currency value.
func (m Money) ToFloat64() float64 {
	return float64(m.AmountMicros) / 1_000_000.0
}

// NewMoneyFromFloat64 creates Money from a float64 currency value.
func NewMoneyFromFloat64(amount float64) Money {
	return Money{AmountMicros: int64(amount * 1_000_000)}
}

// PreFreezeToken represents a reserved budget amount for a pending bid.
type PreFreezeToken struct {
	ID           string
	Amount       Money
	AdvertiserID int64 // owner of the budget, for Release/Confirm
	ExpiresAt    time.Time
}

// IsExpired returns true if the token has expired.
func (t PreFreezeToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

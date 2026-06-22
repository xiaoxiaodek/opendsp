package data

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func float64ToNumeric(v *float64) pgtype.Numeric {
	if v == nil {
		return pgtype.Numeric{Valid: false}
	}
	n := pgtype.Numeric{}
	_ = n.Scan(fmt.Sprintf("%.2f", *v))
	return n
}

func numericToFloat64(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f, _ := n.Float64Value()
	return &f.Float64
}

func timeToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func timestamptzToTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func ptrInt16(v *int16) int16 {
	if v == nil { return 0 }
	return *v
}

func ptrInt32(v *int32) int32 {
	if v == nil { return 0 }
	return *v
}

func ptrInt64(v *int64) int64 {
	if v == nil { return 0 }
	return *v
}

func ptrFloat64(v *float64) float64 {
	if v == nil { return 0 }
	return *v
}

func ptrStr(v *string) string {
	if v == nil { return "" }
	return *v
}

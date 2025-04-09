package storage

import (
	"database/sql"
	"time"
)

// NewNullFloat64 returns a value as null type. Will assume invalid when value is zero.
func NewNullFloat64(v float64) sql.NullFloat64 {
	if v == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: v, Valid: true}
}

// NewNullInt64 returns a value as null type. Will assume invalid when value is zero.
func NewNullInt64(v int64) sql.NullInt64 {
	if v == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: v, Valid: true}
}

func NewNullString(v string) sql.NullString {
	return sql.NullString{String: v, Valid: true}
}

// NewNullTimeFromTime returns a value as null type. Will assume invalid when value is zero.
func NewNullTimeFromTime(v time.Time) sql.NullTime {
	if v.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: v, Valid: true}
}

func NewTimeFromNullTime(v sql.NullTime) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return v.Time
}

package optional

import (
	"database/sql"
	"time"

	"golang.org/x/exp/constraints"
)

// FromNullFloat64 converts a sql.Null variable to it's Optional equivalent and returns it.
func FromNullFloat64(v sql.NullFloat64) Optional[float64] {
	if !v.Valid {
		return Optional[float64]{}
	}
	return New(v.Float64)
}

// FromNullInt64 converts a sql.Null variable to it's Optional equivalent and returns it.
func FromNullInt64(v sql.NullInt64) Optional[int64] {
	if !v.Valid {
		return Optional[int64]{}
	}
	return New(v.Int64)
}

// FromNullInt64ToInteger converts an sql.Null variable to an Optional of a different integer type and returns it.
func FromNullInt64ToInteger[T constraints.Integer](v sql.NullInt64) Optional[T] {
	if !v.Valid {
		return Optional[T]{}
	}
	return New(T(v.Int64))
}

// FromNullInt64 converts a sql.Null variable to it's Optional equivalent and returns it.
func FromNullString(v sql.NullString) Optional[string] {
	if !v.Valid {
		return Optional[string]{}
	}
	return New(v.String)
}

// FromNullTime converts a sql.Null variable to it's Optional equivalent and returns it.
func FromNullTime(v sql.NullTime) Optional[time.Time] {
	if !v.Valid {
		return Optional[time.Time]{}
	}
	return New(v.Time)
}

// ToNullFloat64 converts an Optional variable to it's sql.Null equivalent and returns it.
func ToNullFloat64[T constraints.Float](o Optional[T]) sql.NullFloat64 {
	if o.IsEmpty() {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: float64(o.ValueOrZero()), Valid: true}
}

// ToNullInt64 converts an Optional variable to it's sql.Null equivalent and returns it.
func ToNullInt64[T constraints.Integer](o Optional[T]) sql.NullInt64 {
	if o.IsEmpty() {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(o.ValueOrZero()), Valid: true}
}

// ToNullString converts an Optional variable to it's sql.Null equivalent and returns it.
func ToNullString(o Optional[string]) sql.NullString {
	if o.IsEmpty() {
		return sql.NullString{}
	}
	return sql.NullString{String: o.ValueOrZero(), Valid: true}
}

// ToNullTime converts an Optional variable to it's sql.Null equivalent and returns it.
func ToNullTime(o Optional[time.Time]) sql.NullTime {
	if o.IsEmpty() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: o.ValueOrZero(), Valid: true}
}

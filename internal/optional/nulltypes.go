package optional

import (
	"database/sql"
	"time"

	"golang.org/x/exp/constraints"
)

func FromNullFloat64(v sql.NullFloat64) Optional[float64] {
	if !v.Valid {
		return Optional[float64]{}
	}
	return New(v.Float64)
}

func FromNullInt64(v sql.NullInt64) Optional[int64] {
	if !v.Valid {
		return Optional[int64]{}
	}
	return New(v.Int64)
}

func FromNullInt64ToInteger[T constraints.Integer](v sql.NullInt64) Optional[T] {
	if !v.Valid {
		return Optional[T]{}
	}
	return New(T(v.Int64))
}

func FromNullTime(v sql.NullTime) Optional[time.Time] {
	if !v.Valid {
		return Optional[time.Time]{}
	}
	return New(v.Time)
}

func ToNullFloat64[T constraints.Float](o Optional[T]) sql.NullFloat64 {
	if o.IsNil() {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: float64(o.ValueOrZero()), Valid: true}
}

func ToNullInt64[T constraints.Integer](o Optional[T]) sql.NullInt64 {
	if o.IsNil() {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(o.ValueOrZero()), Valid: true}
}

func ToNullTime(o Optional[time.Time]) sql.NullTime {
	if o.IsNil() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: o.ValueOrZero(), Valid: true}
}

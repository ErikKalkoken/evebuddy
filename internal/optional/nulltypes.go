package optional

import (
	"database/sql"
	"time"
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

func FromNullInt64ToInteger[T IntType](v sql.NullInt64) Optional[T] {
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

func ToNullFloat64[T FloatType](o Optional[T]) sql.NullFloat64 {
	if o.IsNone() {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: float64(o.ValueOrZero()), Valid: true}
}

func ToNullInt64[T IntType](o Optional[T]) sql.NullInt64 {
	if o.IsNone() {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(o.ValueOrZero()), Valid: true}
}

func ToNullTime(o Optional[time.Time]) sql.NullTime {
	if o.IsNone() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: o.ValueOrZero(), Valid: true}
}

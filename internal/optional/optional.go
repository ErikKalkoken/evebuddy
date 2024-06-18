// Package optional provides optional types.
package optional

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type IntType interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64
}

type FloatType interface {
	float32 | float64
}

type Numeric interface {
	IntType | FloatType
}

type Optional[T any] struct {
	value T
	isSet bool
}

func New[T any](v T) Optional[T] {
	o := Optional[T]{value: v, isSet: true}
	return o
}

func NewNone[T any]() Optional[T] {
	o := Optional[T]{isSet: false}
	return o
}

// NewNumeric returns a new numeric optional with type conversion.
func NewNumeric[X Numeric, Y Numeric](v X) Optional[Y] {
	o := Optional[Y]{value: Y(v), isSet: true}
	return o
}

func (o Optional[T]) IsNone() bool {
	return !o.isSet
}

func (o Optional[T]) IsValue() bool {
	return o.isSet
}

func (o *Optional[T]) Set(v T) {
	o.value = v
	o.isSet = true
}

func (o *Optional[T]) SetNone() {
	var z T
	o.value = z
	o.isSet = false
}

func (o Optional[T]) String() string {
	if o.IsNone() {
		return "None"
	}
	return fmt.Sprint(o.value)
}

func (o Optional[T]) MustValue() T {
	if o.IsNone() {
		panic("None has no value")
	}
	return o.value
}

func (o Optional[T]) Value() (T, error) {
	var z T
	if o.IsNone() {
		return z, errors.New("can not retrieve value from None")
	}
	return o.value, nil
}

func (o Optional[T]) ValueOrFallback(fallback T) T {
	if o.IsNone() {
		return fallback
	}
	return o.value
}

func (o Optional[T]) ValueOrZero() T {
	var z T
	if o.IsNone() {
		return z
	}
	return o.value
}

// ConvertNumeric converts between numeric optionals.
func ConvertNumeric[X Numeric, Y Numeric](o Optional[X]) Optional[Y] {
	if o.IsNone() {
		return Optional[Y]{}
	}
	return New(Y(o.ValueOrZero()))
}

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

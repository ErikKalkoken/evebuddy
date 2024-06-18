// Package optional provides optional types.
package optional

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

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

func FromNullTime(v sql.NullTime) Optional[time.Time] {
	if !v.Valid {
		return Optional[time.Time]{}
	}
	return New(v.Time)
}

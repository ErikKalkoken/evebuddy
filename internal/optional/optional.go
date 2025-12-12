// Package optional provides the generic type Optional, which represents an optional value.
//
// It also includes helpers to convert variables of sql.NullX type to Optional type and vice versa.
package optional

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/exp/constraints"
)

type Numeric interface {
	constraints.Integer | constraints.Float
}

var ErrIsEmpty = errors.New("optional is empty")

// Optional represents a variable that may contain a value or not.
//
// The zero value for an Optional is an empty optional.
type Optional[T any] struct {
	value     T
	isPresent bool
}

// New returns a new Optional with the value v.
func New[T any](v T) Optional[T] {
	o := Optional[T]{value: v, isPresent: true}
	return o
}

// Clear removes any value.
func (o *Optional[T]) Clear() {
	var z T
	o.value = z
	o.isPresent = false
}

// IsEmpty reports whether an Optional is empty.
func (o Optional[T]) IsEmpty() bool {
	return !o.isPresent
}

// MustValue returns the value of an Optional or panics if it is empty.
func (o Optional[T]) MustValue() T {
	if o.IsEmpty() {
		panic(ErrIsEmpty)
	}
	return o.value
}

// Set sets a new value.
func (o *Optional[T]) Set(v T) {
	o.value = v
	o.isPresent = true
}

// String returns a string representation of an Optional.
func (o Optional[T]) String() string {
	if o.IsEmpty() {
		return "<empty>"
	}
	return fmt.Sprint(o.value)
}

// StringFunc returns the result of applying convert when the optional has a value.
// Or it returns the provided fallback, when the optional is empty,
func (o Optional[T]) StringFunc(fallback string, convert func(v T) string) string {
	if o.IsEmpty() {
		return fallback
	}
	return convert(o.ValueOrZero())
}

// Value returns the value of an Optional and reports whether the value exists.
func (o Optional[T]) Value() (T, bool) {
	var z T
	if o.IsEmpty() {
		return z, false
	}
	return o.value, true
}

// ValueOrFallback returns the value of an Optional or a fallback if it is empty.
func (o Optional[T]) ValueOrFallback(fallback T) T {
	if o.IsEmpty() {
		return fallback
	}
	return o.value
}

// ValueOrZero returns the value of an Optional or it's type's zero value if it is empty.
func (o Optional[T]) ValueOrZero() T {
	var z T
	if o.IsEmpty() {
		return z
	}
	return o.value
}

// ConvertNumeric converts between numeric optionals.
func ConvertNumeric[X Numeric, Y Numeric](o Optional[X]) Optional[Y] {
	if o.IsEmpty() {
		return Optional[Y]{}
	}
	return New(Y(o.ValueOrZero()))
}

// FromIntegerWithZero returns an optional from an integer
// where a zero value is interpreted as empty.
func FromIntegerWithZero[T constraints.Integer](v T) Optional[T] {
	if v == 0 {
		return Optional[T]{}
	}
	return New(v)
}

// FromTimeWithZero returns an optional from a [time.Time]
// where a zero value is interpreted as empty.
func FromTimeWithZero(v time.Time) Optional[time.Time] {
	if v.IsZero() {
		return Optional[time.Time]{}
	}
	return New(v)
}

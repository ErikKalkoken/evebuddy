// Package optional provides type safe optional types
// and the ability to convert them to and from sql Null types.
package optional

import (
	"errors"
	"fmt"

	"golang.org/x/exp/constraints"
)

type Numeric interface {
	constraints.Integer | constraints.Float
}

// Optional represents a variable that may contain a value or not.
type Optional[T any] struct {
	value T
	isNil bool
}

// New returns a new optional variable with a value.
func New[T any](v T) Optional[T] {
	o := Optional[T]{value: v, isNil: true}
	return o
}

// IsNil reports wether an optional is nil.
func (o Optional[T]) IsNil() bool {
	return !o.isNil
}

func (o Optional[T]) IsValue() bool {
	return o.isNil
}

// Set sets a new value.
func (o *Optional[T]) Set(v T) {
	o.value = v
	o.isNil = true
}

// SetNil removes any value.
func (o *Optional[T]) SetNil() {
	var z T
	o.value = z
	o.isNil = false
}

// String returns a string representation of an optional.
func (o Optional[T]) String() string {
	if o.IsNil() {
		return "Nil"
	}
	return fmt.Sprint(o.value)
}

// MustValue returns the value of an optional or panics if it is nil.
func (o Optional[T]) MustValue() T {
	if o.IsNil() {
		panic("None has no value")
	}
	return o.value
}

// Value returns the value of an optional.
func (o Optional[T]) Value() (T, error) {
	var z T
	if o.IsNil() {
		return z, errors.New("optional is nil")
	}
	return o.value, nil
}

// ValueOrFallback returns the value of an optional or a given fallback if it is nil.
func (o Optional[T]) ValueOrFallback(fallback T) T {
	if o.IsNil() {
		return fallback
	}
	return o.value
}

// ValueOrZero returns the value of an optional or it's type's zero value if it is nil.
func (o Optional[T]) ValueOrZero() T {
	var z T
	if o.IsNil() {
		return z
	}
	return o.value
}

// ConvertNumeric converts between numeric optionals.
func ConvertNumeric[X Numeric, Y Numeric](o Optional[X]) Optional[Y] {
	if o.IsNil() {
		return Optional[Y]{}
	}
	return New(Y(o.ValueOrZero()))
}

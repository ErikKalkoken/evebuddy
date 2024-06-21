// Package optional provides type safe optional variables.
//
// It also provides the ability to convert optionals to and from sql Null types.
package optional

import (
	"errors"
	"fmt"

	"golang.org/x/exp/constraints"
)

type Numeric interface {
	constraints.Integer | constraints.Float
}

var ErrIsEmpty = errors.New("optional is empty")

// Optional represents a variable that may contain a value or not.
//
// Note that the zero value of an Optional is a an empty Optional.
type Optional[T any] struct {
	value     T
	isPresent bool
}

// New returns a new Optional with a value.
func New[T any](v T) Optional[T] {
	o := Optional[T]{value: v, isPresent: true}
	return o
}

// IsEmpty reports wether an Optional is empty.
func (o Optional[T]) IsEmpty() bool {
	return !o.isPresent
}

// Set sets a new value.
func (o *Optional[T]) Set(v T) {
	o.value = v
	o.isPresent = true
}

// Clear removes any value.
func (o *Optional[T]) Clear() {
	var z T
	o.value = z
	o.isPresent = false
}

// String returns a string representation of an Optional.
func (o Optional[T]) String() string {
	if o.IsEmpty() {
		return "<empty>"
	}
	return fmt.Sprint(o.value)
}

// MustValue returns the value of an Optional or panics if it is empty.
func (o Optional[T]) MustValue() T {
	if o.IsEmpty() {
		panic(ErrIsEmpty)
	}
	return o.value
}

// Value returns the value of an Optional.
func (o Optional[T]) Value() (T, error) {
	var z T
	if o.IsEmpty() {
		return z, ErrIsEmpty
	}
	return o.value, nil
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

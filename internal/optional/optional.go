// Package optional provides the generic type Optional, which represents an optional value.
//
// It also includes helpers to convert variables of sql.NullX type to Optional type and vice versa.
package optional

import (
	"encoding/json"
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
// The zero value for an Optional is an empty optional.
type Optional[T any] struct {
	value     T
	isPresent bool
}

// New returns an optional with the value v.
func New[T any](v T) Optional[T] {
	o := Optional[T]{value: v, isPresent: true}
	return o
}

// FromZeroValue returns an optional from a value
// where it's zero value is interpreted as empty.
func FromZeroValue[T comparable](v T) Optional[T] {
	var z T
	if v == z {
		return Optional[T]{}
	}
	return New(v)
}

// FromPtr returns a new optional from a pointer optional.
// A nil pointer is interpreted as not present.
func FromPtr[T any](x *T) Optional[T] {
	if x == nil {
		var z Optional[T]
		return z
	}
	return New(*x)
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

// Ptr returns the optional as pointer optional.
func (o Optional[T]) Ptr() *T {
	if !o.isPresent {
		return nil
	}
	v := o.value
	return &v
}

// Set sets a new value.
func (o *Optional[T]) Set(v T) {
	o.value = v
	o.isPresent = true
}

// SetWhenEmpty sets a new value when the optional is empty.
func (o *Optional[T]) SetWhenEmpty(v T) {
	if !o.isPresent {
		o.Set(v)
	}
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

// MarshalJSON returns the JSON encoding of the optional.
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.isPresent {
		return json.Marshal(nil)
	}
	v := o.value
	return json.Marshal(&v)
}

// UnmarshalJSON parses the JSON-encoded data b and replaces the current optional.
// JSON null values will be unmarshaled into an empty optional.
func (o *Optional[T]) UnmarshalJSON(b []byte) error {
	var x *T
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}
	if x == nil {
		o.Clear()
		return nil
	}
	o.Set(*x)
	return nil
}

// ConvertNumeric converts between numeric optionals.
func ConvertNumeric[X Numeric, Y Numeric](o Optional[X]) Optional[Y] {
	if o.IsEmpty() {
		return Optional[Y]{}
	}
	return New(Y(o.ValueOrZero()))
}

// Equal reports whether two optionals with comparable values are equal.
func Equal[T comparable](a, b Optional[T]) bool {
	if a.isPresent != b.isPresent {
		return false
	}
	if !a.isPresent && !b.isPresent {
		return true
	}
	return a.value == b.value
}

type Equaler[T any] interface {
	Equal(other T) bool
}

// Equal2 reports whether two optionals with values
// that satisfy the Equaler interface are equal.
func Equal2[T Equaler[T]](a, b Optional[T]) bool {
	if a.isPresent != b.isPresent {
		return false
	}
	if !a.isPresent && !b.isPresent {
		return true
	}
	return a.value.Equal(b.value)
}

// Sum returns the sum of values v.
// Empty values are added with their zero value (e.g. 0).
// When all values are empty it returns an empty value.
func Sum[T Numeric](v ...Optional[T]) Optional[T] {
	var s T
	var isPresent bool
	for _, u := range v {
		if u.isPresent {
			s += u.value
			isPresent = true
		}
	}
	if !isPresent {
		return Optional[T]{}
	}
	return New(s)
}

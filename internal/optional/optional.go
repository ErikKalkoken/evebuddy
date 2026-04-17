// Package optional provides the generic type Optional, which represents an optional value.
//
// It also includes helpers to convert variables of sql.NullX type to Optional type and vice versa.
package optional

import (
	"cmp"
	"encoding/json"
	"fmt"

	"golang.org/x/exp/constraints"
)

type numeric interface {
	constraints.Integer | constraints.Float
}

// Optional represents a variable that may contain a value or not.
//
// The zero value for an Optional is an empty optional.
type Optional[T any] struct {
	value     T
	isPresent bool
}

// New returns an optional with the value v.
func New[T any](v T) Optional[T] {
	o := Optional[T]{
		value:     v,
		isPresent: true,
	}
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

// MarshalJSON returns the JSON encoding of the optional.
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.isPresent {
		return json.Marshal(nil)
	}
	v := o.value
	return json.Marshal(&v)
}

// MustValue returns the value of an Optional or panics if it is empty.
func (o Optional[T]) MustValue() T {
	if !o.isPresent {
		panic("optional is empty")
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
	if !o.isPresent {
		return "<empty>"
	}
	return fmt.Sprint(o.value)
}

// StringFunc returns the result of applying convert when the optional has a value.
// Or it returns the provided fallback, when the optional is empty,
func (o Optional[T]) StringFunc(fallback string, mapper func(v T) string) string {
	if !o.isPresent {
		return fallback
	}
	return mapper(o.value)
}

// Value returns the value of an Optional and reports whether the value exists.
func (o Optional[T]) Value() (T, bool) {
	var z T
	if !o.isPresent {
		return z, false
	}
	return o.value, true
}

// ValueOrFallback returns the value of an Optional or a fallback if it is empty.
func (o Optional[T]) ValueOrFallback(fallback T) T {
	if !o.isPresent {
		return fallback
	}
	return o.value
}

// ValueOrZero returns the value of an Optional or it's type's zero value if it is empty.
func (o Optional[T]) ValueOrZero() T {
	var z T
	if !o.isPresent {
		return z
	}
	return o.value
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
func ConvertNumeric[X numeric, Y numeric](o Optional[X]) Optional[Y] {
	if !o.isPresent {
		return Optional[Y]{}
	}
	return New(Y(o.value))
}

// Compare compares the optional a with b.
// If a is less then b, it returns -1;
// if a is greater then b, it returns +1;
// if they're the same, it returns 0;
// An empty optional is less then a non-empty optional.
func Compare[T cmp.Ordered](a, b Optional[T]) int {
	if !a.isPresent && !b.isPresent {
		return 0
	}
	if !a.isPresent {
		return -1
	}
	if !b.isPresent {
		return 1
	}
	return cmp.Compare(a.value, b.value)
}

// CompareFunc compares the optional a with b by applying the compare function.
func CompareFunc[T any](a, b Optional[T], comparer func(T, T) int) int {
	if !a.isPresent && !b.isPresent {
		return 0
	}
	if !a.isPresent {
		return -1
	}
	if !b.isPresent {
		return 1
	}
	return comparer(a.value, b.value)
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

type equaler[T any] interface {
	Equal(other T) bool
}

// Equal2 reports whether two optionals with values
// that satisfy the Equaler interface are equal.
func Equal2[T equaler[T]](a, b Optional[T]) bool {
	if a.isPresent != b.isPresent {
		return false
	}
	if !a.isPresent && !b.isPresent {
		return true
	}
	return a.value.Equal(b.value)
}

// EqualFunc reports whether two optionals are equal
// using an equality function on each pair of elements.
func EqualFunc[T any](a, b Optional[T], eq func(a2, b2 T) bool) bool {
	if a.isPresent != b.isPresent {
		return false
	}
	if !a.isPresent && !b.isPresent {
		return true
	}
	return eq(a.value, b.value)
}

// Map returns the result of applying mapper on the value of o
// or fallback if o is empty.
func Map[X, Y any](o Optional[X], fallback Y, mapper func(v X) Y) Y {
	if !o.isPresent {
		return fallback
	}
	return mapper(o.value)
}

// FlatMap returns another optional with is the result of applying mapper on o.
func FlatMap[X, Y any](o Optional[X], mapper func(v X) Optional[Y]) Optional[Y] {
	if !o.isPresent {
		return Optional[Y]{}
	}
	return mapper(o.value)
}

// Sum returns the sum of values v.
// When any value is empty it returns an empty value.
// The behavior is models after SQL's SUM of nullable values.
func Sum[T numeric](v ...Optional[T]) Optional[T] {
	var s Optional[T]
	for _, u := range v {
		if !u.isPresent {
			return Optional[T]{}
		}
		s.value += u.value
		s.isPresent = true
	}
	return s
}

// SumNonEmpty returns the sum of non-empty values v.
// Empty values are ignored.
// When all values are empty it returns an empty value.
func SumNonEmpty[T numeric](v ...Optional[T]) Optional[T] {
	var s Optional[T]
	for _, u := range v {
		var v T
		if u.isPresent {
			v = u.value
			s.isPresent = true
			s.value += v
		}
	}
	return s
}

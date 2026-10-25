// Package xatomic provides additional atomic types.
package xatomic

import (
	"fmt"
	"sync/atomic"
)

// Integer is any type whose underlying type is an integer.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Enum is an atomic value of an integer-based type T.
// The zero value holds T(0). Must not be copied after first use.
type Enum[T Integer] struct {
	_ noCopy
	v atomic.Uint64 // any integer round-trips losslessly through uint64
}

// Load returns the current value.
func (a *Enum[T]) Load() T {
	return T(a.v.Load())
}

// Store sets the value to v.
func (a *Enum[T]) Store(v T) {
	a.v.Store(uint64(v))
}

// Swap sets the value to v and returns the old value.
func (a *Enum[T]) Swap(v T) (old T) {
	return T(a.v.Swap(uint64(v)))
}

// CompareAndSwap sets the value to new if it equals old and reports whether it did.
func (a *Enum[T]) CompareAndSwap(old, new T) (swapped bool) {
	return a.v.CompareAndSwap(uint64(old), uint64(new))
}

// String formats the current value, using T's String method if present.
func (a *Enum[T]) String() string {
	return fmt.Sprint(a.Load())
}

// noCopy makes go vet flag copies of the containing struct.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

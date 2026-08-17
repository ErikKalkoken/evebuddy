// Package stack provides a simple Stack container.
package stack

import (
	"fmt"
)

// Stack represents a simple generic Stack.
// The zero value is an empty stack ready to use.
// A Stack is not thread safe.
type Stack[T any] struct {
	s []T
}

// Clear removes all items.
func (s *Stack[T]) Clear() {
	if s == nil {
		return
	}
	clear(s.s)
	s.s = s.s[:0]
}

// Push adds an item on top.
func (s *Stack[T]) Push(v T) {
	if s == nil {
		return
	}
	s.s = append(s.s, v)
}

// Peek tries to return the top element and reports whether it exists.
func (s *Stack[T]) Peek() (T, bool) {
	if s == nil || len(s.s) == 0 {
		var z T
		return z, false
	}
	return s.s[len(s.s)-1], true
}

// Pop tries to return the top element and reports whether it exists.
func (s *Stack[T]) Pop() (T, bool) {
	var z T
	if s == nil || len(s.s) == 0 {
		return z, false
	}
	idx := len(s.s) - 1
	v := s.s[idx]
	s.s[idx] = z // Zero out to allow GC
	s.s = s.s[:idx]
	return v, true
}

// Size returns the number of items.
func (s *Stack[T]) Size() int {
	if s == nil {
		return 0
	}
	return len(s.s)
}

// String returns a string representation.
func (s *Stack[T]) String() string {
	if s == nil {
		return ""
	}
	return fmt.Sprint(s.s)
}

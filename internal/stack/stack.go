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
	clear(s.s)
	s.s = s.s[0:0]
}

// Push adds an item on top.
func (s *Stack[T]) Push(v T) {
	s.s = append(s.s, v)
}

// Peek tries to return the top element and reports whether it exists.
func (s Stack[T]) Peek() (T, bool) {
	if len(s.s) == 0 {
		var z T
		return z, false
	}
	return s.s[len(s.s)-1], true
}

// Pop tries to return the top element and reports whether it exists.
func (s *Stack[T]) Pop() (T, bool) {
	v, ok := s.Peek()
	if !ok {
		var z T
		return z, false
	}
	s.s = s.s[:(len(s.s) - 1)]
	return v, true
}

// Size returns the number of items.
func (s Stack[T]) Size() int {
	return len(s.s)
}

// String returns a string representation.
func (s Stack[T]) String() string {
	return fmt.Sprint(s.s)
}

// Package stack provides a simple Stack container.
package stack

import (
	"errors"
	"fmt"
)

var ErrEmpty = errors.New("empty")

// Stack represents a simple generic Stack.
// The zero value is an empty stack ready to use.
// A Stack is not thread safe.
type Stack[T any] struct {
	s []T
}

func (s *Stack[T]) Clear() {
	s.s = s.s[0:0]
}

func (s *Stack[T]) Push(v T) {
	if s.s == nil {
		s.s = make([]T, 0)
	}
	s.s = append(s.s, v)
}

func (s Stack[T]) Peek() (T, error) {
	var x T
	if len(s.s) == 0 {
		return x, ErrEmpty
	}
	return s.s[len(s.s)-1], nil
}

func (s *Stack[T]) Pop() (T, error) {
	v, err := s.Peek()
	if err != nil {
		return v, err
	}
	s.s = s.s[:(len(s.s) - 1)]
	return v, nil
}

func (s Stack[T]) Size() int {
	return len(s.s)
}

func (s Stack[T]) String() string {
	return fmt.Sprint(s.s)
}

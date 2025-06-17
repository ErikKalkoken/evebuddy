package stack_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/stack"
)

func TestStack(t *testing.T) {
	t.Run("can push and pop", func(t *testing.T) {
		var s stack.Stack[int]
		s.Push(99)
		s.Push(42)
		v, err := s.Pop()
		if assert.NoError(t, err) {
			assert.Equal(t, 42, v)
		}
		v, err = s.Pop()
		if assert.NoError(t, err) {
			assert.Equal(t, 99, v)
		}
	})
	t.Run("should return specific error when trying to pop from empty stack", func(t *testing.T) {
		var s stack.Stack[int]
		_, err := s.Pop()
		assert.ErrorIs(t, stack.ErrEmpty, err)
	})
	t.Run("should return correct stack size", func(t *testing.T) {
		var s stack.Stack[int]
		s.Push(99)
		s.Push(42)
		v := s.Size()
		assert.Equal(t, 2, v)
	})
	t.Run("can clear the stack", func(t *testing.T) {
		var s stack.Stack[int]
		s.Push(99)
		s.Clear()
		assert.Equal(t, 0, s.Size())
	})
	t.Run("can return the current value without popping", func(t *testing.T) {
		var s stack.Stack[int]
		s.Push(99)
		v, err := s.Peek()
		if assert.NoError(t, err) {
			assert.Equal(t, 99, v)
			assert.Equal(t, 1, s.Size())
		}
	})
	t.Run("should return specific error when trying to peek at empty stack", func(t *testing.T) {
		var s stack.Stack[int]
		_, err := s.Peek()
		assert.ErrorIs(t, stack.ErrEmpty, err)
	})
	t.Run("can print stack", func(t *testing.T) {
		var s stack.Stack[int]
		s.Push(99)
		x := fmt.Sprint(s)
		assert.Equal(t, "[99]", x)
	})
}

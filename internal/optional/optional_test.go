package optional_test

import (
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

func TestXxx(t *testing.T) {
	t.Run("can create new empty optional", func(t *testing.T) {
		x := optional.NewNone[int]()
		assert.True(t, x.IsNone())
	})
	t.Run("can create new optional with value", func(t *testing.T) {
		x := optional.New(55)
		assert.True(t, x.IsValue())
	})
	t.Run("should return value", func(t *testing.T) {
		x := optional.New(55)
		assert.True(t, x.IsValue())
	})
	t.Run("can update a none", func(t *testing.T) {
		x := optional.NewNone[int]()
		x.Set(45)
		assert.Equal(t, 45, x.MustValue())
	})
	t.Run("can update a non none", func(t *testing.T) {
		x := optional.New(12)
		x.Set(45)
		assert.Equal(t, 45, x.MustValue())
	})
	t.Run("can make a value to none", func(t *testing.T) {
		x := optional.New(12)
		x.SetNone()
		assert.True(t, x.IsNone())
	})
	t.Run("can print a value", func(t *testing.T) {
		x := optional.New(12)
		s := fmt.Sprint(x)
		assert.Equal(t, "12", s)
	})
	t.Run("can print a none", func(t *testing.T) {
		x := optional.NewNone[int]()
		s := fmt.Sprint(x)
		assert.Equal(t, "None", s)
	})
	t.Run("should return value when set", func(t *testing.T) {
		x := optional.New(12)
		got := x.ValueOrFallback(4)
		assert.Equal(t, 12, got)
	})
	t.Run("should return fallback when none", func(t *testing.T) {
		x := optional.NewNone[int]()
		got := x.ValueOrFallback(4)
		assert.Equal(t, 4, got)
	})
	t.Run("should return value when set", func(t *testing.T) {
		x := optional.New(12)
		got, err := x.Value()
		if assert.NoError(t, err) {
			assert.Equal(t, 12, got)
		}
	})
	t.Run("should return error when none", func(t *testing.T) {
		x := optional.NewNone[int]()
		_, err := x.Value()
		assert.Error(t, err)
	})
	t.Run("should panic when none", func(t *testing.T) {
		x := optional.NewNone[int]()
		assert.Panics(t, func() {
			x.MustValue()
		})
	})
	t.Run("should return value when set and not panic", func(t *testing.T) {
		x := optional.New(12)
		got := x.MustValue()
		assert.Equal(t, 12, got)
	})

}

func TestValueOrZero(t *testing.T) {
	t.Run("should return value when set", func(t *testing.T) {
		x := optional.New(12)
		got := x.ValueOrZero()
		assert.Equal(t, 12, got)
	})
	t.Run("should return zero value when none", func(t *testing.T) {
		x := optional.NewNone[int]()
		got := x.ValueOrZero()
		assert.Equal(t, 0, got)
	})
	t.Run("should return zero value when none", func(t *testing.T) {
		x := optional.NewNone[string]()
		got := x.ValueOrZero()
		assert.Equal(t, "", got)
	})
}

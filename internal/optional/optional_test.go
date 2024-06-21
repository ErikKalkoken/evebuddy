package optional_test

import (
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

func TestOptional(t *testing.T) {
	t.Run("can create new optional with value", func(t *testing.T) {
		x := optional.New(55)
		assert.True(t, x.IsValue())
	})
	t.Run("should return value", func(t *testing.T) {
		x := optional.New(55)
		assert.True(t, x.IsValue())
	})
	t.Run("can update an empty optional", func(t *testing.T) {
		x := optional.Optional[int]{}
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
	t.Run("can print an empty optional", func(t *testing.T) {
		x := optional.Optional[int]{}
		s := fmt.Sprint(x)
		assert.Equal(t, "None", s)
	})
	t.Run("should return value when set", func(t *testing.T) {
		x := optional.New(12)
		got := x.ValueOrFallback(4)
		assert.Equal(t, 12, got)
	})
	t.Run("should return fallback when empty", func(t *testing.T) {
		x := optional.Optional[int]{}
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
	t.Run("should return error when empty", func(t *testing.T) {
		x := optional.Optional[int]{}
		_, err := x.Value()
		assert.Error(t, err)
	})
	t.Run("should panic when empty", func(t *testing.T) {
		x := optional.Optional[int]{}
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
	t.Run("should return zero value integer optional is empty", func(t *testing.T) {
		x := optional.Optional[int]{}
		got := x.ValueOrZero()
		assert.Equal(t, 0, got)
	})
	t.Run("should return zero string value is empty", func(t *testing.T) {
		x := optional.Optional[string]{}
		got := x.ValueOrZero()
		assert.Equal(t, "", got)
	})
}

func TestConvertNumeric(t *testing.T) {
	assert.Equal(
		t,
		optional.New(int(99)),
		optional.ConvertNumeric[int64, int](optional.New(int64(99))),
	)
	assert.Equal(
		t,
		optional.New(float64(99)),
		optional.ConvertNumeric[int32, float64](optional.New(int32(99))),
	)
	assert.Equal(
		t,
		optional.Optional[float64]{},
		optional.ConvertNumeric[int32, float64](optional.Optional[int32]{}),
	)
}

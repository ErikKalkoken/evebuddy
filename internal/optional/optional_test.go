package optional_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestOptional_New(t *testing.T) {
	t.Run("can create new optional with value", func(t *testing.T) {
		x := optional.New(55)
		xassert.Equal(t, 55, x.MustValue())
		assert.False(t, x.IsEmpty())
	})
	t.Run("can create an empty optional", func(t *testing.T) {
		x := optional.Optional[int]{}
		assert.True(t, x.IsEmpty())
	})
}

func TestOptional_Clear(t *testing.T) {
	t.Run("can clear a set value", func(t *testing.T) {
		x := optional.New(12)
		x.Clear()
		assert.True(t, x.IsEmpty())
	})
	t.Run("can clear an empty value", func(t *testing.T) {
		var x optional.Optional[int]
		x.Clear()
		assert.True(t, x.IsEmpty())
	})
}

func TestOptional_Set(t *testing.T) {
	t.Run("can update an empty optional", func(t *testing.T) {
		x := optional.Optional[int]{}
		x.Set(45)
		xassert.Equal(t, 45, x.MustValue())
	})
	t.Run("can update a non none", func(t *testing.T) {
		x := optional.New(12)
		x.Set(45)
		xassert.Equal(t, 45, x.MustValue())
	})
}

func TestOptional_SetWhenEmpty(t *testing.T) {
	t.Run("sets an empty optional", func(t *testing.T) {
		x := optional.Optional[int]{}
		x.SetWhenEmpty(45)
		xassert.Equal(t, 45, x.ValueOrZero())
	})
	t.Run("does not set a non-empty optional", func(t *testing.T) {
		x := optional.New(12)
		x.SetWhenEmpty(45)
		xassert.Equal(t, 12, x.ValueOrZero())
	})
}

func TestOptional_Print(t *testing.T) {
	t.Run("can print a value", func(t *testing.T) {
		x := optional.New(12)
		s := fmt.Sprint(x)
		xassert.Equal(t, "12", s)
	})
	t.Run("can print an empty optional", func(t *testing.T) {
		x := optional.Optional[int]{}
		s := fmt.Sprint(x)
		xassert.Equal(t, "<empty>", s)
	})
}

func TestOptional_Ptr(t *testing.T) {
	t.Run("can convert a non-empty value", func(t *testing.T) {
		o := optional.New(12)
		x := o.Ptr()
		xassert.Equal(t, *x, 12)
	})
	t.Run("can convert an empty value", func(t *testing.T) {
		o := optional.Optional[int]{}
		x := o.Ptr()
		assert.Nil(t, x)
	})
}

func TestOptional_ValueOrFallback(t *testing.T) {
	t.Run("should return value when set", func(t *testing.T) {
		x := optional.New(12)
		got := x.ValueOrFallback(4)
		xassert.Equal(t, 12, got)
	})
	t.Run("should return fallback when empty", func(t *testing.T) {
		x := optional.Optional[int]{}
		got := x.ValueOrFallback(4)
		xassert.Equal(t, 4, got)
	})
}

func TestOptional_MustValue(t *testing.T) {
	t.Run("should return value when set and not panic", func(t *testing.T) {
		x := optional.New(12)
		got := x.MustValue()
		xassert.Equal(t, 12, got)
	})
	t.Run("should panic when empty", func(t *testing.T) {
		x := optional.Optional[int]{}
		assert.Panics(t, func() {
			x.MustValue()
		})
	})
}

func TestOptional_Value(t *testing.T) {
	t.Run("should return value when set", func(t *testing.T) {
		x := optional.New(12)
		got, ok := x.Value()
		require.True(t, ok)
		xassert.Equal(t, 12, got)
	})
	t.Run("should return error when empty", func(t *testing.T) {
		x := optional.Optional[int]{}
		_, ok := x.Value()
		assert.False(t, ok)
	})
}

func TestOptional_StringFunc(t *testing.T) {
	t.Run("should return converted string when optional has value", func(t *testing.T) {
		x := optional.New(12)
		got := x.StringFunc("", func(v int) string {
			return fmt.Sprint(v)
		})
		xassert.Equal(t, "12", got)
	})
	t.Run("should return fallback when optional is empty", func(t *testing.T) {
		var x optional.Optional[int]
		got := x.StringFunc("x", func(v int) string {
			return fmt.Sprint(v)
		})
		xassert.Equal(t, "x", got)
	})
}

func TestOptional_ValueOrZero(t *testing.T) {
	t.Run("should return value when set", func(t *testing.T) {
		x := optional.New(12)
		got := x.ValueOrZero()
		xassert.Equal(t, 12, got)
	})
	t.Run("should return zero value integer optional is empty", func(t *testing.T) {
		x := optional.Optional[int]{}
		got := x.ValueOrZero()
		xassert.Equal(t, 0, got)
	})
	t.Run("should return zero string value is empty", func(t *testing.T) {
		x := optional.Optional[string]{}
		got := x.ValueOrZero()
		xassert.Equal(t, "", got)
	})
}

func TestOptional_JSON(t *testing.T) {
	t.Run("should marshal and unmarshal a non-empty value", func(t *testing.T) {
		o1 := optional.New(12)
		b, err := json.Marshal(o1)
		require.NoError(t, err)
		var o2 optional.Optional[int]
		err = json.Unmarshal(b, &o2)
		require.NoError(t, err)
		assert.True(t, optional.Equal(o1, o2))
	})
	t.Run("should marshal and unmarshal an empty value", func(t *testing.T) {
		var o1 optional.Optional[int]
		b, err := json.Marshal(o1)
		require.NoError(t, err)
		var o2 optional.Optional[int]
		err = json.Unmarshal(b, &o2)
		require.NoError(t, err)
		assert.True(t, optional.Equal(o1, o2))
	})
}

func TestConvertNumeric(t *testing.T) {
	xassert.Equal(
		t,
		optional.New(int(99)),
		optional.ConvertNumeric[int64, int](optional.New(int64(99))),
	)
	xassert.Equal(
		t,
		optional.New(float64(99)),
		optional.ConvertNumeric[int32, float64](optional.New(int32(99))),
	)
	xassert.Equal(
		t,
		optional.Optional[float64]{},
		optional.ConvertNumeric[int32, float64](optional.Optional[int32]{}),
	)
}

func TestFromZeroValue(t *testing.T) {
	xassert.Equal(t, optional.New(5), optional.FromZeroValue(5))
	xassert.Equal(t, optional.Optional[int]{}, optional.FromZeroValue(0))

	xassert.Equal(t, optional.New(0.5), optional.FromZeroValue(0.5))
	xassert.Equal(t, optional.Optional[float64]{}, optional.FromZeroValue(float64(0)))

	x := time.Now()
	xassert.Equal(t, optional.New(x), optional.FromZeroValue(x))
	xassert.Equal(t, optional.Optional[time.Time]{}, optional.FromZeroValue(time.Time{}))

}

func TestSum(t *testing.T) {
	cases := []struct {
		a, b, want optional.Optional[int]
	}{
		{optional.New(5), optional.New(3), optional.New(8)},
		{optional.New(5), optional.Optional[int]{}, optional.New(5)},
		{optional.Optional[int]{}, optional.New(5), optional.New(5)},
		{optional.Optional[int]{}, optional.Optional[int]{}, optional.Optional[int]{}},
	}
	for _, tc := range cases {
		got := optional.Sum(tc.a, tc.b)
		xassert.Equal(t, tc.want, got)
	}
}

func TestFromPointerOptional(t *testing.T) {
	var x *int

	a := 5
	x = &a
	xassert.Equal(t, optional.New(5), optional.FromPtr(x))

	x = nil
	xassert.Equal(t, optional.Optional[int]{}, optional.FromPtr(x))
}

func TestEqual(t *testing.T) {
	cases := []struct {
		a, b optional.Optional[int]
		want bool
	}{
		{optional.New(5), optional.New(5), true},
		{optional.New(5), optional.New(3), false},
		{optional.New(5), optional.Optional[int]{}, false},
		{optional.Optional[int]{}, optional.Optional[int]{}, true},
	}
	for _, tc := range cases {
		got := optional.Equal(tc.a, tc.b)
		xassert.Equal(t, tc.want, got)
	}
}

func TestEqual2(t *testing.T) {
	v := time.Now()
	cases := []struct {
		a, b optional.Optional[time.Time]
		want bool
	}{
		{optional.New(v), optional.New(v), true},
		{optional.New(v), optional.New(v.Add(1 * time.Hour)), false},
		{optional.New(v), optional.Optional[time.Time]{}, false},
		{optional.Optional[time.Time]{}, optional.Optional[time.Time]{}, true},
	}
	for _, tc := range cases {
		got := optional.Equal2(tc.a, tc.b)
		xassert.Equal(t, tc.want, got)
	}
}

func TestEqualFunc(t *testing.T) {
	cases := []struct {
		a, b optional.Optional[int]
		want bool
	}{
		{optional.New(5), optional.New(5), true},
		{optional.New(5), optional.New(3), false},
		{optional.New(5), optional.Optional[int]{}, false},
		{optional.Optional[int]{}, optional.Optional[int]{}, true},
	}
	for _, tc := range cases {
		got := optional.EqualFunc(tc.a, tc.b, func(a, b int) bool {
			return a == b
		})
		xassert.Equal(t, tc.want, got)
	}
}

func TestMapOrFallback(t *testing.T) {
	t.Run("should return applied value when set", func(t *testing.T) {
		x := optional.New(12)
		got := optional.MapOrFallback(x, "nope", func(x int) string {
			return fmt.Sprint(x)
		})
		xassert.Equal(t, "12", got)
	})
	t.Run("should return fallback when empty", func(t *testing.T) {
		x := optional.Optional[int]{}
		got := optional.MapOrFallback(x, "nope", func(x int) string {
			return fmt.Sprint(x)
		})
		xassert.Equal(t, "nope", got)
	})
}

func TestMapOrFallbackFunc(t *testing.T) {
	t.Run("should return applied value when set", func(t *testing.T) {
		x := optional.New(12)
		got := optional.MapOrFallbackFunc(x,
			func() string {
				return "nope"
			},
			func(x int) string {
				return fmt.Sprint(x)
			},
		)
		xassert.Equal(t, "12", got)
	})
	t.Run("should return fallback when empty", func(t *testing.T) {
		x := optional.Optional[int]{}
		got := optional.MapOrFallbackFunc(x,
			func() string {
				return "nope"
			},
			func(x int) string {
				return fmt.Sprint(x)
			},
		)
		xassert.Equal(t, "nope", got)
	})
}

func TestMapOrZero(t *testing.T) {
	t.Run("should return applied value when set", func(t *testing.T) {
		x := optional.New(12)
		got := optional.MapOrZero(x, func(x int) string {
			return fmt.Sprint(x)
		})
		xassert.Equal(t, "12", got)
	})
	t.Run("should return fallback when empty", func(t *testing.T) {
		x := optional.Optional[int]{}
		got := optional.MapOrZero(x, func(x int) string {
			return fmt.Sprint(x)
		})
		xassert.Equal(t, "", got)
	})
}

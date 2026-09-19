package settings

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func newTestSettings(t *testing.T) *Settings {
	t.Helper()
	_, st, _ := testutil.NewDBInMemory()
	s, err := New(context.Background(), st)
	require.NoError(t, err)
	return s
}

func TestGetSet(t *testing.T) {
	t.Run("returns false when key is absent", func(t *testing.T) {
		s := newTestSettings(t)
		v, ok := s.get("nope")
		assert.False(t, ok)
		assert.Equal(t, "", v)
	})
	t.Run("returns stored value and true when key is present", func(t *testing.T) {
		s := newTestSettings(t)
		s.set("x", "hello")
		v, ok := s.get("x")
		assert.True(t, ok)
		assert.Equal(t, "hello", v)
	})
	t.Run("persists the value to storage", func(t *testing.T) {
		s := newTestSettings(t)
		s.set("x", "hello")
		s.Flush()
		v, err := s.st.GetSetting(context.Background(), "x")
		require.NoError(t, err)
		assert.Equal(t, "hello", v)
	})
	t.Run("a persisted value is visible to a freshly loaded Settings instance", func(t *testing.T) {
		_, st, _ := testutil.NewDBInMemory()
		s1, err := New(context.Background(), st)
		require.NoError(t, err)
		s1.set("x", "hello")
		s1.Flush()

		s2, err := New(context.Background(), st)
		require.NoError(t, err)
		v, ok := s2.get("x")
		assert.True(t, ok)
		assert.Equal(t, "hello", v)
	})
}

func TestGetSetString(t *testing.T) {
	t.Run("returns fallback when absent", func(t *testing.T) {
		s := newTestSettings(t)
		assert.Equal(t, "fallback", s.getString("x", "fallback"))
	})
	t.Run("round trip", func(t *testing.T) {
		s := newTestSettings(t)
		s.setString("x", "hello")
		assert.Equal(t, "hello", s.getString("x", "fallback"))
	})
	t.Run("an explicitly stored empty string is not the same as absent", func(t *testing.T) {
		s := newTestSettings(t)
		s.setString("x", "")
		assert.Equal(t, "", s.getString("x", "fallback"))
	})
}

func TestGetSetBool(t *testing.T) {
	t.Run("returns fallback when absent", func(t *testing.T) {
		s := newTestSettings(t)
		assert.True(t, s.getBool("x", true))
		assert.False(t, s.getBool("x", false))
	})
	t.Run("round trip true and false", func(t *testing.T) {
		s := newTestSettings(t)
		s.setBool("x", true)
		assert.True(t, s.getBool("x", false))
		s.setBool("x", false)
		assert.False(t, s.getBool("x", true))
	})
	t.Run("returns fallback when stored value is not a valid bool", func(t *testing.T) {
		s := newTestSettings(t)
		s.set("x", "not-a-bool")
		assert.True(t, s.getBool("x", true))
	})
}

func TestGetSetInt(t *testing.T) {
	t.Run("returns fallback when absent", func(t *testing.T) {
		s := newTestSettings(t)
		assert.Equal(t, 42, s.getInt("x", 42))
	})
	t.Run("round trip positive and negative", func(t *testing.T) {
		s := newTestSettings(t)
		s.setInt("x", 123)
		assert.Equal(t, 123, s.getInt("x", 0))
		s.setInt("x", -5)
		assert.Equal(t, -5, s.getInt("x", 0))
	})
	t.Run("returns fallback when stored value is not a valid int", func(t *testing.T) {
		s := newTestSettings(t)
		s.set("x", "not-an-int")
		assert.Equal(t, 42, s.getInt("x", 42))
	})
}

func TestGetSetInt64(t *testing.T) {
	t.Run("returns fallback when absent", func(t *testing.T) {
		s := newTestSettings(t)
		assert.Equal(t, int64(42), s.getInt64("x", 42))
	})
	t.Run("round trip positive and negative", func(t *testing.T) {
		s := newTestSettings(t)
		s.setInt64("x", 123)
		assert.Equal(t, int64(123), s.getInt64("x", 0))
		s.setInt64("x", -5)
		assert.Equal(t, int64(-5), s.getInt64("x", 0))
	})
	t.Run("round trip a value beyond int32 range", func(t *testing.T) {
		s := newTestSettings(t)
		x := int64(2_200_000_000) // beyond int32 range, e.g. a valid EVE ID
		s.setInt64("x", x)
		assert.Equal(t, x, s.getInt64("x", 0))
	})
	t.Run("returns fallback when stored value is not a valid int64", func(t *testing.T) {
		s := newTestSettings(t)
		s.set("x", "not-an-int")
		assert.Equal(t, int64(42), s.getInt64("x", 42))
	})
}

func TestGetSetFloat(t *testing.T) {
	t.Run("returns fallback when absent", func(t *testing.T) {
		s := newTestSettings(t)
		assert.Equal(t, 1.5, s.getFloat("x", 1.5))
	})
	t.Run("round trip", func(t *testing.T) {
		s := newTestSettings(t)
		s.setFloat("x", 2.75)
		assert.Equal(t, 2.75, s.getFloat("x", 0))
	})
	t.Run("returns fallback when stored value is not a valid float", func(t *testing.T) {
		s := newTestSettings(t)
		s.set("x", "not-a-float")
		assert.Equal(t, 1.5, s.getFloat("x", 1.5))
	})
}

func TestGetSetTime(t *testing.T) {
	t.Run("returns fallback when absent", func(t *testing.T) {
		s := newTestSettings(t)
		fallback := time.Now().UTC().Truncate(time.Second)
		assert.True(t, fallback.Equal(s.getTime("x", fallback)))
	})
	t.Run("returns zero fallback when absent", func(t *testing.T) {
		s := newTestSettings(t)
		assert.True(t, s.getTime("x", time.Time{}).IsZero())
	})
	t.Run("round trip", func(t *testing.T) {
		s := newTestSettings(t)
		want := time.Now().UTC().Truncate(time.Second)
		s.setTime("x", want)
		assert.True(t, want.Equal(s.getTime("x", time.Time{})))
	})
	t.Run("returns fallback when stored value is not a valid time", func(t *testing.T) {
		s := newTestSettings(t)
		s.set("x", "not-a-time")
		fallback := time.Now().UTC().Truncate(time.Second)
		assert.True(t, fallback.Equal(s.getTime("x", fallback)))
	})
}

func TestGetSetStringList(t *testing.T) {
	t.Run("returns fallback when absent", func(t *testing.T) {
		s := newTestSettings(t)
		fallback := []string{"a"}
		assert.Equal(t, fallback, s.getStringList("x", fallback))
	})
	t.Run("round trip", func(t *testing.T) {
		s := newTestSettings(t)
		want := []string{"alpha", "bravo"}
		s.setStringList("x", want)
		assert.Equal(t, want, s.getStringList("x", nil))
	})
	t.Run("round trip empty slice", func(t *testing.T) {
		s := newTestSettings(t)
		s.setStringList("x", []string{})
		assert.Equal(t, []string{}, s.getStringList("x", []string{"fallback"}))
	})
	t.Run("returns fallback when stored value is not valid JSON", func(t *testing.T) {
		s := newTestSettings(t)
		s.set("x", "not-json")
		fallback := []string{"a"}
		assert.Equal(t, fallback, s.getStringList("x", fallback))
	})
}

func TestGetSetFloatList(t *testing.T) {
	t.Run("returns fallback when absent", func(t *testing.T) {
		s := newTestSettings(t)
		fallback := []float64{1}
		assert.Equal(t, fallback, s.getFloatList("x", fallback))
	})
	t.Run("round trip", func(t *testing.T) {
		s := newTestSettings(t)
		want := []float64{1000, 600}
		s.setFloatList("x", want)
		assert.Equal(t, want, s.getFloatList("x", nil))
	})
	t.Run("returns fallback when stored value is not valid JSON", func(t *testing.T) {
		s := newTestSettings(t)
		s.set("x", "not-json")
		fallback := []float64{1}
		assert.Equal(t, fallback, s.getFloatList("x", fallback))
	})
	t.Run("does not update the cache when the value cannot be marshaled", func(t *testing.T) {
		s := newTestSettings(t)
		s.setFloatList("x", []float64{1, 2})
		s.setFloatList("x", []float64{math.NaN()})
		assert.Equal(t, []float64{1, 2}, s.getFloatList("x", nil))
	})
}

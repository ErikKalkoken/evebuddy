package humanize_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestNumber1(t *testing.T) {
	var cases = []struct {
		value    float64
		decimals int
		want     string
	}{
		{99, 2, "99.00"},
		{42.1234, 2, "42.12"},
		{1000, 2, "1.00 K"},
		{1234.56, 2, "1.23 K"},
		{1234.56, 0, "1 K"},
		{1234000.56, 2, "1.23 M"},
		{1234000000.56, 2, "1.23 B"},
		{1234000000000.56, 2, "1.23 T"},
		{-1234000.56, 2, "-1.23 M"},
		{0, 2, "0.00"},
		{1234.56, 3, "1.235 K"},
		{1234.56, 1, "1.2 K"},
		{1234.56, 0, "1 K"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("Can format numbers: %f", tc.value), func(t *testing.T) {
			got := humanize.Number(tc.value, tc.decimals)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("should panic when called with undefined decimals", func(t *testing.T) {
		assert.Panics(t, func() {
			humanize.Number(99, 7)
		})
	})
}

func TestDuration(t *testing.T) {
	var cases = []struct {
		name string
		in   time.Duration
		want string
	}{
		{"days and hours", 24*3*time.Hour + 5*time.Hour + 3*time.Minute, "3d 5h"},
		{"days and hours 2", 24*10*time.Hour + 5*time.Hour + 3*time.Minute, "10d 5h"},
		{"hours and minutes", 5*time.Hour + 3*time.Minute, "5h 3m"},
		{"below 1 minute", 59 * time.Second, "<1m"},
		{"negative duration", -5*time.Hour - 3*time.Minute, "0m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := humanize.Duration(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRomanLetters(t *testing.T) {
	var cases = []struct {
		value int
		want  string
	}{
		{1, "I"},
		{2, "II"},
		{3, "III"},
		{4, "IV"},
		{5, "V"},
		{5, "V"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("Can return correct roman letter: %d", tc.value), func(t *testing.T) {
			got := humanize.RomanLetter(tc.value)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("should panic when called for undefined numbers", func(t *testing.T) {
		assert.Panics(t, func() {
			humanize.RomanLetter(99)
		})
	})
}

func TestOptional(t *testing.T) {
	t.Run("fallback when empty", func(t *testing.T) {
		assert.Equal(t, "fallback", humanize.Optional(optional.Optional[int]{}, "fallback"))
	})
	t.Run("time", func(t *testing.T) {
		x := time.Now().Add(5 * time.Minute)
		assert.Equal(t, "0h 5m", humanize.Optional(optional.New(x), ""))
	})
	t.Run("time", func(t *testing.T) {
		x := 5 * time.Minute
		assert.Equal(t, "0h 5m", humanize.Optional(optional.New(x), ""))
	})
	t.Run("string", func(t *testing.T) {
		assert.Equal(t, "alpha", humanize.Optional(optional.New("alpha"), ""))
	})
	t.Run("number", func(t *testing.T) {
		assert.Equal(t, "42", humanize.Optional(optional.New(int(42)), ""))
		assert.Equal(t, "42", humanize.Optional(optional.New(int32(42)), ""))
		assert.Equal(t, "42", humanize.Optional(optional.New(int64(42)), ""))
	})
	t.Run("bool", func(t *testing.T) {
		assert.Equal(t, "yes", humanize.Optional(optional.New(true), ""))
		assert.Equal(t, "no", humanize.Optional(optional.New(false), ""))
	})
	t.Run("other", func(t *testing.T) {
		x := []int{1, 2, 3}
		assert.Equal(t, "[1 2 3]", humanize.Optional(optional.New(x), ""))
	})
}

func TestOptionalWithComma(t *testing.T) {
	assert.Equal(t, "fallback", humanize.OptionalWithComma(optional.Optional[int]{}, "fallback"))
	assert.Equal(t, "1,234", humanize.OptionalWithComma(optional.New(1234), ""))
}

func TestOptionalWithDecemals(t *testing.T) {
	assert.Equal(t, "fallback", humanize.OptionalWithDecimals(optional.Optional[float64]{}, 1, "fallback"))
	assert.Equal(t, "1.2", humanize.OptionalWithDecimals(optional.New(float64(1.23)), 1, ""))
	assert.Equal(t, "1.2", humanize.OptionalWithDecimals(optional.New(float32(1.23)), 1, ""))
}

func TestComma(t *testing.T) {
	assert.Equal(t, "1,234", humanize.Comma(int(1234)))
	assert.Equal(t, "1,234", humanize.Comma(int32(1234)))
	assert.Equal(t, "1,234", humanize.Comma(int64(1234)))
	assert.Equal(t, "1,234", humanize.Comma(uint(1234)))
	assert.Equal(t, "1,234", humanize.Comma(uint32(1234)))
	assert.Equal(t, "1,234", humanize.Comma(uint64(1234)))
}

func TestTimeWithFallback(t *testing.T) {
	x := time.Now().Add(-21 * 24 * time.Hour)
	assert.Equal(t, "3 weeks ago", humanize.TimeWithFallback(x, ""))
	assert.Equal(t, "fallback", humanize.TimeWithFallback(time.Time{}, "fallback"))
}

func TestRelTime(t *testing.T) {
	var cases = []struct {
		name string
		in   time.Time
		want string
	}{
		{"future", time.Now().Add(3 * time.Minute), "0h 3m"},
		{"past", time.Now().Add(-3 * time.Minute), "0h 3m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := humanize.RelTime(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

// Package humanize transforms values into more user friendly representations.
package humanize

import (
	"fmt"
	"math"
	"time"

	"github.com/dustin/go-humanize"
	"golang.org/x/exp/constraints"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// Number returns a humanized number, e.g. 1234 becomes 1.23K
func Number(value float64, decimals int) string {
	var s int
	var a string
	v2 := math.Abs(value)
	switch {
	case v2 >= 1_000_000_000_000:
		s = 12
		a = " T"
	case v2 >= 1_000_000_000:
		s = 9
		a = " B"
	case v2 >= 1_000_000:
		s = 6
		a = " M"
	case v2 >= 1_000:
		s = 3
		a = " K"
	default:
		s = 0
		a = ""
	}
	x := value / math.Pow10(s)
	var f string
	switch {
	case decimals == 3:
		f = "%.3f"
	case decimals == 2:
		f = "%.2f"
	case decimals == 1:
		f = "%.1f"
	case decimals == 0:
		f = "%.0f"
	default:
		panic(fmt.Sprintf("Undefined decimals: %d", decimals))
	}
	r := fmt.Sprintf(f, x) + a
	return r
}

// Duration returns a humanized duration, e.g. "10h 5m".
//
// Shows days and hours for duration over 1 day, else hours and minutes.
// Rounds to full minutes.
// Negative durations are returned as "0 m"
func Duration(duration time.Duration) string {
	if duration < 0 {
		return "0m"
	}
	mRaw := duration.Abs().Minutes()
	if mRaw < 1 {
		return "<1m"
	}
	m := int(math.Round(mRaw))
	w := m / 60 / 24 / 7
	m -= w * 60 * 24 * 7
	d := m / 60 / 24
	m -= d * 60 * 24
	h := m / 60
	m -= h * 60
	if w > 0 {
		return fmt.Sprintf("%dw %dd %dh", w, d, h)
	} else if d > 0 {
		return fmt.Sprintf("%dd %dh", d, h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

// RelTime returns the duration until a time in the future.
func RelTime(t time.Time) string {
	return Duration(time.Until(t))
}

// Comma produces a string form of the given number in base 10
// with commas after every three orders of magnitude.
// This is a variation of Comma from the external humanize package,
// that works with any integer like type.
func Comma[T constraints.Integer](x T) string {
	return humanize.Comma(int64(x))
}

// Optional returns a string representation of on optional value when set
// or the fallback when not set.
func Optional[T any](o optional.Optional[T], fallback string) string {
	if o.IsEmpty() {
		return fallback
	}
	v := o.ValueOrZero()
	switch x := any(v).(type) {
	case time.Duration:
		return Duration(x)
	case time.Time:
		return RelTime(x)
	case string:
		return x
	case int:
		return Number(float64(x), 0)
	case int32:
		return Number(float64(x), 0)
	case int64:
		return Number(float64(x), 0)
	case bool:
		if x {
			return "yes"
		}
		return "no"
	}
	return fmt.Sprint(v)
}

func OptionalWithComma[T constraints.Integer](o optional.Optional[T], fallback string) string {
	if o.IsEmpty() {
		return fallback
	}
	return humanize.Comma(int64(o.ValueOrZero()))
}

func OptionalWithDecimals[T float32 | float64](o optional.Optional[T], decimals int, fallback string) string {
	if o.IsEmpty() {
		return fallback
	}
	return Number(float64(o.ValueOrZero()), decimals)
}

// RomanLetter returns a number as roman letters.
func RomanLetter[T constraints.Integer](v T) string {
	m := map[int]string{
		1: "I",
		2: "II",
		3: "III",
		4: "IV",
		5: "V",
	}
	r, ok := m[int(v)]
	if !ok {
		panic(fmt.Sprintf("invalid value: %d", v))
	}
	return r
}

// TimeWithFallback returns a given time as relative string.
// Or returns the fallback when time is zero.
func TimeWithFallback(v time.Time, fallback string) string {
	if v.IsZero() {
		return fallback
	}
	return humanize.Time(v)
}

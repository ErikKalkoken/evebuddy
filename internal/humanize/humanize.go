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

// NumberF returns a humanized float number, e.g. 1234 becomes 1.23K
func NumberF[T constraints.Float](value T, decimals uint) string {
	return number(float64(value), decimals)
}

// Number returns a humanized int number, e.g. 1234 becomes 1.23K
func Number[T constraints.Integer](value T, decimals uint) string {
	if v := int(value); v > -1000 && v < 1000 {
		return fmt.Sprint(value)
	}
	return number(float64(value), decimals)
}

func number(value float64, decimals uint) string {
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
	f := fmt.Sprintf("%%.%df%%s", decimals)
	x := value / math.Pow10(s)
	r := fmt.Sprintf(f, x, a)
	return r
}

// Duration returns a humanized duration, e.g. "10h 5m".
//
// Shows days and hours for duration over 1 day, else hours and minutes.
// Rounds up to full minutes / full hours.
// Negative durations are returned as "0m"
func Duration(d time.Duration) string {
	return duration(d, false)
}

// DurationRoundedUp returns a humanized duration similar to Duration, but always rounds up.
func DurationRoundedUp(d time.Duration) string {
	return duration(d, true)
}

func duration(d time.Duration, roundUp bool) string {
	if d <= 0 {
		return "0m"
	}
	minutesFloat := d.Abs().Minutes()
	if minutesFloat < 1 {
		return "<1m"
	}
	var minutes int
	if roundUp {
		minutes = int(math.Ceil(minutesFloat))
	} else {
		minutes = int(math.Round(minutesFloat))
	}
	days := minutes / 60 / 24
	minutes -= days * 60 * 24
	hours := minutes / 60
	minutes -= hours * 60
	if days > 0 {
		if minutes > 30 || (roundUp && minutes > 0) {
			hours++
		}
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// RelTime returns the duration until a time in the future.
func RelTime(t time.Time) string {
	d := time.Until(t)
	if d < 0 {
		d = -1 * d
	}
	return Duration(d)
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
	v, ok := o.Value()
	if !ok {
		return fallback
	}
	switch x := any(v).(type) {
	case time.Duration:
		return Duration(x)
	case time.Time:
		return RelTime(x)
	case string:
		return x
	case int:
		return NumberF(float64(x), 0)
	case int64:
		return NumberF(float64(x), 0)
	case bool:
		if x {
			return "yes"
		}
		return "no"
	}
	return fmt.Sprint(v)
}

func OptionalWithComma[T constraints.Integer](o optional.Optional[T], fallback string) string {
	v, ok := o.Value()
	if !ok {
		return fallback
	}
	return humanize.Comma(int64(v))
}

func OptionalWithDecimals[T float32 | float64](o optional.Optional[T], decimals uint, fallback string) string {
	v, ok := o.Value()
	if !ok {
		return fallback
	}
	return NumberF(float64(v), decimals)
}

// RomanLetter returns a number as roman letters.
// Returns an empty string if v can not be resolved.
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
		return ""
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

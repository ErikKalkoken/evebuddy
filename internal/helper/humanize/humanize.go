// Package humanize provides helper functions to humanize values.
package humanize

import (
	"fmt"
	"math"
)

// Number returns a humanized number, e.g. 1234 becomes 1.23K
func Number(value float64, decimals int) string {
	var s int
	var a string
	v2 := math.Abs(value)
	switch {
	case v2 >= 1000000000000:
		s = 12
		a = "T"
	case v2 >= 1000000000:
		s = 9
		a = "B"
	case v2 >= 1000000:
		s = 6
		a = "M"
	case v2 >= 1000:
		s = 3
		a = "K"
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

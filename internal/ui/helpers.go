package ui

import (
	"time"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
)

func stringOrDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func timeFormattedOrDefault(t time.Time, layout, d string) string {
	if t.IsZero() {
		return d
	}
	return t.Format(layout)
}

func numberOrDefault[T int | float64](v T, d string) string {
	if v == 0 {
		return d
	}
	return ihumanize.Number(float64(v), 1)
}

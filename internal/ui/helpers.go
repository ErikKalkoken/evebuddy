package ui

import (
	"time"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
)

// func stringOrFallback(s, fallback string) string {
// 	if s == "" {
// 		return fallback
// 	}
// 	return s
// }

func timeFormattedOrFallback(t time.Time, layout, fallback string) string {
	if t.IsZero() {
		return fallback
	}
	return t.Format(layout)
}

// func numberOrDefault[T int | float64](v T, d string) string {
// 	if v == 0 {
// 		return d
// 	}
// 	return ihumanize.Number(float64(v), 1)
// }

func humanizedNullDuration(d types.NullDuration, fallback string) string {
	if !d.Valid {
		return fallback
	}
	return ihumanize.Duration(d.Duration)
}

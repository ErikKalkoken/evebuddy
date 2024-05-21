package ui

import (
	"database/sql"
	"fmt"
	"time"

	"fyne.io/fyne/v2/data/binding"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
	"github.com/dustin/go-humanize"
)

func nullStringOrFallback(s sql.NullString, fallback string) string {
	if !s.Valid {
		return fallback
	}
	return s.String
}

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

func humanizedRelNullTime(v sql.NullTime, fallback string) string {
	if !v.Valid {
		return fallback
	}
	return humanize.RelTime(v.Time, time.Now(), "", "")
}

func humanizedNullTime(v sql.NullTime, fallback string) string {
	if !v.Valid {
		return fallback
	}
	return humanize.Time(v.Time)
}

func humanizedNullFloat64(v sql.NullFloat64, decimals int, fallback string) string {
	if !v.Valid {
		return fallback
	}
	return ihumanize.Number(v.Float64, decimals)
}

func humanizedNullInt64(v sql.NullInt64, fallback string) string {
	return humanizedNullFloat64(sql.NullFloat64{Float64: float64(v.Int64), Valid: v.Valid}, 0, fallback)
}

// getFromBoundUntypedList returns the value from an untyped list in the target type.
func getFromBoundUntypedList[T any](l binding.UntypedList, index int) (T, error) {
	var z T
	xx, err := l.GetItem(index)
	if err != nil {
		return z, err
	}
	x, err := xx.(binding.Untyped).Get()
	if err != nil {
		return z, err
	}
	c := x.(T)
	return c, nil
}

// convertDataItem returns the value of the data item in the target type.
func convertDataItem[T any](i binding.DataItem) (T, error) {
	var z T
	x, err := i.(binding.Untyped).Get()
	if err != nil {
		return z, err
	}
	q, ok := x.(T)
	if !ok {
		return z, fmt.Errorf("failed to convert untyped to %T", z)
	}
	return q, nil

}

// copyToUntypedSlice copies a slice of any type into an untyped slice.
func copyToUntypedSlice[T any](s []T) []any {
	x := make([]any, len(s))
	for i, v := range s {
		x[i] = v
	}
	return x
}

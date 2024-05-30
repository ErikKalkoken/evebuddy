package ui

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
	"github.com/dustin/go-humanize"
)

// newObjectFromJSON returns a new object unmarshaled from a JSON string.
func newObjectFromJSON[T any](s string) (T, error) {
	var n T
	err := json.Unmarshal([]byte(s), &n)
	if err != nil {
		return n, err
	}
	return n, nil
}

// objectToJSON returns a JSON string marshaled from the given object.
func objectToJSON[T any](o T) (string, error) {
	s, err := json.Marshal(o)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

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

// func humanizedNullTime(v sql.NullTime, fallback string) string {
// 	if !v.Valid {
// 		return fallback
// 	}
// 	return humanize.Time(v.Time)
// }

func humanizeTime(v time.Time, fallback string) string {
	if v.IsZero() {
		return fallback
	}
	return humanize.Time(v)
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

// getItemUntypedList returns the value from an untyped list in the target type.
func getItemUntypedList[T any](l binding.UntypedList, index int) (T, error) {
	var v T
	xx, err := l.GetItem(index)
	if err != nil {
		return v, err
	}
	x, err := xx.(binding.Untyped).Get()
	if err != nil {
		return v, err
	}
	v = x.(T)
	return v, nil
}

// func getUntypedList[T any](l binding.UntypedList) ([]T, error) {
// 	xx, err := l.Get()
// 	if err != nil {
// 		return nil, err
// 	}
// 	v := make([]T, len(xx))
// 	for i, x := range xx {
// 		v[i] = x.(T)
// 	}
// 	return v, nil
// }

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

// // newImageResourceAsync shows a placeholder resource and refreshes it once the main resource is loaded asynchronously.
// func newImageResourceAsync(placeholder fyne.Resource, loader func() (fyne.Resource, error)) *canvas.Image {
// 	image := canvas.NewImageFromResource(placeholder)
// 	refreshImageResourceAsync(image, loader)
// 	return image
// }

// refreshImageResourceAsync refreshes the resource of an image asynchronously.
// This prevents fyne to wait with rendering an image until a resource is fully loaded from a web server.
func refreshImageResourceAsync(image *canvas.Image, loader func() (fyne.Resource, error)) {
	go func(*canvas.Image) {
		r, err := loader()
		if err != nil {
			slog.Warn("Failed to fetch image resource", "err", err)
		} else {
			image.Resource = r
			image.Refresh()
		}
	}(image)
}

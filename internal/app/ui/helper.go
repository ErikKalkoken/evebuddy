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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
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

// func objectToJSONOrPanic[T any](o T) string {
// 	s, err := objectToJSON(o)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return s
// }

func entityNameOrFallback[T int | int32 | int64](e *app.EntityShort[T], fallback string) string {
	if e == nil {
		return fallback
	}
	return e.Name
}

// func nullStringOrFallback(s sql.NullString, fallback string) string {
// 	if !s.Valid {
// 		return fallback
// 	}
// 	return s.String
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

// func humanizedNullDuration(d optional.Duration, fallback string) string {
// 	if !d.Valid {
// 		return fallback
// 	}
// 	return ihumanize.Duration(d.Duration)
// }

// func humanizedRelNullTime(v sql.NullTime, fallback string) string {
// 	if !v.Valid {
// 		return fallback
// 	}
// 	return humanize.RelTime(v.Time, time.Now(), "", "")
// }

func humanizedRelOptionTime(v optional.Optional[time.Time], fallback string) string {
	if v.IsNone() {
		return fallback
	}
	return humanize.RelTime(v.MustValue(), time.Now(), "", "")
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

type numeric interface {
	int | int32 | int64 | float32 | float64
}

func humanizedNumericOption[T numeric](v optional.Optional[T], decimals int, fallback string) string {
	if v.IsNone() {
		return fallback
	}
	return ihumanize.Number(float64(v.MustValue()), decimals)
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

// convertDataItem returns the value of an untyped data item in the target type.
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

// newImageResourceAsync shows a placeholder resource and refreshes it once the main resource is loaded asynchronously.
func newImageResourceAsync(placeholder fyne.Resource, loader func() (fyne.Resource, error)) *canvas.Image {
	image := canvas.NewImageFromResource(placeholder)
	refreshImageResourceAsync(image, loader)
	return image
}

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

// treeNodeFromBoundTree fetches and returns a tree node from a string tree.
func treeNodeFromBoundTree[T any](data binding.StringTree, uid widget.TreeNodeID) (T, error) {
	var zero T
	v, err := data.GetValue(uid)
	if err != nil {
		return zero, fmt.Errorf("failed to get tree node: %w", err)
	}
	n, err := newObjectFromJSON[T](v)
	if err != nil {
		return zero, fmt.Errorf("failed to unmarshal tree node: %w", err)
	}
	return n, nil
}

// treeNodeFromDataItem fetches a tree node from a data item and returns it.
func treeNodeFromDataItem[T any](di binding.DataItem) (T, error) {
	var zero T
	v, err := di.(binding.String).Get()
	if err != nil {
		return zero, err
	}
	n, err := newObjectFromJSON[T](v)
	if err != nil {
		return zero, err
	}
	return n, nil
}

func skillDisplayName[N int | int32 | int64 | uint | uint32 | uint64](name string, level N) string {
	return fmt.Sprintf("%s %s", name, ihumanize.ToRomanLetter(level))
}

func boolIconResource(ok bool) fyne.Resource {
	if ok {
		return theme.NewSuccessThemedResource(theme.ConfirmIcon())
	}
	return theme.NewErrorThemedResource(theme.CancelIcon())
}

func systemSecurity2Importance(t app.SolarSystemSecurityType) widget.Importance {
	switch t {
	case app.SuperHighSec:
		return widget.HighImportance
	case app.HighSec:
		return widget.SuccessImportance
	case app.LowSec:
		return widget.WarningImportance
	case app.NullSec:
		return widget.DangerImportance
	}
	panic("Invalid security")
}

func status2widgetImportance(s app.Status) widget.Importance {
	m := map[app.Status]widget.Importance{
		app.StatusError:   widget.DangerImportance,
		app.StatusMissing: widget.WarningImportance,
		app.StatusOK:      widget.MediumImportance,
		app.StatusUnknown: widget.LowImportance,
		app.StatusWorking: widget.MediumImportance,
	}
	i, ok := m[s]
	if !ok {
		i = widget.MediumImportance
	}
	return i
}

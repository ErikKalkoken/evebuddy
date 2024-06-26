package ui

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/app/humanize"
)

func entityNameOrFallback[T int | int32 | int64](e *app.EntityShort[T], fallback string) string {
	if e == nil {
		return fallback
	}
	return e.Name
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

func skillDisplayName[N int | int32 | int64 | uint | uint32 | uint64](name string, level N) string {
	return fmt.Sprintf("%s %s", name, ihumanize.RomanLetter(level))
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

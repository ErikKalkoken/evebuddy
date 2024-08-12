package ui

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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

package shared

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// NewImageResourceAsync shows a placeholder resource and refreshes it once the main resource is loaded asynchronously.
func NewImageResourceAsync(placeholder fyne.Resource, minSize fyne.Size, loader func() (fyne.Resource, error)) *canvas.Image {
	image := iwidget.NewImageFromResource(placeholder, minSize)
	RefreshImageResourceAsync(image, loader)
	return image
}

// RefreshImageResourceAsync refreshes the resource of an image asynchronously.
// This prevents fyne to wait with rendering an image until a resource is fully loaded from a web server.
func RefreshImageResourceAsync(image *canvas.Image, loader func() (fyne.Resource, error)) {
	go func() {
		r, err := loader()
		if err != nil {
			slog.Warn("Failed to fetch image resource", "err", err)
			r = theme.BrokenImageIcon()
		}
		image.Resource = r
		image.Refresh()
	}()
}

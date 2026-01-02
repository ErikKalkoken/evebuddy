package widget

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
)

// DefaultImageScaleMode for images
var DefaultImageScaleMode canvas.ImageScale

// NewImageFromResource creates an canvas.Image with defaults.
func NewImageFromResource(res fyne.Resource, minSize fyne.Size) *canvas.Image {
	x := canvas.NewImageFromResource(res)
	x.FillMode = canvas.ImageFillContain
	x.ScaleMode = DefaultImageScaleMode
	x.CornerRadius = theme.InputRadiusSize()
	x.SetMinSize(minSize)
	return x
}

// NewImageWithLoader shows a placeholder resource and refreshes it once the main resource is loaded asynchronously.
func NewImageWithLoader(placeholder fyne.Resource, minSize fyne.Size, loader func() (fyne.Resource, error)) *canvas.Image {
	image := NewImageFromResource(placeholder, minSize)
	RefreshImageAsync(image, loader)
	return image
}

// RefreshImageAsync refreshes the resource of an image asynchronously.
// This prevents fyne to wait with rendering an image until a resource is fully loaded from a web server.
func RefreshImageAsync(image *canvas.Image, loader func() (fyne.Resource, error)) {
	go func() {
		r, err := loader()
		if err != nil {
			slog.Warn("Failed to fetch image resource", "err", err)
			r = theme.BrokenImageIcon()
		}
		fyne.Do(func() {
			image.Resource = r
			image.Refresh()
		})
	}()
}

// RefreshTappableImageAsync refreshes the resource of an image asynchronously.
// This prevents fyne to wait with rendering an image until a resource is fully loaded from a web server.
func RefreshTappableImageAsync(image *TappableImage, loader func() (fyne.Resource, error)) {
	go func() {
		r, err := loader()
		if err != nil {
			slog.Warn("Failed to fetch image resource", "err", err)
			r = theme.BrokenImageIcon()
		}
		fyne.Do(func() {
			image.SetResource(r)
		})
	}()
}

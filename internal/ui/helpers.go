package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
)

func stringOrDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// func timeFormattedOrDefault(t time.Time, layout, d string) string {
// 	if t.IsZero() {
// 		return d
// 	}
// 	return t.Format(layout)
// }

func numberOrDefault[T int | float64](v T, d string) string {
	if v == 0 {
		return d
	}
	return ihumanize.Number(float64(v), 1)
}

// setIconFromURI sets the icon with a resource loaded from a URI.
// This should usually be called async to reduce draw delays
func setIconFromURI(icon *widget.Icon, uri fyne.URI) {
	image := canvas.NewImageFromURI(uri)
	icon.SetResource(image.Resource)
}

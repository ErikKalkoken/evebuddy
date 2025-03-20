package cross

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// MakeTopLabel returns a new empty label meant for the top bar on a screen.
func MakeTopLabel() *widget.Label {
	l := widget.NewLabel("")
	l.TextStyle.Bold = true
	l.Wrapping = fyne.TextWrapWord
	return l
}

func EntityNameOrFallback[T int | int32 | int64](e *app.EntityShort[T], fallback string) string {
	if e == nil {
		return fallback
	}
	return e.Name
}

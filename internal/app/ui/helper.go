package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func newSpacer(s fyne.Size) fyne.CanvasObject {
	w := canvas.NewRectangle(color.Transparent)
	w.SetMinSize(s)
	return w
}

func newStandardSpacer() fyne.CanvasObject {
	return newSpacer(fyne.NewSquareSize(theme.Padding()))
}

// characterIDOrZero returns the ID of a character or 0 if the c does not exist.
func characterIDOrZero(c *app.Character) int64 {
	if c == nil {
		return 0
	}
	return c.ID
}

// corporationIDOrZero returns the ID of a corporation or 0 if the c does not exist.
func corporationIDOrZero(c *app.Corporation) int64 {
	if c == nil {
		return 0
	}
	return c.ID
}

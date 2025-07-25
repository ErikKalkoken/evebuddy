package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type detailWindowParams struct {
	content fyne.CanvasObject
	image   fyne.Resource
	minSize fyne.Size
	title   string
	window  fyne.Window
}

// setDetailWindow sets the content of a window to create a "detail window".
// Detail windows are used to show more information about objects in data lists.
func setDetailWindow(arg detailWindowParams) {
	if arg.window == nil {
		panic("must define window for detailWindow")
	}
	if arg.minSize.IsZero() {
		arg.minSize = fyne.NewSize(600, 500)
	}
	t := widget.NewLabel(arg.title)
	t.SizeName = theme.SizeNameSubHeadingText
	t.Truncation = fyne.TextTruncateEllipsis
	top := container.NewVBox(t, widget.NewSeparator())
	vs := container.NewVScroll(arg.content)
	vs.SetMinSize(arg.minSize)
	var image fyne.CanvasObject
	if arg.image != nil && !fyne.CurrentDevice().IsMobile() {
		x := iwidget.NewImageFromResource(arg.image, fyne.NewSquareSize(100))
		image = container.NewVBox(container.NewPadded(x))
	}
	c := container.NewBorder(
		top,
		nil,
		nil,
		image,
		vs,
	)
	c.Refresh()
	arg.window.SetContent(container.NewPadded(c))
}

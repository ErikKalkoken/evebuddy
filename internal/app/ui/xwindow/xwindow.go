// Package xwindow provides an extension to Fyne's window package.
package xwindow

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type Params struct {
	Content        fyne.CanvasObject
	EnableTooltips bool
	ImageAction    func()
	ImageLoader    func(setter func(r fyne.Resource))
	ImageSize      float32
	MinSize        fyne.Size
	Title          string
	Window         fyne.Window
}

// Set sets the content of a window to create a "detail window".
// Detail windows are used to show more information about objects in data lists.
func Set(arg Params) {
	if arg.Window == nil {
		panic("must define window for detailWindow")
	}
	if arg.MinSize.IsZero() {
		arg.MinSize = fyne.NewSize(600, 500)
	}
	if arg.ImageSize == 0 {
		arg.ImageSize = 64
	}

	var image2 fyne.CanvasObject
	if arg.ImageLoader != nil {
		image := xwidget.NewTappableImage(icons.BlankSvg, arg.ImageAction)
		image.SetFillMode(canvas.ImageFillContain)
		image.SetMinSize(fyne.NewSquareSize(arg.ImageSize))
		arg.ImageLoader(func(r fyne.Resource) {
			image.SetResource(r)
		})
		image2 = container.NewPadded(container.NewVBox((image)))
	}

	main := container.NewBorder(
		nil,
		nil,
		nil,
		image2,
		arg.Content,
	)

	vs := container.NewVScroll(main)
	vs.SetMinSize(arg.MinSize)

	t := widget.NewLabel(arg.Title)
	t.SizeName = theme.SizeNameSubHeadingText
	t.Truncation = fyne.TextTruncateEllipsis
	top := container.NewVBox(t, widget.NewSeparator())

	c := container.NewPadded(container.NewBorder(
		top,
		nil,
		nil,
		nil,
		vs,
	))
	c.Refresh()
	if arg.EnableTooltips {
		arg.Window.SetContent(fynetooltip.AddWindowToolTipLayer(c, arg.Window.Canvas()))
	} else {
		arg.Window.SetContent(c)
	}
}

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type detailWindowParams struct {
	content        fyne.CanvasObject
	enableTooltips bool
	imageAction    func()
	imageLoader    func() (fyne.Resource, error) // async loader
	imageSize      float32
	minSize        fyne.Size
	title          string
	window         fyne.Window
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
	if arg.imageSize == 0 {
		arg.imageSize = 64
	}

	var image2 fyne.CanvasObject
	if arg.imageLoader != nil {
		image := kxwidget.NewTappableImage(icons.BlankSvg, arg.imageAction)
		image.SetFillMode(canvas.ImageFillContain)
		image.SetMinSize(fyne.NewSquareSize(arg.imageSize))
		iwidget.RefreshTappableImageAsync(image, arg.imageLoader)
		image2 = container.NewPadded(container.NewVBox((image)))
	}

	main := container.NewBorder(
		nil,
		nil,
		nil,
		image2,
		arg.content,
	)

	vs := container.NewVScroll(main)
	vs.SetMinSize(arg.minSize)

	t := widget.NewLabel(arg.title)
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
	if arg.enableTooltips {
		arg.window.SetContent(fynetooltip.AddWindowToolTipLayer(c, arg.window.Canvas()))
	} else {
		arg.window.SetContent(c)
	}
}

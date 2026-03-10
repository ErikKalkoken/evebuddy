package ui

import (
	"fmt"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type MakeDetailWindowParams struct {
	Content        fyne.CanvasObject
	EnableTooltips bool
	ImageAction    func()
	ImageLoader    func(setter func(r fyne.Resource))
	ImageSize      float32
	MinSize        fyne.Size
	Title          string
	Window         fyne.Window
}

// MakeDetailWindow sets the content of a window to create a "detail window".
// Detail windows are used to show more information about objects in data lists.
func MakeDetailWindow(arg MakeDetailWindowParams) {
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

// FormatISKAmount returns a formatted ISK amount.
func FormatISKAmount(v float64) string {
	t := humanize.FormatFloat(FloatFormat, v) + " ISK"
	if math.Abs(v) > 999 {
		t += fmt.Sprintf(" (%s)", ihumanize.NumberF(v, 2))
	}
	return t
}

func MakeCharacterActionLabel(id int64, name string, action func(o *app.EveEntity)) fyne.CanvasObject {
	o := &app.EveEntity{
		ID:       id,
		Name:     name,
		Category: app.EveEntityCharacter,
	}
	return MakeEveEntityActionLabel(o, action)
}

// MakeEveEntityActionLabel returns a Hyperlink for existing entities or a placeholder label otherwise.
func MakeEveEntityActionLabel(o *app.EveEntity, action func(o *app.EveEntity)) fyne.CanvasObject {
	if o == nil {
		return widget.NewLabel("-")
	}
	return MakeLinkLabelWithWrap(o.Name, func() {
		action(o)
	})
}

func MakeLinkLabel(text string, action func()) *widget.Hyperlink {
	x := widget.NewHyperlink(text, nil)
	x.OnTapped = action
	return x
}

func MakeLinkLabelWithWrap(text string, action func()) *widget.Hyperlink {
	x := MakeLinkLabel(text, action)
	x.Wrapping = fyne.TextWrapWord
	return x
}

func MakeLocationLabel(o *app.EveLocationShort, show func(int64)) fyne.CanvasObject {
	if o == nil {
		return widget.NewLabel("?")
	}
	x := MakeLinkLabelWithWrap(o.DisplayName(), func() {
		show(o.ID)
	})
	x.Wrapping = fyne.TextWrapWord
	return x
}

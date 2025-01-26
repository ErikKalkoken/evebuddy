package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
)

type assetBadge struct {
	widget.BaseWidget

	quantity *canvas.Text
}

func NewAssetBadge() *assetBadge {
	q := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	q.TextSize = theme.CaptionTextSize()
	w := &assetBadge{quantity: q}
	w.ExtendBaseWidget(w)
	return w
}

func (w *assetBadge) SetQuantity(q int) {
	w.quantity.Text = humanize.Comma(int64(q))
	w.quantity.Refresh()
}

func (w *assetBadge) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	bgPadding := layout.NewCustomPaddedLayout(0, 0, p, p)
	customPadding := layout.NewCustomPaddedLayout(p/2, p/2, p/2, p/2)
	c := container.New(customPadding, container.NewStack(
		canvas.NewRectangle(theme.Color(theme.ColorNameBackground)),
		container.New(bgPadding, w.quantity),
	))
	return widget.NewSimpleRenderer(c)
}

package core

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// PageHeader is a widget for rendering the header on a page.
// Headers contain a title and can also have a leading icon and a drop down menu.
// It must be wrapped in a container that keeps it at its MinSize, e.g. an HBox.
// Otherwise hover, tap and tooltip trigger beyond the drawn area.
type PageHeader struct {
	ttwidget.ToolTipWidget

	background       *canvas.Rectangle
	hovered          bool
	label            *widget.Label
	leadingContainer *fyne.Container
	leadingImage     *canvas.Image
	menu             *fyne.Menu
	trailingIcon     *widget.Icon
}

var _ fyne.Tappable = (*PageHeader)(nil)
var _ desktop.Hoverable = (*PageHeader)(nil)

func NewPageHeader(title string, leading fyne.Resource) *PageHeader {
	w := &PageHeader{
		background:   canvas.NewRectangle(color.Transparent),
		label:        widget.NewLabel(title),
		leadingImage: xwidget.NewImageFromResource(leading, fyne.NewSquareSize(ui.IconUnitSize)),
		menu:         fyne.NewMenu(""),
		trailingIcon: widget.NewIcon(theme.MenuDropDownIcon()),
	}
	w.ExtendBaseWidget(w)
	w.updateStyling()
	p := theme.Padding()
	w.leadingContainer = container.New(layout.NewCustomPaddedLayout(0, 0, p, p), w.leadingImage)
	if leading == nil {
		w.leadingContainer.Hide()
	}
	w.trailingIcon.Hide()
	w.label.SizeName = theme.SizeNameSubHeadingText
	return w
}

func (w *PageHeader) CreateRenderer() fyne.WidgetRenderer {
	w.updateStyling()
	spacer := xwidget.NewSpacer(w.trailingIcon.MinSize())
	p := theme.Padding()
	trailing := container.New(layout.NewCustomPaddedLayout(0, 0, -0.5*p, 0), container.NewStack(
		spacer,
		container.NewCenter(w.trailingIcon),
	))
	c := container.NewStack(
		w.background,
		container.New(layout.NewCustomPaddedHBoxLayout(0), w.leadingContainer, w.label, trailing),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *PageHeader) Refresh() {
	w.updateStyling()
	w.background.Refresh()
	w.BaseWidget.Refresh()
}

func (w *PageHeader) updateStyling() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.background.FillColor = th.Color(theme.ColorNameHover, v)
	w.background.CornerRadius = th.Size(theme.SizeNameButtonRadius)
	if w.hovered && w.hasMenu() {
		w.background.Show()
	} else {
		w.background.Hide()
	}
}

func (w *PageHeader) SetIcon(r fyne.Resource) {
	w.leadingImage.Resource = r
	w.leadingImage.Refresh()
	if r != nil {
		w.leadingContainer.Show()
	} else {
		w.leadingContainer.Hide()
	}
}

func (w *PageHeader) SetMenu(it []*fyne.MenuItem) {
	w.menu.Items = it
	w.menu.Refresh()
	if w.hasMenu() {
		w.trailingIcon.Show()
	} else {
		w.trailingIcon.Hide()
	}
	w.updateStyling()
	w.background.Refresh()
}

func (w *PageHeader) SetTitle(s string) {
	w.label.SetText(s)
}

func (w *PageHeader) Tapped(_ *fyne.PointEvent) {
	if !w.hasMenu() {
		return
	}
	xwidget.ShowPopUpMenuBelowLeading(w, w.menu)
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (w *PageHeader) MouseIn(me *desktop.MouseEvent) {
	w.ToolTipWidget.MouseIn(me)
	w.setHovered(true)
}

func (w *PageHeader) MouseMoved(me *desktop.MouseEvent) {
	w.ToolTipWidget.MouseMoved(me)
}

// MouseOut is a hook that is called if the mouse pointer leaves the element.
func (w *PageHeader) MouseOut() {
	w.ToolTipWidget.MouseOut()
	w.setHovered(false)
}

// setHovered updates the background only when the hover state changes,
// since mouse events fire often.
func (w *PageHeader) setHovered(v bool) {
	if w.hovered == v {
		return
	}
	w.hovered = v
	w.updateStyling()
	w.background.Refresh()
}

func (w *PageHeader) hasMenu() bool {
	return len(w.menu.Items) > 0
}

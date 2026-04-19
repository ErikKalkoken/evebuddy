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

	"github.com/ErikKalkoken/evebuddy/internal/icons"
)

// StatusBarItem is a widget with a label and an optional icon, which can be tapped.
type StatusBarItem struct {
	ttwidget.ToolTipWidget

	// The function that is called when the label is tapped.
	OnTapped func()

	bg       *canvas.Rectangle
	label    *widget.Label
	leading  *widget.Icon
	trailing fyne.CanvasObject
}

var _ fyne.Tappable = (*StatusBarItem)(nil)
var _ desktop.Hoverable = (*StatusBarItem)(nil)

func NewStatusBarItem(leading fyne.Resource, text string, tapped func()) *StatusBarItem {
	return NewStatusBarItemWithTrailing(leading, nil, text, tapped)
}

func NewStatusBarItemWithTrailing(leading fyne.Resource, trailing fyne.CanvasObject, text string, tapped func()) *StatusBarItem {
	icon := widget.NewIcon(icons.BlankSvg)
	if leading != nil {
		icon.SetResource(leading)
	} else {
		icon.Hide()
	}
	if trailing == nil {
		trailing = canvas.NewRectangle(color.Transparent)
		trailing.Hide()
	}
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameHover))
	bg.Hide()
	w := &StatusBarItem{
		bg:       bg,
		label:    widget.NewLabel(text),
		leading:  icon,
		OnTapped: tapped,
		trailing: trailing,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *StatusBarItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewStack(
		w.bg,
		container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, p),
			container.New(layout.NewCustomPaddedHBoxLayout(0),
				container.NewVBox(layout.NewSpacer(), w.leading, layout.NewSpacer()),
				container.NewVBox(layout.NewSpacer(), w.label, layout.NewSpacer()),
				container.NewVBox(layout.NewSpacer(), w.trailing, layout.NewSpacer()),
			)),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *StatusBarItem) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(theme.ColorNameHover, v)
	w.bg.Refresh()
	w.leading.Refresh()
	w.label.Refresh()
	w.BaseWidget.Refresh()
}

// SetLeading updates the leading icon.
func (w *StatusBarItem) SetLeading(icon fyne.Resource) {
	w.leading.SetResource(icon)
}

// SetText updates the label's text.
func (w *StatusBarItem) SetText(text string) {
	w.SetTextAndImportance(text, widget.MediumImportance)
}

// SetTextAndImportance updates the label's text and importance.
func (w *StatusBarItem) SetTextAndImportance(text string, importance widget.Importance) {
	w.label.Text = text
	w.label.Importance = importance
	w.label.Refresh()
}

func (w *StatusBarItem) Tapped(_ *fyne.PointEvent) {
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *StatusBarItem) TappedSecondary(_ *fyne.PointEvent) {
}

func (w *StatusBarItem) MouseIn(e *desktop.MouseEvent) {
	w.ToolTipWidget.MouseIn(e)
	if w.OnTapped != nil {
		w.bg.Show()
	}
}

func (w *StatusBarItem) MouseMoved(e *desktop.MouseEvent) {
	w.ToolTipWidget.MouseMoved(e)
}

func (w *StatusBarItem) MouseOut() {
	w.ToolTipWidget.MouseOut()
	w.bg.Hide()
}

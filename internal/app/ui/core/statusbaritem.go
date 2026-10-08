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

	hoover      *canvas.Rectangle
	label       *widget.Label
	leadingIcon *widget.Icon
	trailing    fyne.CanvasObject
}

var _ fyne.Tappable = (*StatusBarItem)(nil)
var _ desktop.Hoverable = (*StatusBarItem)(nil)

func NewStatusBarItem(leading fyne.Resource, text string, tapped func()) *StatusBarItem {
	return NewStatusBarItemWithTrailing(leading, nil, text, tapped)
}

func NewStatusBarItemWithTrailing(leading fyne.Resource, trailing fyne.CanvasObject, text string, tapped func()) *StatusBarItem {
	if trailing == nil {
		trailing = canvas.NewRectangle(color.Transparent)
		trailing.Hide()
	}
	w := &StatusBarItem{
		hoover:      canvas.NewRectangle(color.Transparent),
		label:       widget.NewLabel(text),
		leadingIcon: widget.NewIcon(icons.BlankSvg),
		OnTapped:    tapped,
		trailing:    trailing,
	}
	w.ExtendBaseWidget(w)
	w.hoover.Hide()
	if leading != nil {
		w.leadingIcon.SetResource(leading)
	} else {
		w.leadingIcon.Hide()
	}
	return w
}

func (w *StatusBarItem) CreateRenderer() fyne.WidgetRenderer {
	w.updateStyle()
	p := w.Theme().Size(theme.SizeNamePadding)
	c := container.NewStack(
		w.hoover,
		container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, p),
			container.New(layout.NewCustomPaddedHBoxLayout(0),
				container.NewVBox(layout.NewSpacer(), w.leadingIcon, layout.NewSpacer()),
				container.NewVBox(layout.NewSpacer(), w.label, layout.NewSpacer()),
				container.NewVBox(layout.NewSpacer(), w.trailing, layout.NewSpacer()),
			)),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *StatusBarItem) Refresh() {
	w.updateStyle()
	w.hoover.Refresh()
	w.leadingIcon.Refresh()
	w.label.Refresh()
	w.BaseWidget.Refresh()
}

func (w *StatusBarItem) updateStyle() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.hoover.FillColor = th.Color(theme.ColorNameHover, v)
	w.hoover.CornerRadius = th.Size(theme.SizeNameButtonRadius)
}

// SetLeading updates the leading icon. A nil icon hides it.
func (w *StatusBarItem) SetLeading(icon fyne.Resource) {
	if icon == nil {
		w.leadingIcon.Hide()
		return
	}
	w.leadingIcon.SetResource(icon)
	w.leadingIcon.Show()
	w.Refresh()
}

// SetText updates the label's text. And resets its importance to medium.
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
	w.ToolTipWidget.MouseOut() // cancel pending tooltip
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *StatusBarItem) MouseIn(e *desktop.MouseEvent) {
	w.ToolTipWidget.MouseIn(e)
	if w.OnTapped != nil {
		w.hoover.Show()
	}
}

func (w *StatusBarItem) MouseMoved(e *desktop.MouseEvent) {
	w.ToolTipWidget.MouseMoved(e)
}

func (w *StatusBarItem) MouseOut() {
	w.ToolTipWidget.MouseOut()
	w.hoover.Hide()
}

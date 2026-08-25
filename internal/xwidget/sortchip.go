package xwidget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type SortChip struct {
	widget.DisableableWidget

	Text string
	On   bool

	// OnChanged is called when the state changed
	OnChanged func(on bool)

	bg                   *canvas.Rectangle
	focused              bool
	hovered              bool
	icon                 *widget.Icon
	iconPadded           *fyne.Container
	label                *widget.Label
	resourceIcon         fyne.Resource
	resourceIconDisabled fyne.Resource
}

var _ desktop.Hoverable = (*SortChip)(nil)
var _ fyne.Disableable = (*SortChip)(nil)
var _ fyne.Focusable = (*SortChip)(nil)
var _ fyne.Tappable = (*SortChip)(nil)
var _ fyne.Widget = (*SortChip)(nil)

// NewSortChip returns a new [SortChip] object.
func NewSortChip(text string, changed func(on bool)) *SortChip {
	bg := canvas.NewRectangle(color.Transparent)
	bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder)
	bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	w := &SortChip{
		bg:                   bg,
		icon:                 widget.NewIcon(theme.ConfirmIcon()),
		label:                widget.NewLabel(text),
		OnChanged:            changed,
		resourceIcon:         theme.ConfirmIcon(),
		resourceIconDisabled: theme.NewDisabledResource(theme.ConfirmIcon()),
		Text:                 text,
	}
	w.ExtendBaseWidget(w)
	p := theme.Padding()
	w.iconPadded = container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, -p), w.icon)
	w.iconPadded.Hide()
	return w
}

// SetState sets the state.
func (w *SortChip) SetState(v bool) {
	if w.On == v {
		return
	}
	w.On = v
	if w.OnChanged != nil {
		w.OnChanged(v)
	}
	w.Refresh()
}

func (w *SortChip) Refresh() {
	w.updateState()
	w.bg.Refresh()
	w.label.Refresh()
	w.icon.Refresh()
	w.BaseWidget.Refresh()
}

func (w *SortChip) updateState() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	w.label.Text = w.Text

	if w.Disabled() {
		w.label.Importance = widget.LowImportance
		w.icon.SetResource(theme.NewDisabledResource(theme.ConfirmIcon()))
		w.bg.StrokeColor = th.Color(theme.ColorNameDisabled, v)
	} else {
		w.label.Importance = widget.MediumImportance
		w.icon.SetResource(theme.ConfirmIcon())
		w.bg.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	}
	if w.On {
		w.iconPadded.Show()
		if w.Disabled() {
			w.bg.FillColor = th.Color(theme.ColorNameDisabledButton, v)
			w.bg.StrokeColor = th.Color(theme.ColorNameDisabledButton, v)
		} else {
			w.bg.FillColor = th.Color(theme.ColorNameSelection, v)
			w.bg.StrokeColor = th.Color(theme.ColorNameSelection, v)
		}
	} else {
		w.iconPadded.Hide()
		w.bg.FillColor = color.Transparent
	}

	if w.focused {
		w.bg.StrokeColor = th.Color(theme.ColorNameFocus, v)
	}
}

func (w *SortChip) Tapped(pe *fyne.PointEvent) {
	if w.Disabled() {
		return
	}
	w.SetState(!w.On)
}

func (w *SortChip) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *SortChip) MouseIn(me *desktop.MouseEvent) {
	if w.Disabled() {
		return
	}
	w.hovered = true
}

func (w *SortChip) MouseMoved(me *desktop.MouseEvent) {
	// needed to satisfy the interface only
}

func (w *SortChip) MouseOut() {
	w.hovered = false
}

// FocusGained is called when the Check has been given focus.
func (w *SortChip) FocusGained() {
	if w.Disabled() {
		return
	}
	w.focused = true
	w.Refresh()
}

// FocusLost is called when the Check has had focus removed.
func (w *SortChip) FocusLost() {
	w.focused = false
	w.Refresh()
}

// TypedRune receives text input events when the Check is focused.
func (w *SortChip) TypedRune(r rune) {
	if w.Disabled() {
		return
	}
	if r == ' ' {
		w.SetState(!w.On)
	}
}

// TypedKey receives key input events when the Check is focused.
func (w *SortChip) TypedKey(key *fyne.KeyEvent) {}

func (w *SortChip) CreateRenderer() fyne.WidgetRenderer {
	w.updateState()
	c := container.NewStack(
		w.bg,
		container.NewCenter(container.New(
			layout.NewCustomPaddedHBoxLayout(0),
			w.iconPadded,
			w.label,
		),
		))
	return widget.NewSimpleRenderer(c)
}

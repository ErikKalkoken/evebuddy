package widget

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

// FilterChip represents a simple filter chip widget which only has two states: on or off.
// Filter chips use tags or descriptive words to filter content.
type FilterChip struct {
	widget.DisableableWidget

	Text string
	On   bool

	// OnChanged is called when the state changed
	OnChanged func(on bool)

	bg         *canvas.Rectangle
	focused    bool
	hovered    bool
	icon       *widget.Icon
	iconPadded *fyne.Container
	label      *widget.Label
	minSize    fyne.Size // cached for hover/top pos calcs
}

var _ desktop.Hoverable = (*FilterChip)(nil)
var _ fyne.Disableable = (*FilterChip)(nil)
var _ fyne.Focusable = (*FilterChip)(nil)
var _ fyne.Tappable = (*FilterChip)(nil)
var _ fyne.Widget = (*FilterChip)(nil)

// NewFilterChip returns a new [FilterChip] object.
func NewFilterChip(text string, changed func(on bool)) *FilterChip {
	w := &FilterChip{
		label:     widget.NewLabel(text),
		OnChanged: changed,
		Text:      text,
	}
	w.ExtendBaseWidget(w)
	p := theme.Padding()
	w.icon = widget.NewIcon(theme.ConfirmIcon())
	w.iconPadded = container.New(layout.NewCustomPaddedLayout(0, 0, p, 0), w.icon)
	w.iconPadded.Hide()
	w.bg = canvas.NewRectangle(color.Transparent)
	w.bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder) * 2
	w.bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	return w
}

// SetState sets the state.
func (w *FilterChip) SetState(v bool) {
	if w.On == v {
		return
	}
	w.On = v
	if w.OnChanged != nil {
		w.OnChanged(v)
	}
	w.Refresh()
}

func (w *FilterChip) Refresh() {
	w.updateState()
	w.bg.Refresh()
	w.label.Refresh()
	w.icon.Refresh()
	w.BaseWidget.Refresh()
}

func (w *FilterChip) updateState() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	w.label.Text = w.Text

	if w.Disabled() {
		w.label.Importance = widget.LowImportance
		w.icon.Resource = theme.NewDisabledResource(theme.ConfirmIcon())
		w.bg.StrokeColor = th.Color(theme.ColorNameDisabled, v)
	} else {
		w.label.Importance = widget.MediumImportance
		w.icon.Resource = theme.ConfirmIcon()
		w.bg.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	}
	if w.On {
		w.iconPadded.Show()
		if w.Disabled() {
			w.bg.FillColor = th.Color(theme.ColorNameDisabledButton, v)
			w.bg.StrokeColor = th.Color(theme.ColorNameDisabledButton, v)
		} else {
			w.bg.FillColor = th.Color(theme.ColorNameButton, v)
			w.bg.StrokeColor = th.Color(theme.ColorNameButton, v)
		}
	} else {
		w.iconPadded.Hide()
		w.bg.FillColor = color.Transparent
	}

	if w.focused {
		w.bg.StrokeColor = th.Color(theme.ColorNameFocus, v)
	}
}

func (w *FilterChip) MinSize() fyne.Size {
	w.ExtendBaseWidget(w)
	w.minSize = w.BaseWidget.MinSize()
	return w.minSize
}

func (w *FilterChip) Tapped(pe *fyne.PointEvent) {
	if w.Disabled() {
		return
	}
	if !w.minSize.IsZero() &&
		(pe.Position.X > w.minSize.Width || pe.Position.Y > w.minSize.Height) {
		// tapped outside
		return
	}
	// if !w.focused {
	// 	if !fyne.CurrentDevice().IsMobile() {
	// 		if c := fyne.CurrentApp().Driver().CanvasForObject(w); c != nil {
	// 			c.Focus(w)
	// 		}
	// 	}
	// }
	w.SetState(!w.On)
}

func (w *FilterChip) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *FilterChip) MouseIn(me *desktop.MouseEvent) {
	w.MouseMoved(me)
}

func (w *FilterChip) MouseMoved(me *desktop.MouseEvent) {
	if w.Disabled() {
		return
	}
	oldHovered := w.hovered
	w.hovered = w.minSize.IsZero() ||
		(me.Position.X <= w.minSize.Width && me.Position.Y <= w.minSize.Height)

	if oldHovered != w.hovered {
		w.Refresh()
	}
}

func (w *FilterChip) MouseOut() {
	if w.hovered {
		w.hovered = false
		w.Refresh()
	}
}

// FocusGained is called when the Check has been given focus.
func (w *FilterChip) FocusGained() {
	if w.Disabled() {
		return
	}
	w.focused = true
	w.Refresh()
}

// FocusLost is called when the Check has had focus removed.
func (w *FilterChip) FocusLost() {
	w.focused = false
	w.Refresh()
}

// TypedRune receives text input events when the Check is focused.
func (w *FilterChip) TypedRune(r rune) {
	if w.Disabled() {
		return
	}
	if r == ' ' {
		w.SetState(!w.On)
	}
}

// TypedKey receives key input events when the Check is focused.
func (w *FilterChip) TypedKey(key *fyne.KeyEvent) {}

func (w *FilterChip) CreateRenderer() fyne.WidgetRenderer {
	w.updateState()
	p := theme.Padding()
	c := container.NewHBox(container.NewStack(
		w.bg,
		container.New(
			layout.NewCustomPaddedLayout(0, 0, p, p),
			container.New(
				layout.NewCustomPaddedHBoxLayout(0),
				layout.NewSpacer(),
				container.NewVBox(layout.NewSpacer(), w.iconPadded, layout.NewSpacer()),
				container.NewVBox(layout.NewSpacer(), w.label, layout.NewSpacer()),
				layout.NewSpacer(),
			),
		)))
	return widget.NewSimpleRenderer(c)
}

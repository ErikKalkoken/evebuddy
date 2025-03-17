package widget

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TODO: Add focus feature

// Filter chips use tags or descriptive words to filter content.
type filterChip struct {
	widget.DisableableWidget

	Text       string
	IsSelected bool

	// OnChanged is called when the state changed
	OnChanged func(selected bool)

	hovered    bool
	minSize    fyne.Size // cached for hover/top pos calcs
	iconPadded *fyne.Container
	icon       *widget.Icon
	label      *widget.Label
	bg         *canvas.Rectangle
	mu         sync.RWMutex
}

var _ fyne.Widget = (*filterChip)(nil)
var _ fyne.Tappable = (*filterChip)(nil)
var _ desktop.Hoverable = (*filterChip)(nil)
var _ fyne.Disableable = (*filterChip)(nil)

// newFilterChip returns a new filterChip object.
func newFilterChip(text string, changed func(selected bool)) *filterChip {
	w := &filterChip{
		label:     widget.NewLabel(text),
		OnChanged: changed,
		Text:      text,
	}
	p := theme.Padding()
	w.icon = widget.NewIcon(theme.ConfirmIcon())
	w.iconPadded = container.New(layout.NewCustomPaddedLayout(0, 0, p, 0), w.icon)
	w.iconPadded.Hide()
	w.bg = canvas.NewRectangle(color.Transparent)
	w.bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder) * 2
	w.bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	w.ExtendBaseWidget(w)
	return w
}

// Selected reports whether the widget is selected.
func (w *filterChip) Selected() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.IsSelected
}

// SetSelected sets the selected state.
func (w *filterChip) SetSelected(v bool) {
	w.mu.Lock()
	old := w.IsSelected
	w.IsSelected = v
	w.mu.Unlock()
	if w.OnChanged != nil && old != v {
		w.OnChanged(v)
	}
	w.Refresh()
}

func (w *filterChip) Refresh() {
	w.updateState()
	w.bg.Refresh()
	w.label.Refresh()
	w.icon.Refresh()
	w.BaseWidget.Refresh()
}

func (w *filterChip) updateState() {
	w.mu.RLock()
	defer w.mu.RUnlock()
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
	if w.IsSelected {
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
}

func (w *filterChip) MinSize() fyne.Size {
	w.ExtendBaseWidget(w)
	w.minSize = w.BaseWidget.MinSize()
	return w.minSize
}

func (w *filterChip) Tapped(pe *fyne.PointEvent) {
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
	w.SetSelected(!w.IsSelected)
}

func (w *filterChip) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *filterChip) MouseIn(me *desktop.MouseEvent) {
	w.MouseMoved(me)
}

func (w *filterChip) MouseMoved(me *desktop.MouseEvent) {
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

func (w *filterChip) MouseOut() {
	if w.hovered {
		w.hovered = false
		w.Refresh()
	}
}

func (w *filterChip) CreateRenderer() fyne.WidgetRenderer {
	w.updateState()
	p := theme.Padding()
	c := container.NewHBox(container.NewStack(
		w.bg,
		container.New(
			layout.NewCustomPaddedLayout(0, 0, p, p),
			container.New(
				layout.NewCustomPaddedHBoxLayout(0),
				layout.NewSpacer(),
				w.iconPadded,
				w.label,
				layout.NewSpacer(),
			),
		)))
	return widget.NewSimpleRenderer(c)
}

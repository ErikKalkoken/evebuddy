package xwidget

import (
	"image/color"
	"math"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SegmentedButton is a Material Design 3 style segmented button
// for choosing exactly one of a few (2-5) options.
//
// Labels are never truncated, because Material Design requires them to be fully visible.
// Keep them short, or use a select widget for long options.
type SegmentedButton struct {
	widget.DisableableWidget

	// OnChanged is called when the selection changed,
	// either by the user or through SetSelected.
	// It passes the index of the selected segment.
	OnChanged func(index int)

	// Selected is the index of the selected segment.
	// Setting it directly does not call OnChanged. Call Refresh afterwards.
	// When it is out of range, no segment is shown as selected.
	Selected int

	focused bool
	hovered bool
	labels  []string
}

var _ desktop.Hoverable = (*SegmentedButton)(nil)
var _ fyne.Disableable = (*SegmentedButton)(nil)
var _ fyne.Focusable = (*SegmentedButton)(nil)
var _ fyne.Tappable = (*SegmentedButton)(nil)
var _ fyne.Widget = (*SegmentedButton)(nil)

// NewSegmentedButton creates and returns a new [SegmentedButton].
// The first segment is selected initially.
// It panics when labels is empty.
func NewSegmentedButton(labels []string, changed func(index int)) *SegmentedButton {
	if len(labels) == 0 {
		panic("SegmentedButton: labels must not be empty")
	}
	w := &SegmentedButton{
		OnChanged: changed,
		labels:    slices.Clone(labels),
	}
	w.ExtendBaseWidget(w)
	return w
}

// SetSelected selects the segment with the given index.
// Out-of-range indexes are ignored.
// Calls OnChanged when the selection changed.
func (w *SegmentedButton) SetSelected(index int) {
	if index < 0 || index >= len(w.labels) || index == w.Selected {
		return
	}
	w.Selected = index
	w.Refresh()
	if w.OnChanged != nil {
		w.OnChanged(index)
	}
}

func (w *SegmentedButton) Tapped(pe *fyne.PointEvent) {
	if w.Disabled() {
		return
	}
	w.SetSelected(w.segmentAt(pe.Position.X))
}

// segmentAt returns the index of the segment at horizontal position x.
func (w *SegmentedButton) segmentAt(x float32) int {
	width := w.Size().Width
	if width <= 0 {
		return w.Selected
	}
	n := len(w.labels)
	for i := range n - 1 {
		if x < segmentBoundary(i+1, n, width) {
			return i
		}
	}
	return n - 1
}

// segmentBoundary returns the x position where segment i of n starts.
// Positions are rounded to whole pixels to keep edges and dividers crisp.
func segmentBoundary(i, n int, width float32) float32 {
	return float32(math.Round(float64(width) * float64(i) / float64(n)))
}

func (w *SegmentedButton) Cursor() desktop.Cursor {
	if !w.Disabled() && w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *SegmentedButton) MouseIn(me *desktop.MouseEvent) {
	if w.Disabled() {
		return
	}
	w.hovered = true
}

func (w *SegmentedButton) MouseMoved(me *desktop.MouseEvent) {}

func (w *SegmentedButton) MouseOut() {
	w.hovered = false
}

func (w *SegmentedButton) FocusGained() {
	if w.Disabled() {
		return
	}
	w.focused = true
	w.Refresh()
}

func (w *SegmentedButton) FocusLost() {
	w.focused = false
	w.Refresh()
}

func (w *SegmentedButton) TypedRune(r rune) {}

// TypedKey moves the selection with the left and right arrow keys.
func (w *SegmentedButton) TypedKey(key *fyne.KeyEvent) {
	if w.Disabled() {
		return
	}
	switch key.Name {
	case fyne.KeyLeft:
		w.SetSelected(w.Selected - 1)
	case fyne.KeyRight:
		w.SetSelected(w.Selected + 1)
	}
}

func (w *SegmentedButton) CreateRenderer() fyne.WidgetRenderer {
	r := &segmentedButtonRenderer{
		w:       w,
		outline: canvas.NewRectangle(color.Transparent),
	}
	for range w.labels {
		r.segments = append(r.segments, segment{
			bg:    canvas.NewRectangle(color.Transparent),
			icon:  widget.NewIcon(nil),
			label: canvas.NewText("", color.Transparent),
		})
	}
	for range len(w.labels) - 1 {
		r.dividers = append(r.dividers, canvas.NewRectangle(color.Transparent))
	}
	r.updateState()
	return r
}

type segment struct {
	bg    *canvas.Rectangle
	icon  *widget.Icon
	label *canvas.Text
}

type segmentedButtonRenderer struct {
	dividers []*canvas.Rectangle
	outline  *canvas.Rectangle
	segments []segment
	w        *SegmentedButton
}

func (r *segmentedButtonRenderer) Objects() []fyne.CanvasObject {
	var objs []fyne.CanvasObject
	for _, s := range r.segments {
		objs = append(objs, s.bg)
	}
	for _, s := range r.segments {
		objs = append(objs, s.icon, s.label)
	}
	for _, d := range r.dividers {
		objs = append(objs, d)
	}
	objs = append(objs, r.outline)
	return objs
}

func (r *segmentedButtonRenderer) Destroy() {}

func (r *segmentedButtonRenderer) Layout(size fyne.Size) {
	th := r.w.Theme()
	innerPadding := th.Size(theme.SizeNameInnerPadding)
	gap := innerPadding / 2
	iconSize := th.Size(theme.SizeNameInlineIcon)
	n := len(r.segments)

	r.outline.Resize(size)
	r.outline.Move(fyne.NewPos(0, 0))

	for i, s := range r.segments {
		x := segmentBoundary(i, n, size.Width)
		segWidth := segmentBoundary(i+1, n, size.Width) - x
		s.bg.Resize(fyne.NewSize(segWidth, size.Height))
		s.bg.Move(fyne.NewPos(x, 0))

		var iconWidth float32
		if !s.icon.Hidden {
			iconWidth = iconSize + gap
		}
		textMin := s.label.MinSize()

		contentWidth := iconWidth + textMin.Width
		cx := x + (segWidth-contentWidth)/2
		if !s.icon.Hidden {
			s.icon.Resize(fyne.NewSquareSize(iconSize))
			s.icon.Move(fyne.NewPos(cx, (size.Height-iconSize)/2))
			cx += iconWidth
		}
		s.label.Resize(textMin)
		s.label.Move(fyne.NewPos(cx, (size.Height-textMin.Height)/2))
	}

	strokeWidth := th.Size(theme.SizeNameInputBorder)
	for i, d := range r.dividers {
		d.Resize(fyne.NewSize(strokeWidth, size.Height))
		d.Move(fyne.NewPos(segmentBoundary(i+1, n, size.Width), 0))
	}
}

func (r *segmentedButtonRenderer) MinSize() fyne.Size {
	th := r.w.Theme()
	innerPadding := th.Size(theme.SizeNameInnerPadding)
	gap := innerPadding / 2
	iconSize := th.Size(theme.SizeNameInlineIcon)
	textSize := th.Size(theme.SizeNameText)

	// Reserve space for the checkmark in every segment,
	// so the size does not change with the selection.
	var maxText fyne.Size
	for _, l := range r.w.labels {
		s := fyne.MeasureText(l, textSize, fyne.TextStyle{})
		maxText = maxText.Max(s)
	}
	// Round up to whole pixels, so snapping the segment boundaries
	// can not make a segment narrower than its content.
	segWidth := float32(math.Ceil(float64(iconSize + gap + maxText.Width + 2*innerPadding)))
	height := max(maxText.Height, iconSize) + 2*innerPadding
	return fyne.NewSize(segWidth*float32(len(r.segments)), height)
}

func (r *segmentedButtonRenderer) updateState() {
	w := r.w
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	radius := th.Size(theme.SizeNameInputRadius)
	strokeWidth := th.Size(theme.SizeNameInputBorder)

	var strokeColor, textColor color.Color
	if w.Disabled() {
		strokeColor = th.Color(theme.ColorNameDisabled, v)
		textColor = th.Color(theme.ColorNameDisabled, v)
	} else {
		strokeColor = th.Color(theme.ColorNameInputBorder, v)
		textColor = th.Color(theme.ColorNameForeground, v)
	}

	n := len(r.segments)
	for i, s := range r.segments {
		s.label.Text = w.labels[i]
		s.label.TextSize = th.Size(theme.SizeNameText)
		s.label.Color = textColor

		var tl, bl, tr, br float32
		if i == 0 {
			tl, bl = radius, radius
		}
		if i == n-1 {
			tr, br = radius, radius
		}
		s.bg.TopLeftCornerRadius = tl
		s.bg.BottomLeftCornerRadius = bl
		s.bg.TopRightCornerRadius = tr
		s.bg.BottomRightCornerRadius = br

		if i == w.Selected {
			if w.Disabled() {
				s.bg.FillColor = th.Color(theme.ColorNameDisabledButton, v)
				s.icon.SetResource(theme.NewDisabledResource(theme.ConfirmIcon()))
			} else {
				s.bg.FillColor = th.Color(theme.ColorNameSelection, v)
				s.icon.SetResource(theme.ConfirmIcon())
			}
			s.icon.Show()
		} else {
			s.bg.FillColor = color.Transparent
			s.icon.Hide()
		}
	}

	for _, d := range r.dividers {
		d.FillColor = strokeColor
	}

	if w.focused {
		strokeColor = th.Color(theme.ColorNameFocus, v)
	}
	r.outline.StrokeColor = strokeColor
	r.outline.StrokeWidth = strokeWidth
	r.outline.CornerRadius = radius
}

func (r *segmentedButtonRenderer) Refresh() {
	r.updateState()
	r.Layout(r.w.Size())
	for _, s := range r.segments {
		s.bg.Refresh()
		s.icon.Refresh()
		s.label.Refresh()
	}
	for _, d := range r.dividers {
		d.Refresh()
	}
	r.outline.Refresh()
}

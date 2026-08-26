package xwidget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// A Chip is a compact, interactive widget that represents a single entity, attribute,
// action, or filter.
//
// A Chip has a label and can have a leading and/or trailing icon.
// It has an on and off state and can be disabled.
type Chip struct {
	widget.DisableableWidget

	OnTapped     func()
	On           bool
	LeadingIcon  fyne.Resource
	TrailingIcon fyne.Resource
	Text         string

	focused bool
	hovered bool
}

var _ desktop.Hoverable = (*Chip)(nil)
var _ fyne.Disableable = (*Chip)(nil)
var _ fyne.Focusable = (*Chip)(nil)
var _ fyne.Tappable = (*Chip)(nil)
var _ fyne.Widget = (*Chip)(nil)

// NewChip returns a new [Chip] object.
func NewChip(text string, tapped func()) *Chip {
	w := &Chip{
		OnTapped: tapped,
		Text:     text,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *Chip) Tapped(pe *fyne.PointEvent) {
	if w.Disabled() {
		return
	}
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *Chip) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *Chip) MouseIn(me *desktop.MouseEvent) {
	if w.Disabled() {
		return
	}
	w.hovered = true
}

func (w *Chip) MouseMoved(me *desktop.MouseEvent) {}

func (w *Chip) MouseOut() {
	w.hovered = false
}

func (w *Chip) FocusGained() {
	if w.Disabled() {
		return
	}
	w.focused = true
	w.Refresh()
}

func (w *Chip) FocusLost() {
	w.focused = false
	w.Refresh()
}

func (w *Chip) TypedRune(r rune) {
	if w.Disabled() {
		return
	}
	if r == ' ' && w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *Chip) TypedKey(key *fyne.KeyEvent) {}

func (w *Chip) CreateRenderer() fyne.WidgetRenderer {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	bg := canvas.NewRectangle(color.Transparent)
	bg.StrokeWidth = th.Size(theme.SizeNameInputBorder)
	bg.CornerRadius = th.Size(theme.SizeNameInputRadius)

	leadingIcon := widget.NewIcon(w.LeadingIcon)
	trailingIcon := widget.NewIcon(w.TrailingIcon)

	label := canvas.NewText(w.Text, th.Color(theme.ColorNameForeground, v))
	label.TextSize = th.Size(theme.SizeNameText)

	r := &chipRenderer{
		chip:         w,
		bg:           bg,
		leadingIcon:  leadingIcon,
		trailingIcon: trailingIcon,
		label:        label,
	}

	if w.LeadingIcon == nil {
		leadingIcon.Hide()
	}
	if w.TrailingIcon == nil {
		trailingIcon.Hide()
	}

	return r
}

type chipRenderer struct {
	bg           *canvas.Rectangle
	chip         *Chip
	label        *canvas.Text
	leadingIcon  *widget.Icon
	trailingIcon *widget.Icon
}

func (r *chipRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.leadingIcon, r.label, r.trailingIcon}
}

func (r *chipRenderer) Destroy() {}

func (r *chipRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))

	textMin := r.label.MinSize()
	th := r.chip.Theme()
	gapSize := th.Size(theme.SizeNameInnerPadding) / 2

	var leadWidth, leadHeight float32
	var leadGap float32
	if !r.leadingIcon.Hidden {
		leadMin := r.leadingIcon.MinSize()
		leadWidth = leadMin.Width
		leadHeight = leadMin.Height
		if textMin.Width > 0 || !r.trailingIcon.Hidden {
			leadGap = gapSize
		}
	} else {
		r.leadingIcon.Resize(fyne.NewSize(0, 0))
	}

	var trailWidth, trailHeight float32
	var trailGap float32
	if !r.trailingIcon.Hidden {
		trailMin := r.trailingIcon.MinSize()
		trailWidth = trailMin.Width
		trailHeight = trailMin.Height
		if textMin.Width > 0 || !r.leadingIcon.Hidden {
			trailGap = gapSize
		}
	} else {
		r.trailingIcon.Resize(fyne.NewSize(0, 0))
	}

	contentWidth := leadWidth + leadGap + textMin.Width + trailGap + trailWidth
	contentHeight := fyne.Max(fyne.Max(leadHeight, textMin.Height), trailHeight)

	startX := (size.Width - contentWidth) / 2
	startY := (size.Height - contentHeight) / 2
	currentX := startX

	// Position Leading Icon
	if !r.leadingIcon.Hidden {
		iconY := startY + (contentHeight-leadHeight)/2
		r.leadingIcon.Resize(fyne.NewSize(leadWidth, leadHeight))
		r.leadingIcon.Move(fyne.NewPos(currentX, iconY))
		currentX += leadWidth + leadGap
	}

	// Position Label
	textY := startY + (contentHeight-textMin.Height)/2
	r.label.Resize(textMin)
	r.label.Move(fyne.NewPos(currentX, textY))
	currentX += textMin.Width + trailGap

	// Position Trailing Icon
	if !r.trailingIcon.Hidden {
		iconY := startY + (contentHeight-trailHeight)/2
		r.trailingIcon.Resize(fyne.NewSize(trailWidth, trailHeight))
		r.trailingIcon.Move(fyne.NewPos(currentX, iconY))
	}
}

func (r *chipRenderer) MinSize() fyne.Size {
	textMin := r.label.MinSize()
	th := r.chip.Theme()
	innerPadding := th.Size(theme.SizeNameInnerPadding)
	gapSize := innerPadding / 2

	var leadWidth, leadHeight float32
	var leadGap float32
	if !r.leadingIcon.Hidden {
		leadMin := r.leadingIcon.MinSize()
		leadWidth = leadMin.Width
		leadHeight = leadMin.Height
		if textMin.Width > 0 || !r.trailingIcon.Hidden {
			leadGap = gapSize
		}
	}

	var trailWidth, trailHeight float32
	var trailGap float32
	if !r.trailingIcon.Hidden {
		trailMin := r.trailingIcon.MinSize()
		trailWidth = trailMin.Width
		trailHeight = trailMin.Height
		if textMin.Width > 0 || !r.leadingIcon.Hidden {
			trailGap = gapSize
		}
	}

	width := leadWidth + leadGap + textMin.Width + trailGap + trailWidth + (innerPadding * 2)
	height := fyne.Max(fyne.Max(leadHeight, textMin.Height), trailHeight) + (innerPadding * 2)

	return fyne.NewSize(width, height)
}

func (r *chipRenderer) Refresh() {
	w := r.chip
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	// Update text properties
	r.label.Text = w.Text
	r.label.TextSize = th.Size(theme.SizeNameText)

	// Dynamically hide or show icons
	updateIcon := func(icon *widget.Icon, res fyne.Resource) {
		if res == nil {
			icon.Hide()
			return
		}
		icon.Show()
		if w.Disabled() {
			icon.SetResource(theme.NewDisabledResource(res))
		} else {
			icon.SetResource(res)
		}
	}

	updateIcon(r.leadingIcon, w.LeadingIcon)
	updateIcon(r.trailingIcon, w.TrailingIcon)

	if w.Disabled() {
		r.bg.StrokeColor = th.Color(theme.ColorNameDisabled, v)
		r.label.Color = th.Color(theme.ColorNameDisabled, v)
	} else {
		r.bg.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
		r.label.Color = th.Color(theme.ColorNameForeground, v)
	}

	if w.On {
		if w.Disabled() {
			r.bg.FillColor = th.Color(theme.ColorNameDisabledButton, v)
			r.bg.StrokeColor = th.Color(theme.ColorNameDisabledButton, v)
		} else {
			r.bg.FillColor = th.Color(theme.ColorNameSelection, v)
			r.bg.StrokeColor = th.Color(theme.ColorNameSelection, v)
		}
	} else {
		r.bg.FillColor = color.Transparent
	}

	if w.focused {
		r.bg.StrokeColor = th.Color(theme.ColorNameFocus, v)
	}

	r.bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder)
	r.bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)

	r.label.Refresh()
	r.leadingIcon.Refresh()
	r.trailingIcon.Refresh()
	r.bg.Refresh()
}

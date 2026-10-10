package core

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// navSelector is a drop down for switching the current entity, e.g. the character.
// It shows the entity's name and a menu with the alternatives.
type navSelector struct {
	ttwidget.ToolTipWidget

	background   *canvas.Rectangle
	hovered      bool
	label        *widget.Label
	menu         *fyne.Menu
	trailingIcon *widget.Icon
}

var _ fyne.Tappable = (*navSelector)(nil)
var _ desktop.Hoverable = (*navSelector)(nil)
var _ desktop.Cursorable = (*navSelector)(nil)

func newNavSelector() *navSelector {
	w := &navSelector{
		background:   canvas.NewRectangle(color.Transparent),
		label:        widget.NewLabel(""),
		menu:         fyne.NewMenu(""),
		trailingIcon: widget.NewIcon(theme.MenuDropDownIcon()),
	}
	w.ExtendBaseWidget(w)
	w.label.Truncation = fyne.TextTruncateEllipsis
	w.label.TextStyle.Bold = true
	w.trailingIcon.Hide()
	w.updateStyling()
	return w
}

func (w *navSelector) CreateRenderer() fyne.WidgetRenderer {
	w.updateStyling()
	return &navSelectorRenderer{w: w}
}

func (w *navSelector) Refresh() {
	w.updateStyling()
	w.background.Refresh()
	w.BaseWidget.Refresh()
}

// updateStyling mirrors the colors of [widget.Select].
func (w *navSelector) updateStyling() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.background.CornerRadius = th.Size(theme.SizeNameInputRadius)
	if w.hovered && w.hasMenu() {
		w.background.FillColor = th.Color(theme.ColorNameHover, v)
	} else {
		w.background.FillColor = th.Color(theme.ColorNameInputBackground, v)
	}
}

func (w *navSelector) SetMenu(it []*fyne.MenuItem) {
	w.menu.Items = it
	w.menu.Refresh()
	if w.hasMenu() {
		w.trailingIcon.Show()
	} else {
		w.trailingIcon.Hide()
	}
	w.Refresh()
}

// SetTitle sets the shown name. The tooltip shows the full name in case it is truncated.
func (w *navSelector) SetTitle(s string) {
	w.label.SetText(s)
	w.SetToolTip(s)
}

func (w *navSelector) Tapped(_ *fyne.PointEvent) {
	if !w.hasMenu() {
		return
	}
	xwidget.ShowPopUpMenuBelowLeading(w, w.menu)
}

func (w *navSelector) Cursor() desktop.Cursor {
	if w.hovered && w.hasMenu() {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *navSelector) MouseIn(me *desktop.MouseEvent) {
	w.ToolTipWidget.MouseIn(me)
	w.setHovered(true)
}

func (w *navSelector) MouseMoved(me *desktop.MouseEvent) {
	w.ToolTipWidget.MouseMoved(me)
}

func (w *navSelector) MouseOut() {
	w.ToolTipWidget.MouseOut()
	w.setHovered(false)
}

// setHovered updates the hover only when the state changes, since mouse events fire often.
func (w *navSelector) setHovered(v bool) {
	if w.hovered == v {
		return
	}
	w.hovered = v
	w.updateStyling()
	w.background.Refresh()
}

func (w *navSelector) hasMenu() bool {
	return len(w.menu.Items) > 0
}

// navSelectorRenderer mirrors the layout of [widget.Select].
type navSelectorRenderer struct {
	w *navSelector
}

func (r *navSelectorRenderer) Destroy() {}

func (r *navSelectorRenderer) Layout(size fyne.Size) {
	th := r.w.Theme()
	p := th.Size(theme.SizeNamePadding)
	innerPad := th.Size(theme.SizeNameInnerPadding)
	iconSize := th.Size(theme.SizeNameInlineIcon)
	r.w.background.Resize(size)
	iconPos := fyne.NewPos(size.Width-iconSize-innerPad, (size.Height-iconSize)/2)
	r.w.trailingIcon.Resize(fyne.NewSquareSize(iconSize))
	r.w.trailingIcon.Move(iconPos)
	// the label's own inner padding of innerPad leaves the same text inset as the select
	labelHeight := r.w.label.MinSize().Height
	r.w.label.Resize(fyne.NewSize(iconPos.X+innerPad-p, labelHeight))
	r.w.label.Move(fyne.NewPos(0, (size.Height-labelHeight)/2))
}

func (r *navSelectorRenderer) MinSize() fyne.Size {
	th := r.w.Theme()
	innerPad := th.Size(theme.SizeNameInnerPadding)
	s := r.w.label.MinSize()
	return fyne.NewSize(s.Width+th.Size(theme.SizeNameInlineIcon)+innerPad, s.Height)
}

func (r *navSelectorRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.w.background, r.w.label, r.w.trailingIcon}
}

func (r *navSelectorRenderer) Refresh() {
	r.Layout(r.w.Size())
	canvas.Refresh(r.w)
}

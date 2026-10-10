package xwidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// navPlaceholder shows a short text centered in the placeholder color. It starts hidden.
type navPlaceholder struct {
	widget.BaseWidget

	text *widget.RichText
}

func newNavPlaceholder() *navPlaceholder {
	p := &navPlaceholder{text: widget.NewRichText(&widget.TextSegment{Style: widget.RichTextStyle{
		Alignment: fyne.TextAlignCenter,
		ColorName: theme.ColorNamePlaceHolder,
	}})}
	p.ExtendBaseWidget(p)
	p.Hide()
	return p
}

func (p *navPlaceholder) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewCenter(p.text))
}

func (p *navPlaceholder) setText(s string) {
	p.text.Segments[0].(*widget.TextSegment).Text = s
	p.text.Refresh()
}

// ShowPopUpMenuBelowLeading shows a popup menu below a widget and aligned leading.
func ShowPopUpMenuBelowLeading(w fyne.CanvasObject, m *fyne.Menu) {
	if m == nil {
		return
	}
	pos := fyne.NewPos(0, w.Size().Height)
	widget.ShowPopUpMenuAtRelativePosition(
		m,
		fyne.CurrentApp().Driver().CanvasForObject(w),
		pos,
		w,
	)
}

// ShowPopUpMenuBelowTrailing shows a popup menu below a widget and aligned trailing.
func ShowPopUpMenuBelowTrailing(w fyne.CanvasObject, m *fyne.Menu) {
	if m == nil {
		return
	}
	pum := widget.NewPopUpMenu(m, fyne.CurrentApp().Driver().CanvasForObject(w))
	pum.ShowAtRelativePosition(
		fyne.NewPos(
			-pum.Size().Width+w.Size().Width,
			w.Size().Height,
		),
		w,
	)
}

// ShowPopUpMenuTrailingAbove shows a popup menu to the right of a widget with bottom edges aligned.
func ShowPopUpMenuTrailingAbove(w fyne.CanvasObject, m *fyne.Menu) {
	if m == nil {
		return
	}
	pum := widget.NewPopUpMenu(m, fyne.CurrentApp().Driver().CanvasForObject(w))
	pum.ShowAtRelativePosition(
		fyne.NewPos(
			w.Size().Width,
			w.Size().Height-pum.Size().Height,
		),
		w,
	)
}

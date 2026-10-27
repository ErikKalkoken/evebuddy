package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// InfoLink is a widget that shows a link for an entity that opens the infoviewer.
// It can also be cleared to show that the entity does not exist.
type InfoLink struct {
	widget.BaseWidget

	link  *widget.Hyperlink
	label *widget.Label
	iw    InfoViewer
}

func NewInfoLink(iw InfoViewer) *InfoLink {
	w := &InfoLink{
		link:  widget.NewHyperlink("", nil),
		label: widget.NewLabel("-"),
		iw:    iw,
	}
	w.ExtendBaseWidget(w)
	w.link.Truncation = fyne.TextTruncateEllipsis
	w.link.OnTapped = nil
	w.link.Hide()
	return w
}

func (w *InfoLink) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(w.link, w.label))
}

// Set shows a link to entity o. A nil o clears the widget.
func (w *InfoLink) Set(o *app.EveEntity) {
	if o == nil {
		w.Clear()
		return
	}
	w.setLink(o.Name, func() {
		w.iw.Show(o)
	})
}

// SetBloodline shows a link to bloodline o. A nil o clears the widget.
func (w *InfoLink) SetBloodline(o *app.EntityShort) {
	if o == nil {
		w.Clear()
		return
	}
	w.setLink(o.Name, func() {
		w.iw.ShowBloodline(o.ID)
	})
}

// SetLocation shows a link to location o. A nil o clears the widget.
func (w *InfoLink) SetLocation(o *app.EveLocation) {
	if o == nil {
		w.Clear()
		return
	}
	w.setLink(o.Name, func() {
		w.iw.ShowLocation(o.ID)
	})
}

// SetRace shows a link to race o. A nil o clears the widget.
func (w *InfoLink) SetRace(o *app.EveRace) {
	if o == nil {
		w.Clear()
		return
	}
	w.setLink(o.Name, func() {
		w.iw.ShowRace(o.ID)
	})
}

// setLink shows a link with text that runs onTapped.
func (w *InfoLink) setLink(text string, onTapped func()) {
	w.link.SetText(text)
	w.link.OnTapped = onTapped
	w.link.Show()
	w.label.Hide()
}

// Clear shows a placeholder to indicate that the entity does not exist.
func (w *InfoLink) Clear() {
	w.label.Show()
	w.link.Hide()
}

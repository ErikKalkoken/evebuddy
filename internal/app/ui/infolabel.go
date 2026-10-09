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

func (w *InfoLink) Set(o *app.EveEntity) {
	w.link.SetText(o.Name)
	w.link.OnTapped = func() {
		w.iw.Show(o)
	}
	w.link.Show()
	w.label.Hide()
}

func (w *InfoLink) SetBloodline(o *app.EntityShort) {
	w.link.SetText(o.Name)
	w.link.OnTapped = func() {
		w.iw.ShowBloodline(o.ID)
	}
	w.link.Show()
	w.label.Hide()
}

func (w *InfoLink) SetLocation(o *app.EveLocation) {
	w.link.SetText(o.Name)
	w.link.OnTapped = func() {
		w.iw.ShowLocation(o.ID)
	}
	w.link.Show()
	w.label.Hide()
}

func (w *InfoLink) SetRace(o *app.EveRace) {
	w.link.SetText(o.Name)
	w.link.OnTapped = func() {
		w.iw.ShowRace(o.ID)
	}
	w.link.Show()
	w.label.Hide()
}

func (w *InfoLink) Clear() {
	w.label.Show()
	w.link.Hide()
}

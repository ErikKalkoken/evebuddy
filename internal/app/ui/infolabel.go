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

// This widget is not used, because we can not both have truncation and place the icon directly next to the label.
// // InfoLabel is a widget that shows a name label and an info icon for an entity.
// // Clicking the info button opens the info viewer for that entity.
// type InfoLabel struct {
// 	widget.BaseWidget

// 	name *widget.Label
// 	icon *kxwidget.IconButton
// 	iw   InfoViewer
// }

// func NewInfoLabel(iw InfoViewer) *InfoLabel {
// 	w := &InfoLabel{
// 		name: widget.NewLabel(""),
// 		icon: kxwidget.NewIconButton(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil),
// 		iw:   iw,
// 	}
// 	w.ExtendBaseWidget(w)
// 	w.name.Truncation = fyne.TextTruncateEllipsis
// 	w.name.Hide()
// 	w.icon.Hide()
// 	return w
// }

// func (w *InfoLabel) CreateRenderer() fyne.WidgetRenderer {
// 	c := container.NewBorder(nil, nil, nil, container.NewHBox(w.icon, layout.NewSpacer()), w.name)
// 	return widget.NewSimpleRenderer(c)
// }

// func (w *InfoLabel) Set(o *app.EveEntity) {
// 	w.name.SetText(o.Name)
// 	w.icon.OnTapped = func() {
// 		w.iw.Show(o)
// 	}
// 	w.name.Show()
// 	w.icon.Show()
// }

// func (w *InfoLabel) Clear() {
// 	w.name.Hide()
// 	w.icon.Hide()
// }

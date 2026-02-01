package ui

import (
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type mailHeaderItem struct {
	widget.BaseWidget

	FallbackIcon fyne.Resource

	eis       eveEntityEIS
	from      *widget.Label
	icon      *canvas.Image
	subject   *widget.Label
	timestamp *widget.Label
}

func newMailHeaderItem(eis eveEntityEIS) *mailHeaderItem {
	subject := widget.NewLabel("")
	subject.SizeName = theme.SizeNameSubHeadingText
	subject.Truncation = fyne.TextTruncateEllipsis
	from := widget.NewLabel("")
	from.Truncation = fyne.TextTruncateEllipsis
	w := &mailHeaderItem{
		eis:          eis,
		from:         from,
		FallbackIcon: icons.Questionmark32Png,
		subject:      subject,
		timestamp:    widget.NewLabel(""),
	}
	w.icon = iwidget.NewImageFromResource(w.FallbackIcon, fyne.NewSquareSize(app.IconUnitSize))
	w.ExtendBaseWidget(w)
	return w
}

func (w *mailHeaderItem) Set(characterID int32, from *app.EveEntity, subject string, timestamp time.Time, isRead bool) {
	w.from.Text = from.Name
	w.from.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.timestamp.Text = timestamp.Format(app.VariableDateFormat(timestamp))
	w.timestamp.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.subject.Text = subject
	w.subject.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.Refresh()
	go func() {
		res, err := fetchEveEntityAvatar(w.eis, from, w.FallbackIcon)
		if err != nil {
			slog.Error("fetch eve entity avatar", "characterID", characterID, "from", from, "timestamp", timestamp, "error", err)
			res = w.FallbackIcon
		}
		fyne.Do(func() {
			w.icon.Resource = res
			w.icon.Refresh()
		})
	}()
}

func (w *mailHeaderItem) Refresh() {
	w.from.Refresh()
	w.subject.Refresh()
	w.timestamp.Refresh()
	w.BaseWidget.Refresh()
}

func (w *mailHeaderItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	first := container.New(
		layout.NewCustomPaddedLayout(0, -2*p, 0, 0),
		container.NewBorder(nil, nil, nil, w.timestamp, w.from),
	)
	second := container.New(layout.NewCustomPaddedLayout(-2*p, 0, 0, 0), w.subject)
	main := container.New(layout.NewCustomPaddedVBoxLayout(0), first, second)
	c := container.NewBorder(nil, nil, container.NewPadded(w.icon), nil, main)
	return widget.NewSimpleRenderer(c)
}

type mailHeader struct {
	widget.BaseWidget

	eis        eveEntityEIS
	from       *kxwidget.TappableLabel
	icon       *iwidget.TappableImage
	recipients *fyne.Container
	to         *widget.Label
	showInfo   func(*app.EveEntity)
	timestamp  *widget.Label
}

func newMailHeader(eis eveEntityEIS, show func(*app.EveEntity)) *mailHeader {
	from := kxwidget.NewTappableLabel("", nil)
	from.TextStyle.Bold = true
	p := theme.Padding()
	w := &mailHeader{
		eis:        eis,
		from:       from,
		recipients: container.New(layout.NewRowWrapLayoutWithCustomPadding(0, -3*p)),
		showInfo:   show,
		timestamp:  widget.NewLabel(""),
		to:         widget.NewLabel("to"),
	}
	w.ExtendBaseWidget(w)
	w.icon = iwidget.NewTappableImage(icons.BlankSvg, nil)
	w.icon.SetFillMode(canvas.ImageFillContain)
	w.icon.SetMinSize(fyne.NewSquareSize(app.IconUnitSize))
	w.to.Hide()
	return w
}

func (w *mailHeader) Set(characterID int32, from *app.EveEntity, timestamp time.Time, recipients ...*app.EveEntity) {
	w.timestamp.Text = timestamp.Format(app.DateTimeFormat)
	w.recipients.RemoveAll()
	for _, r := range recipients {
		x := kxwidget.NewTappableLabel(r.Name, func() {
			w.showInfo(r)
		})
		w.recipients.Add(x)
	}
	w.from.Text = from.Name
	w.from.OnTapped = func() {
		w.showInfo(from)
	}
	w.icon.OnTapped = func() {
		w.showInfo(from)
	}
	w.to.Show()
	w.Refresh()
	go func() {
		res, err := fetchEveEntityAvatar(w.eis, from, icons.BlankSvg)
		if err != nil {
			slog.Error("fetch eve entity avatar", "characterID", characterID, "from", from, "timestamp", timestamp, "error", err)
			res = icons.Questionmark32Png
		}
		fyne.Do(func() {
			w.icon.SetResource(res)
		})
	}()
}

func (w *mailHeader) Clear() {
	w.from.Text = ""
	w.from.OnTapped = nil
	w.recipients.RemoveAll()
	w.timestamp.Text = ""
	w.icon.SetResource(icons.BlankSvg)
	w.icon.OnTapped = nil
	w.to.Hide()
	w.Refresh()
}

func (w *mailHeader) Refresh() {
	w.from.Refresh()
	w.recipients.Refresh()
	w.timestamp.Refresh()
	w.BaseWidget.Refresh()
}

func (w *mailHeader) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	first := container.New(
		layout.NewCustomPaddedLayout(0, -2*p, 0, 0),
		container.NewHBox(w.from, w.timestamp),
	)
	second := container.NewBorder(
		nil,
		nil,
		container.NewVBox(w.to),
		nil,
		w.recipients,
	)
	main := container.New(layout.NewCustomPaddedVBoxLayout(0), first, second)
	c := container.NewBorder(nil, nil, container.NewPadded(w.icon), nil, main)
	return widget.NewSimpleRenderer(c)
}

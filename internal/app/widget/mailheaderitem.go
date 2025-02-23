package widget

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	iwidgets "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type MailHeaderItem struct {
	widget.BaseWidget

	FallbackIcon fyne.Resource

	eis        app.EveImageService
	from       *widget.Label
	icon       *canvas.Image
	subject    *iwidgets.Label
	timestamp  *widget.Label
	timeFormat string
}

func NewMailHeaderItem(eis app.EveImageService, timeFormat string) *MailHeaderItem {
	subject := iwidgets.NewLabelWithSize("", theme.SizeNameSubHeadingText)
	subject.Truncation = fyne.TextTruncateEllipsis
	from := widget.NewLabel("")
	from.Truncation = fyne.TextTruncateEllipsis
	w := &MailHeaderItem{
		eis:          eis,
		from:         from,
		FallbackIcon: icon.Questionmark32Png,
		subject:      subject,
		timestamp:    widget.NewLabel(""),
		timeFormat:   timeFormat,
	}
	w.icon = iwidgets.NewImageFromResource(w.FallbackIcon, fyne.NewSquareSize(32))
	w.ExtendBaseWidget(w)
	return w
}

func (w *MailHeaderItem) Set(from *app.EveEntity, subject string, timestamp time.Time, isRead bool) {
	w.from.Text = from.Name
	w.from.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.timestamp.Text = timestamp.Format(w.timeFormat)
	w.timestamp.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.subject.Text = subject
	w.subject.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.Refresh()
	go func() {
		w.icon.Resource = fetchEveEntityAvatar(w.eis, from, w.FallbackIcon)
		w.icon.Refresh()
	}()

}

func (w *MailHeaderItem) Refresh() {
	w.from.Refresh()
	w.subject.Refresh()
	w.timestamp.Refresh()
	w.BaseWidget.Refresh()
}

func (w *MailHeaderItem) CreateRenderer() fyne.WidgetRenderer {
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

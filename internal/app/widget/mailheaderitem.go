package widget

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	iwidgets "github.com/ErikKalkoken/evebuddy/internal/widgets"
)

type MailHeaderItem struct {
	widget.BaseWidget
	from       *widget.Label
	subject    *iwidgets.Label
	timestamp  *widget.Label
	timeFormat string
}

func NewMailHeaderItem(timeFormat string) *MailHeaderItem {
	subject := iwidgets.NewLabelWithSize("", theme.SizeNameSubHeadingText)
	subject.Truncation = fyne.TextTruncateEllipsis
	from := widget.NewLabel("")
	from.Truncation = fyne.TextTruncateEllipsis
	w := &MailHeaderItem{
		from:       from,
		subject:    subject,
		timestamp:  widget.NewLabel(""),
		timeFormat: timeFormat,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *MailHeaderItem) Set(from, subject string, timestamp time.Time, isRead bool) {
	w.from.Text = from
	w.from.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.from.Refresh()
	w.timestamp.Text = timestamp.Format(w.timeFormat)
	w.timestamp.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.timestamp.Refresh()
	w.subject.Text = subject
	w.subject.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.subject.Refresh()
}

func (w *MailHeaderItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.New(
		layout.NewCustomPaddedVBoxLayout(0),
		container.New(
			layout.NewCustomPaddedLayout(0, -2*p, 0, 0),
			container.NewBorder(nil, nil, nil, w.timestamp, w.from),
		),
		container.New(layout.NewCustomPaddedLayout(-2*p, 0, 0, 0), w.subject),
	)
	return widget.NewSimpleRenderer(c)
}

package widgets

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type MailHeaderItem struct {
	widget.BaseWidget
	from       *canvas.Text
	subject    *canvas.Text
	timestamp  *canvas.Text
	timeFormat string
}

func NewMailHeaderItem(timeFormat string) *MailHeaderItem {
	foregroundColor := theme.ForegroundColor()
	w := &MailHeaderItem{
		from:       canvas.NewText("xxxxxxxxxxxxxxx", foregroundColor),
		subject:    canvas.NewText("xxxxxxxxxxxxxxx", foregroundColor),
		timestamp:  canvas.NewText("xxxxxxxxxxxxxxx", foregroundColor),
		timeFormat: timeFormat,
	}
	w.subject.TextSize = theme.TextSize() * 1.15
	w.ExtendBaseWidget(w)
	return w
}

func (w *MailHeaderItem) Set(from, subject string, timestamp time.Time, isRead bool) {
	fg := theme.ForegroundColor()
	w.from.Text = from
	w.from.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.from.Color = fg
	w.from.Refresh()

	w.timestamp.Text = timestamp.Format(w.timeFormat)
	w.timestamp.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.timestamp.Color = fg
	w.timestamp.Refresh()

	w.subject.Text = subject
	w.subject.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.subject.Color = fg
	w.subject.Refresh()
}

func (w *MailHeaderItem) SetError(s string) {
	w.from.Text = "ERROR"
	w.subject.Color = theme.ErrorColor()
	w.subject.Text = s
	w.subject.Color = theme.ErrorColor()
	w.subject.Refresh()
}

func (w *MailHeaderItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewPadded(container.NewPadded(container.NewVBox(
		container.NewHBox(w.from, layout.NewSpacer(), w.timestamp),
		w.subject,
	)))
	return widget.NewSimpleRenderer(c)
}

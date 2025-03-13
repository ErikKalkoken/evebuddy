package widget

import (
	"log/slog"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

// TODO: Restructure UI so that services are setup before UI New()

type MailHeader struct {
	widget.BaseWidget

	showInfo   func(*app.EveEntity)
	from       *kxwidget.TappableLabel
	icon       *kxwidget.TappableImage
	recipients *kxwidget.TappableLabel
	timestamp  *widget.Label
}

func NewMailHeader(show func(*app.EveEntity)) *MailHeader {
	recipients := kxwidget.NewTappableLabel("", nil)
	recipients.Truncation = fyne.TextTruncateEllipsis
	from := kxwidget.NewTappableLabel("", nil)
	from.TextStyle.Bold = true
	w := &MailHeader{
		from:       from,
		recipients: recipients,
		showInfo:   show,
		timestamp:  widget.NewLabel(""),
	}
	w.icon = kxwidget.NewTappableImage(icon.BlankSvg, nil)
	w.icon.SetFillMode(canvas.ImageFillContain)
	w.icon.SetMinSize(fyne.NewSquareSize(32))
	w.ExtendBaseWidget(w)
	return w
}

func (w *MailHeader) Set(eis app.EveImageService, from *app.EveEntity, timestamp time.Time, recipients ...*app.EveEntity) {
	w.timestamp.Text = timestamp.Format(app.DateTimeDefaultFormat)
	rr := slices.Collect(xiter.MapSlice(recipients, func(x *app.EveEntity) string {
		return x.Name
	}))
	w.recipients.Text = "to " + strings.Join(rr, ", ")
	w.from.Text = from.Name
	w.from.OnTapped = func() {
		w.showInfo(from)
	}
	w.icon.OnTapped = func() {
		w.showInfo(from)
	}
	if len(recipients) > 0 {
		w.recipients.OnTapped = func() {
			w.showInfo(recipients[0])
		}
	}
	w.Refresh()
	go func() {
		res, err := FetchEveEntityAvatar(eis, from, icon.BlankSvg)
		if err != nil {
			slog.Error("fetch eve entity avatar", "error", err)
			res = icon.Questionmark32Png
		}
		w.icon.SetResource(res)
	}()
}

func (w *MailHeader) Clear() {
	w.from.Text = ""
	w.from.OnTapped = nil
	w.recipients.Text = ""
	w.recipients.OnTapped = nil
	w.timestamp.Text = ""
	w.icon.SetResource(icon.BlankSvg)
	w.icon.OnTapped = nil
	w.Refresh()
}

func (w *MailHeader) Refresh() {
	w.from.Refresh()
	w.recipients.Refresh()
	w.timestamp.Refresh()
	w.BaseWidget.Refresh()
}

func (w *MailHeader) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	first := container.New(
		layout.NewCustomPaddedLayout(0, -2*p, 0, 0),
		container.NewHBox(w.from, w.timestamp),
	)
	second := container.New(layout.NewCustomPaddedLayout(-2*p, 0, 0, 0), w.recipients)
	main := container.New(layout.NewCustomPaddedVBoxLayout(0), first, second)
	c := container.NewBorder(nil, nil, container.NewPadded(w.icon), nil, main)
	return widget.NewSimpleRenderer(c)
}

package character

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
	ilayout "github.com/ErikKalkoken/evebuddy/internal/layout"
)

type MailHeader struct {
	widget.BaseWidget

	showInfo   func(*app.EveEntity)
	from       *kxwidget.TappableLabel
	icon       *kxwidget.TappableImage
	recipients *fyne.Container
	timestamp  *widget.Label
	eis        app.EveImageService
}

func NewMailHeader(eis app.EveImageService, show func(*app.EveEntity)) *MailHeader {
	from := kxwidget.NewTappableLabel("", nil)
	from.TextStyle.Bold = true
	p := theme.Padding()
	w := &MailHeader{
		from:       from,
		recipients: container.New(ilayout.NewRowWrapLayoutWithCustomPadding(0, -3*p)),
		showInfo:   show,
		timestamp:  widget.NewLabel(""),
		eis:        eis,
	}
	w.icon = kxwidget.NewTappableImage(icons.BlankSvg, nil)
	w.icon.SetFillMode(canvas.ImageFillContain)
	w.icon.SetMinSize(fyne.NewSquareSize(app.IconUnitSize))
	w.ExtendBaseWidget(w)
	return w
}

func (w *MailHeader) Set(from *app.EveEntity, timestamp time.Time, recipients ...*app.EveEntity) {
	w.timestamp.Text = timestamp.Format(app.DateTimeFormat)
	w.recipients.RemoveAll()
	// p := theme.Padding()
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
	w.Refresh()
	go func() {
		res, err := FetchEveEntityAvatar(w.eis, from, icons.BlankSvg)
		if err != nil {
			slog.Error("fetch eve entity avatar", "error", err)
			res = icons.Questionmark32Png
		}
		w.icon.SetResource(res)
	}()
}

func (w *MailHeader) Clear() {
	w.from.Text = ""
	w.from.OnTapped = nil
	w.recipients.RemoveAll()
	w.timestamp.Text = ""
	w.icon.SetResource(icons.BlankSvg)
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
	second := container.NewBorder(nil, nil, container.NewVBox(widget.NewLabel("to")), nil, w.recipients)
	main := container.New(layout.NewCustomPaddedVBoxLayout(0), first, second)
	c := container.NewBorder(nil, nil, container.NewPadded(w.icon), nil, main)
	return widget.NewSimpleRenderer(c)
}

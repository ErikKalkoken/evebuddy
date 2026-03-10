package character

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type MailHeaderItem struct {
	widget.BaseWidget

	FallbackIcon fyne.Resource

	from      *widget.Label
	icon      *canvas.Image
	loadIcon  awidget.EveEntityIconLoader
	subject   *widget.Label
	timestamp *widget.Label
}

func NewMailHeaderItem(loadIcon awidget.EveEntityIconLoader) *MailHeaderItem {
	subject := widget.NewLabel("")
	subject.SizeName = theme.SizeNameSubHeadingText
	subject.Truncation = fyne.TextTruncateEllipsis
	from := widget.NewLabel("")
	from.Truncation = fyne.TextTruncateEllipsis
	w := &MailHeaderItem{
		FallbackIcon: icons.Questionmark32Png,
		from:         from,
		loadIcon:     loadIcon,
		subject:      subject,
		timestamp:    widget.NewLabel(""),
	}
	w.icon = xwidget.NewImageFromResource(w.FallbackIcon, fyne.NewSquareSize(app.IconUnitSize))
	w.icon.CornerRadius = app.IconUnitSize / 2
	w.ExtendBaseWidget(w)
	return w
}

func (w *MailHeaderItem) Set(from *app.EveEntity, subject string, timestamp time.Time, isRead bool) {
	w.from.Text = from.Name
	w.from.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.timestamp.Text = timestamp.Format(app.VariableDateFormat(timestamp))
	w.timestamp.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.subject.Text = subject
	w.subject.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.loadIcon(from, app.IconPixelSize, func(r fyne.Resource) {
		w.icon.Resource = r
		w.icon.Refresh()
	})
	w.Refresh()
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

type MailHeader struct {
	widget.BaseWidget

	from       *kxwidget.TappableLabel
	icon       *xwidget.TappableImage
	loadIcon   awidget.EveEntityIconLoader
	recipients *fyne.Container
	showInfo   func(*app.EveEntity)
	timestamp  *widget.Label
	to         *widget.Label
}

func NewMailHeader(loadIcon awidget.EveEntityIconLoader, show func(*app.EveEntity)) *MailHeader {
	from := kxwidget.NewTappableLabel("", nil)
	from.TextStyle.Bold = true
	p := theme.Padding()
	w := &MailHeader{
		from:       from,
		loadIcon:   loadIcon,
		recipients: container.New(layout.NewRowWrapLayoutWithCustomPadding(0, -3*p)),
		showInfo:   show,
		timestamp:  widget.NewLabel(""),
		to:         widget.NewLabel("to"),
	}
	w.ExtendBaseWidget(w)
	w.icon = xwidget.NewTappableImage(icons.BlankSvg, nil)
	w.icon.SetFillMode(canvas.ImageFillContain)
	w.icon.SetMinSize(fyne.NewSquareSize(app.IconUnitSize))
	w.icon.SetCornerRadius(app.IconUnitSize / 2)
	w.to.Hide()
	return w
}

func (w *MailHeader) Set(from *app.EveEntity, timestamp time.Time, recipients ...*app.EveEntity) {
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
	w.loadIcon(from, app.IconPixelSize, func(r fyne.Resource) {
		w.icon.SetResource(r)
	})
	w.Refresh()
}

func (w *MailHeader) Clear() {
	w.from.Text = ""
	w.from.OnTapped = nil
	w.recipients.RemoveAll()
	w.timestamp.Text = ""
	w.icon.SetResource(icons.BlankSvg)
	w.icon.OnTapped = nil
	w.to.Hide()
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

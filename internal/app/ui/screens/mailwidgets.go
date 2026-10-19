package screens

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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type MailHeaderItemWidget struct {
	widget.BaseWidget

	FallbackIcon fyne.Resource

	fromLabel      *widget.Label
	iconImage      *canvas.Image
	iconLoader     ui.EveEntityIconLoader
	subjectLabel   *widget.Label
	timestampLabel *widget.Label
}

func NewMailHeaderItemWidget(iconLoader ui.EveEntityIconLoader) *MailHeaderItemWidget {
	w := &MailHeaderItemWidget{
		FallbackIcon:   icons.Questionmark32Png,
		fromLabel:      widget.NewLabel(""),
		iconLoader:     iconLoader,
		subjectLabel:   widget.NewLabel(""),
		timestampLabel: widget.NewLabel(""),
	}
	w.fromLabel.Truncation = fyne.TextTruncateEllipsis
	w.iconImage = xwidget.NewImageFromResource(w.FallbackIcon, fyne.NewSquareSize(ui.IconUnitSize))
	w.iconImage.CornerRadius = ui.IconUnitSize / 2
	w.subjectLabel.Truncation = fyne.TextTruncateEllipsis
	w.ExtendBaseWidget(w)
	return w
}

func (w *MailHeaderItemWidget) Set(from *app.EveEntity, subject string, timestamp time.Time, isRead bool) {
	w.fromLabel.Text = from.NameOrZero()
	w.fromLabel.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.timestampLabel.Text = timestamp.Format(app.VariableDateFormat(timestamp))
	w.timestampLabel.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.subjectLabel.Text = subject
	w.subjectLabel.TextStyle = fyne.TextStyle{Bold: !isRead}
	w.iconLoader(from, ui.IconPixelSize, func(r fyne.Resource) {
		w.iconImage.Resource = r
		w.iconImage.Refresh()
	})
	w.Refresh()
}

func (w *MailHeaderItemWidget) Refresh() {
	fyne.Do(func() {
		w.fromLabel.Refresh()
		w.subjectLabel.Refresh()
		w.timestampLabel.Refresh()
		w.BaseWidget.Refresh()
	})
}

func (w *MailHeaderItemWidget) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	first := container.New(
		layout.NewCustomPaddedLayout(0, -2*p, 0, 0),
		container.NewBorder(nil, nil, nil, w.timestampLabel, w.fromLabel),
	)
	second := container.New(layout.NewCustomPaddedLayout(-2*p, 0, 0, 0), w.subjectLabel)
	main := container.New(layout.NewCustomPaddedVBoxLayout(0), first, second)
	c := container.NewBorder(nil, nil, container.NewPadded(w.iconImage), nil, main)
	return widget.NewSimpleRenderer(c)
}

type MailHeaderWidget struct {
	widget.BaseWidget

	fromLabel      *kxwidget.TappableLabel
	iconImage      *xwidget.TappableImage
	loadIcon       ui.EveEntityIconLoader
	recipients     *fyne.Container
	showInfo       func(*app.EveEntity)
	timestampLabel *widget.Label
	toLabel        *widget.Label
}

func NewMailHeaderWidget(loadIcon ui.EveEntityIconLoader, show func(*app.EveEntity)) *MailHeaderWidget {
	p := theme.Padding()
	w := &MailHeaderWidget{
		fromLabel:      kxwidget.NewTappableLabel("", nil),
		loadIcon:       loadIcon,
		recipients:     container.New(layout.NewRowWrapLayoutWithCustomPadding(0, -3*p)),
		showInfo:       show,
		timestampLabel: widget.NewLabel(""),
		toLabel:        widget.NewLabel("to"),
	}
	w.ExtendBaseWidget(w)
	w.fromLabel.TextStyle.Bold = true
	w.iconImage = xwidget.NewTappableImage(icons.BlankSvg, nil)
	w.iconImage.SetFillMode(canvas.ImageFillContain)
	w.iconImage.SetMinSize(fyne.NewSquareSize(ui.IconUnitSize))
	w.iconImage.CornerRadius = ui.IconUnitSize / 2
	w.toLabel.Hide()
	return w
}

func (w *MailHeaderWidget) Set(from *app.EveEntity, timestamp time.Time, owner *app.EveEntity, recipients ...*app.EveEntity) {
	w.timestampLabel.Text = timestamp.Format(app.DateTimeFormat)
	w.recipients.RemoveAll()
	for _, r := range recipients {
		x := kxwidget.NewTappableLabel(r.Name, func() {
			w.showInfo(r)
		})
		w.recipients.Add(x)
	}
	if owner != nil {
		x := xwidget.NewTappableLabel("["+owner.Name+"]", func() {
			w.showInfo(owner)
		})
		x.SetToolTip("Account: " + owner.Name)
		w.recipients.Add(x)
	}
	w.fromLabel.Text = from.Name
	w.fromLabel.OnTapped = func() {
		w.showInfo(from)
	}
	w.iconImage.OnTapped = func() {
		w.showInfo(from)
	}
	w.toLabel.Show()
	w.loadIcon(from, ui.IconPixelSize, func(r fyne.Resource) {
		w.iconImage.SetResource(r)
	})
	w.Refresh()
}

func (w *MailHeaderWidget) Clear() {
	w.fromLabel.Text = ""
	w.fromLabel.OnTapped = nil
	w.recipients.RemoveAll()
	w.timestampLabel.Text = ""
	w.iconImage.SetResource(icons.BlankSvg)
	w.iconImage.OnTapped = nil
	w.toLabel.Hide()
	w.Refresh()
}

func (w *MailHeaderWidget) Refresh() {
	w.fromLabel.Refresh()
	w.recipients.Refresh()
	w.timestampLabel.Refresh()
	w.BaseWidget.Refresh()
}

func (w *MailHeaderWidget) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	first := container.New(
		layout.NewCustomPaddedLayout(0, -2*p, 0, 0),
		container.NewHBox(w.fromLabel, w.timestampLabel),
	)
	second := container.NewBorder(
		nil,
		nil,
		container.NewVBox(w.toLabel),
		nil,
		w.recipients,
	)
	main := container.New(layout.NewCustomPaddedVBoxLayout(0), first, second)
	c := container.NewBorder(nil, nil, container.NewPadded(w.iconImage), nil, main)
	return widget.NewSimpleRenderer(c)
}

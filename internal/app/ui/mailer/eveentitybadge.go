package mailer

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type eveEntityBadge struct {
	widget.DisableableWidget

	OnTapped func()

	ee           *app.EveEntity
	fallbackIcon fyne.Resource
	hovered      bool
	loadIcon     ui.EveEntityIconLoader
}

var _ fyne.Tappable = (*eveEntityBadge)(nil)
var _ desktop.Hoverable = (*eveEntityBadge)(nil)

func newEveEntityBadge(ee *app.EveEntity, loadIcon ui.EveEntityIconLoader) *eveEntityBadge {
	w := &eveEntityBadge{
		ee:           ee,
		fallbackIcon: icons.Questionmark32Png,
		loadIcon:     loadIcon,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *eveEntityBadge) CreateRenderer() fyne.WidgetRenderer {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	p := th.Size(theme.SizeNamePadding)

	name := widget.NewLabel(w.ee.Name)
	if w.Disabled() {
		name.Importance = widget.LowImportance
	}
	icon := xwidget.NewImageFromResource(
		w.fallbackIcon,
		fyne.NewSquareSize(th.Size(theme.SizeNameInlineIcon)),
	)
	rect := canvas.NewRectangle(color.Transparent)
	rect.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	rect.StrokeWidth = 1
	rect.CornerRadius = 10
	c := container.New(layout.NewCustomPaddedLayout(p, p, 0, 0),
		container.NewHBox(
			container.NewStack(
				rect,
				container.New(layout.NewCustomPaddedLayout(-p, -p, 0, 0),
					container.NewHBox(
						container.New(layout.NewCustomPaddedLayout(0, 0, p, 0), icon), name,
					))),
		),
	)
	w.loadIcon(w.ee, ui.IconPixelSize, func(r fyne.Resource) {
		icon.Resource = r
		icon.Refresh()
	})
	return widget.NewSimpleRenderer(c)
}

func (w *eveEntityBadge) Tapped(_ *fyne.PointEvent) {
	if w.Disabled() {
		return
	}
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *eveEntityBadge) TappedSecondary(_ *fyne.PointEvent) {
}

// Cursor returns the cursor type of this widget
func (w *eveEntityBadge) Cursor() desktop.Cursor {
	if !w.Disabled() && w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (w *eveEntityBadge) MouseIn(_ *desktop.MouseEvent) {
	if w.Disabled() {
		return
	}
	w.hovered = true
}

func (w *eveEntityBadge) MouseMoved(*desktop.MouseEvent) {
	// needed to satisfy the interface only
}

// MouseOut is a hook that is called if the mouse pointer leaves the element.
func (w *eveEntityBadge) MouseOut() {
	if w.Disabled() {
		return
	}
	w.hovered = false
}

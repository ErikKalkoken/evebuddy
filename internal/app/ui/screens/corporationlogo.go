package screens

import (
	"context"
	"log/slog"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type CorporationLogo struct {
	widget.BaseWidget

	corporation atomic.Pointer[app.Corporation]
	image       *canvas.Image
	u           baseUI
}

func NewCorporationLogo(u baseUI) *CorporationLogo {
	a := &CorporationLogo{
		image: xwidget.NewImageFromResource(icons.Corporationplaceholder64Png, fyne.NewSquareSize(PortraitDesktopUnitSize)),
		u:     u,
	}
	a.ExtendBaseWidget(a)

	a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		a.update(ctx)
	})
	return a
}

func (a *CorporationLogo) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(container.NewPadded(a.image))
	return widget.NewSimpleRenderer(c)
}

func (a *CorporationLogo) update(_ context.Context) {
	id := a.corporation.Load().IDOrZero()
	if id == 0 {
		fyne.Do(func() {
			a.image.Resource = icons.Corporationplaceholder64Png
			a.image.Refresh()
		})
		return
	}
	r, err := a.u.EVEImage().CorporationLogo(id, PortraitDesktopPixelSize)
	if err != nil {
		slog.Error("Failed to load character portrait", "characterID", id, "error", err)
		fyne.Do(func() {
			a.image.Resource = icons.Corporationplaceholder64Png
			a.image.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.image.Resource = r
		a.image.Refresh()
	})
}

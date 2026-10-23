package screens

import (
	"context"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type CorporationLogo struct {
	widget.BaseWidget

	corporation atomic.Pointer[app.Corporation]
	logo        *corporationLogoWidget
	u           baseUI
}

func NewCorporationLogo(u baseUI) *CorporationLogo {
	a := &CorporationLogo{
		logo: newCorporationLogoWithLogos(
			u.EVEImage().CorporationLogoAsync,
			u.EVEImage().AllianceLogoAsync,
			u.EVEImage().FactionLogoAsync,
			false,
			u.InfoViewer().Show,
		),
		u: u,
	}
	a.ExtendBaseWidget(a)

	// Signals
	a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		a.update(ctx)
	})
	a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		if a.corporation.Load().IDOrZero() != arg.CorporationID {
			return
		}
		if arg.Section != app.SectionCorporationMembers {
			return
		}
		a.update(ctx)
	})
	return a
}

func (a *CorporationLogo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewVBox(container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, p), a.logo))
	return widget.NewSimpleRenderer(c)
}

func (a *CorporationLogo) update(_ context.Context) {
	c := a.corporation.Load()
	if c == nil {
		fyne.Do(func() {
			a.logo.reset()
		})
		return
	}
	fyne.Do(func() {
		a.logo.set(c.ID, c.EveCorporation.Alliance, c.EveCorporation.Faction)
	})
}

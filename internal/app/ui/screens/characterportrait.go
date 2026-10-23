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

type CharacterPortrait struct {
	widget.BaseWidget

	character atomic.Pointer[app.Character]
	portrait  *characterPortraitWidget
	u         baseUI
}

func NewCharacterPortrait(u baseUI) *CharacterPortrait {
	a := &CharacterPortrait{
		portrait: newCharacterPortraitWithLogos(
			u.EVEImage().CharacterPortraitAsync,
			u.EVEImage().CorporationLogoAsync,
			u.EVEImage().AllianceLogoAsync,
			false,
			u.InfoViewer().Show,
		),
		u: u,
	}
	a.ExtendBaseWidget(a)

	// Signals
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		characterID := a.character.Load().IDOrZero()
		if characterID == 0 {
			return
		}
		if arg.Section == app.SectionEveCharacters && arg.Changed.Contains(characterID) {
			a.update(ctx)
		}
	})
	return a
}

func (a *CharacterPortrait) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewVBox(container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, p), a.portrait))
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterPortrait) update(_ context.Context) {
	c := a.character.Load()
	if c == nil || c.EveCharacter == nil {
		fyne.Do(func() {
			a.portrait.reset()
		})
		return
	}
	fyne.Do(func() {
		a.portrait.set(c.ID, c.EveCharacter.Corporation, c.EveCharacter.Alliance)
	})
}

package ui

import (
	"context"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/uiservices"
)

// CharacterBiography shows the attributes for the current character.
type CharacterBiography struct {
	widget.BaseWidget

	body      *widget.Label
	character atomic.Pointer[app.Character]
	u         uiservices.UIServices
}

func NewCharacterBiography(u uiservices.UIServices) *CharacterBiography {
	body := widget.NewLabel("")
	body.Wrapping = fyne.TextWrapWord
	body.Selectable = true
	a := &CharacterBiography{
		body: body,
		u:    u,
	}
	a.ExtendBaseWidget(a)
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		characterID := characterIDOrZero(a.character.Load())
		if characterID == 0 {
			return
		}
		if arg.Section == app.SectionEveCharacters && arg.Changed.Contains(characterID) {
			a.update(ctx)
		}
	})
	return a
}

func (a *CharacterBiography) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVScroll(a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterBiography) update(_ context.Context) {
	c := a.character.Load()
	if c == nil || c.EveCharacter == nil {
		fyne.Do(func() {
			a.body.Text = "Waiting for character data to be loaded..."
			a.body.Importance = widget.WarningImportance
			a.body.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.body.Text = c.EveCharacter.DescriptionPlain()
		a.body.Importance = widget.MediumImportance
		a.body.Refresh()
	})
}

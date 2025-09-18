package ui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// characterBiography shows the attributes for the current character.
type characterBiography struct {
	widget.BaseWidget

	body      *widget.Label
	character *app.Character
	u         *baseUI
}

func newCharacterBiography(u *baseUI) *characterBiography {
	text := widget.NewLabel("")
	text.Wrapping = fyne.TextWrapWord
	a := &characterBiography{
		body: text,
		u:    u,
	}
	a.ExtendBaseWidget(a)
	a.u.currentCharacterExchanged.AddListener(
		func(_ context.Context, c *app.Character) {
			a.character = c
		},
	)
	a.u.generalSectionChanged.AddListener(
		func(_ context.Context, arg generalSectionUpdated) {
			characterID := characterIDOrZero(a.character)
			if characterID == 0 {
				return
			}
			if arg.section == app.SectionEveCharacters && arg.changed.Contains(characterID) {
				a.update()
			}
		},
	)
	return a
}

func (a *characterBiography) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVScroll(a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *characterBiography) update() {
	if a.character == nil || a.character.EveCharacter == nil {
		fyne.Do(func() {
			a.body.Text = "Waiting for character data to be loaded..."
			a.body.Importance = widget.WarningImportance
			a.body.Refresh()
		})
	} else {
		fyne.Do(func() {
			a.body.Text = a.character.EveCharacter.DescriptionPlain()
			a.body.Importance = widget.MediumImportance
			a.body.Refresh()
		})
	}
}

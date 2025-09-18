package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type characterSheet struct {
	widget.BaseWidget

	character   *app.Character
	born        *widget.Label
	factionLogo *kxwidget.TappableImage
	home        *widget.Hyperlink
	name        *widget.Hyperlink
	portrait    *kxwidget.TappableImage
	race        *widget.Hyperlink
	security    *widget.Label
	sp          *widget.Label
	u           *baseUI
	wealth      *widget.Label
}

func newCharacterSheet(u *baseUI) *characterSheet {
	makeLogo := func() *kxwidget.TappableImage {
		ti := kxwidget.NewTappableImage(icons.BlankSvg, nil)
		ti.SetFillMode(canvas.ImageFillContain)
		ti.SetMinSize(fyne.NewSquareSize(app.IconUnitSize))
		return ti
	}

	portrait := kxwidget.NewTappableImage(icons.BlankSvg, nil)
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(fyne.NewSquareSize(128))
	a := &characterSheet{
		born:        widget.NewLabel("?"),
		factionLogo: makeLogo(),
		home:        widget.NewHyperlink("", nil),
		name:        widget.NewHyperlink("", nil),
		portrait:    portrait,
		race:        widget.NewHyperlink("", nil),
		security:    widget.NewLabel("?"),
		sp:          widget.NewLabel("?"),
		u:           u,
		wealth:      widget.NewLabel("?"),
	}
	a.ExtendBaseWidget(a)

	a.u.currentCharacterExchanged.AddListener(
		func(_ context.Context, c *app.Character) {
			a.character = c
		},
	)
	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character) != arg.characterID {
			return
		}
		switch arg.section {
		case
			app.SectionCharacterAssets,
			app.SectionCharacterRoles,
			app.SectionCharacterSkills,
			app.SectionCharacterWalletBalance:
			a.update()
		}
	})
	a.u.generalSectionChanged.AddListener(func(_ context.Context, arg generalSectionUpdated) {
		if a.character == nil {
			return
		}
		characterID := characterIDOrZero(a.character)
		switch arg.section {
		case app.SectionEveCharacters:
			if arg.changed.Contains(characterID) {
				a.update()
			}
		case app.SectionEveCorporations:
			if arg.changed.Contains(a.character.EveCharacter.Corporation.ID) {
				a.update()
			}
		case app.SectionEveMarketPrices:
			a.update()
		}
	})
	return a
}

func (a *characterSheet) update() {
	c := a.character
	if c == nil || c.EveCharacter == nil {
		fyne.Do(func() {
			a.name.Text = "No character..."
			a.name.OnTapped = nil
			a.name.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.name.SetText(c.EveCharacter.Name)
		a.name.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityCharacter, c.ID)
		}
		a.portrait.OnTapped = a.name.OnTapped

		a.race.SetText(c.EveCharacter.Race.Name)
		a.race.OnTapped = func() {
			a.u.ShowRaceInfoWindow(c.EveCharacter.Race.ID)
		}

		a.born.SetText(c.EveCharacter.Birthday.Format(app.DateTimeFormat))
		a.security.SetText(fmt.Sprintf("%.1f", c.EveCharacter.SecurityStatus))
	})
	iwidget.RefreshTappableImageAsync(a.portrait, func() (fyne.Resource, error) {
		return a.u.eis.CharacterPortrait(c.ID, 512)
	})
	fyne.Do(func() {
		if c.Home == nil {
			a.home.SetText("?")
			a.home.OnTapped = nil
			return
		}
		a.home.SetText(c.Home.DisplayName())
		a.home.OnTapped = func() {
			a.u.ShowLocationInfoWindow(c.Home.ID)
		}
		a.home.Refresh()
	})
	fyne.Do(func() {
		a.sp.SetText(ihumanize.OptionalWithComma(c.TotalSP, "?"))
		if c.AssetValue.IsEmpty() || c.WalletBalance.IsEmpty() {
			a.wealth.SetText("?")
			return
		}
		v := c.AssetValue.ValueOrZero() + c.WalletBalance.ValueOrZero()
		a.wealth.SetText(humanize.Comma(int64(v)) + " ISK")
	})
	fyne.Do(func() {
		if c.EveCharacter.Faction == nil {
			a.factionLogo.Hide()
			return
		}
		a.factionLogo.Show()
		a.factionLogo.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityFaction, c.EveCharacter.Faction.ID)
		}
	})
	if c.EveCharacter.Faction != nil {
		iwidget.RefreshTappableImageAsync(a.factionLogo, func() (fyne.Resource, error) {
			return a.u.eis.FactionLogo(c.EveCharacter.Faction.ID, app.IconPixelSize)
		})
	}
}

func (a *characterSheet) CreateRenderer() fyne.WidgetRenderer {
	main := widget.NewForm(
		widget.NewFormItem("Name", a.name),
		widget.NewFormItem("Born", a.born),
		widget.NewFormItem("Race", a.race),
		widget.NewFormItem("Wealth", a.wealth),
		widget.NewFormItem("Security Status", a.security),
		widget.NewFormItem("Home Station", a.home),
		widget.NewFormItem("Total Skill Points", a.sp),
	)
	main.Orientation = widget.Adaptive

	portraitDesktop := container.NewVBox(container.NewPadded(a.portrait))
	c := container.NewBorder(
		nil,
		nil,
		nil,
		portraitDesktop,
		container.NewVBox(
			main,
			container.NewHBox(
				container.NewPadded(a.factionLogo)),
		),
	)
	if !a.u.isDesktop {
		portraitDesktop.Hide()
	}
	return widget.NewSimpleRenderer(c)
}

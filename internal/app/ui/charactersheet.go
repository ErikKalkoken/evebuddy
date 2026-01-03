package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type characterSheet struct {
	widget.BaseWidget

	born        *widget.Label
	character   atomic.Pointer[app.Character]
	faction     *widget.Hyperlink
	home        *widget.Hyperlink
	lastLoginAt *widget.Label
	location    *widget.Hyperlink
	name        *widget.Hyperlink
	portrait    *iwidget.TappableImage
	race        *widget.Hyperlink
	security    *widget.Label
	ship        *widget.Hyperlink
	skillpoints *widget.Label
	tags        *widget.Label
	u           *baseUI
	wealth      *widget.Label
}

func newCharacterSheet(u *baseUI) *characterSheet {
	makeHyperLink := func() *widget.Hyperlink {
		return widget.NewHyperlink("?", nil)
	}
	makeLabel := func() *widget.Label {
		return widget.NewLabel("?")
	}
	portrait := iwidget.NewTappableImage(icons.Characterplaceholder64Jpeg, nil)
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(fyne.NewSquareSize(128))
	portrait.SetToolTip("Show details")
	a := &characterSheet{
		born:        makeLabel(),
		faction:     makeHyperLink(),
		home:        makeHyperLink(),
		lastLoginAt: makeLabel(),
		location:    makeHyperLink(),
		name:        makeHyperLink(),
		portrait:    portrait,
		race:        makeHyperLink(),
		security:    makeLabel(),
		ship:        makeHyperLink(),
		skillpoints: makeLabel(),
		tags:        makeLabel(),
		u:           u,
		wealth:      makeLabel(),
	}
	a.ExtendBaseWidget(a)

	a.u.currentCharacterExchanged.AddListener(func(_ context.Context, c *app.Character) {
		a.character.Store(c)
		a.update()
	})
	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.characterID {
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
		c := a.character.Load()
		if c == nil {
			return
		}
		characterID := characterIDOrZero(c)
		switch arg.section {
		case app.SectionEveCharacters:
			if arg.changed.Contains(characterID) {
				a.update()
			}
		case app.SectionEveCorporations:
			if arg.changed.Contains(c.EveCharacter.Corporation.ID) {
				a.update()
			}
		case app.SectionEveMarketPrices:
			a.update()
		}
	})
	return a
}

// TODO: Group information

func (a *characterSheet) CreateRenderer() fyne.WidgetRenderer {
	main := widget.NewForm(
		widget.NewFormItem("Name", a.name),
		widget.NewFormItem("Born", a.born),
		widget.NewFormItem("Race", a.race),
		widget.NewFormItem("Faction", a.faction),
		widget.NewFormItem("Home Station", a.home),
		widget.NewFormItem("Security Status", a.security),
		widget.NewFormItem("Wealth", a.wealth),
		widget.NewFormItem("Total Skill Points", a.skillpoints),
		widget.NewFormItem("Ship", a.ship),
		widget.NewFormItem("Location", a.location),
		widget.NewFormItem("Last Login", a.lastLoginAt),
		widget.NewFormItem("Tags", a.tags),
	)
	main.Orientation = widget.Adaptive

	portrait := container.NewVBox(
		container.NewPadded(a.portrait),
	)
	c := container.NewBorder(
		nil,
		nil,
		nil,
		portrait,
		container.NewVScroll(main),
	)
	if a.u.isMobile {
		portrait.Hide()
	}
	return widget.NewSimpleRenderer(c)
}

func (a *characterSheet) update() {
	c := a.character.Load()
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

		a.lastLoginAt.SetText(c.LastLoginAt.StringFunc("?", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))
	})
	iwidget.RefreshTappableImageAsync(a.portrait, func() (fyne.Resource, error) {
		return a.u.eis.CharacterPortrait(c.ID, 512)
	})
	fyne.Do(func() {
		if c.Location == nil {
			a.location.SetText("?")
			a.location.OnTapped = nil
			return
		}
		a.location.SetText(c.Location.DisplayName())
		a.location.OnTapped = func() {
			a.u.ShowLocationInfoWindow(c.Location.ID)
		}
	})
	fyne.Do(func() {
		if c.Ship == nil {
			a.ship.SetText("?")
			a.ship.OnTapped = nil
			return
		}
		a.ship.SetText(c.Ship.Name)
		a.ship.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityInventoryType, c.Ship.ID)
		}
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
	})
	fyne.Do(func() {
		a.skillpoints.SetText(ihumanize.OptionalWithComma(c.TotalSP, "?"))
		if c.AssetValue.IsEmpty() || c.WalletBalance.IsEmpty() {
			a.wealth.SetText("?")
			return
		}
		v := c.AssetValue.ValueOrZero() + c.WalletBalance.ValueOrZero()
		a.wealth.SetText(humanize.Comma(int64(v)) + " ISK")
	})
	fyne.Do(func() {
		if c.EveCharacter.Faction == nil {
			a.faction.SetText("")
			return
		}
		a.faction.SetText(c.EveCharacter.FactionName())
		a.faction.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityFaction, c.EveCharacter.Faction.ID)
		}
	})

	var s string
	tags, err := a.u.cs.ListTagsForCharacter(context.Background(), c.ID)
	if err != nil {
		slog.Error("character sheet: update", "characterID", c.ID, "error", "err")
		s = "?"
	} else {
		if tags.Size() == 0 {
			s = "-"
		} else {
			s = strings.Join(slices.Sorted(tags.All()), ", ")
		}
	}
	fyne.Do(func() {
		a.tags.SetText(s)
	})
}

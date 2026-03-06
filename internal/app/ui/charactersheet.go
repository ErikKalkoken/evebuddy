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
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
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
	portrait    *xwidget.TappableImage
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
		x := widget.NewHyperlink("?", nil)
		x.Truncation = fyne.TextTruncateEllipsis
		return x
	}
	makeLabel := func() *widget.Label {
		x := widget.NewLabel("?")
		x.Selectable = true
		x.Truncation = fyne.TextTruncateEllipsis
		return x
	}
	portrait := xwidget.NewTappableImage(icons.Characterplaceholder64Jpeg, nil)
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

	// Signals
	a.u.signals.CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.signals.CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.CharacterID {
			return
		}
		switch arg.Section {
		case
			app.SectionCharacterAssets,
			app.SectionCharacterRoles,
			app.SectionCharacterShip,
			app.SectionCharacterSkills,
			app.SectionCharacterWalletBalance:
			a.update(ctx)
		}
	})
	a.u.signals.EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		c := a.character.Load()
		if c == nil {
			return
		}
		characterID := characterIDOrZero(c)
		switch arg.Section {
		case app.SectionEveCharacters:
			if arg.Changed.Contains(characterID) {
				a.update(ctx)
			}
		case app.SectionEveCorporations:
			if arg.Changed.Contains(c.EveCharacter.Corporation.ID) {
				a.update(ctx)
			}
		case app.SectionEveMarketPrices:
			a.update(ctx)
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
	if app.IsMobile() {
		portrait.Hide()
	}
	return widget.NewSimpleRenderer(c)
}

func (a *characterSheet) update(ctx context.Context) {
	setName := func(s string) {
		fyne.Do(func() {
			a.name.Text = s
			a.name.OnTapped = nil
			a.name.Refresh()
		})
	}
	c := a.character.Load()
	if c == nil || c.EveCharacter == nil {
		setName("No character...")
		return
	}
	c2, err := a.u.cs.GetCharacter(ctx, c.ID)
	if err != nil {
		slog.Error("Failed to fetch character for sheet", "err", err)
		setName("ERROR: " + app.ErrorDisplay(err))
		return
	} else {
		a.character.Store(c2)
		c = c2
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
		a.security.SetText(c.EveCharacter.SecurityStatus.StringFunc("?", func(v float64) string {
			return fmt.Sprintf("%.1f", v)
		}))

		a.lastLoginAt.SetText(c.LastLoginAt.StringFunc("?", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))
	})
	fyne.Do(func() {
		a.u.eis.CharacterPortraitAsync(c.ID, 512, func(r fyne.Resource) {
			a.portrait.SetResource(r)
		})
		el, ok := c.Location.Value()
		if !ok {
			a.location.SetText("?")
			a.location.OnTapped = nil
			return
		}
		a.location.SetText(el.DisplayName())
		a.location.OnTapped = func() {
			a.u.ShowLocationInfoWindow(el.ID)
		}
	})
	fyne.Do(func() {
		ship, ok := c.Ship.Value()
		if !ok {
			a.ship.SetText("?")
			a.ship.OnTapped = nil
			return
		}
		a.ship.SetText(ship.Name)
		a.ship.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityInventoryType, ship.ID)
		}
	})
	fyne.Do(func() {
		home, ok := c.Home.Value()
		if !ok {
			a.home.SetText("?")
			a.home.OnTapped = nil
			return
		}
		a.home.SetText(home.DisplayName())
		a.home.OnTapped = func() {
			a.u.ShowLocationInfoWindow(home.ID)
		}
	})
	fyne.Do(func() {
		a.skillpoints.SetText(ihumanize.OptionalWithComma(c.TrainedSP, "?"))
		v := optional.Sum(c.AssetValue, c.WalletBalance)
		a.wealth.SetText(v.StringFunc("?", func(v float64) string {
			return humanize.Comma(int64(v))
		}) + " ISK")
	})
	fyne.Do(func() {
		faction, ok := c.EveCharacter.Faction.Value()
		if !ok {
			a.faction.SetText("")
			return
		}
		a.faction.SetText(faction.Name)
		a.faction.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityFaction, faction.ID)
		}
	})

	var s string
	tags, err := a.u.cs.ListTagsForCharacter(ctx, c.ID)
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

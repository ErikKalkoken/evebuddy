package screens

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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type CharacterSheet struct {
	widget.BaseWidget

	bloodlineLink    *ui.InfoLink
	bornLabel        *widget.Label
	character        atomic.Pointer[app.Character]
	factionLink      *ui.InfoLink
	homeLink         *ui.InfoLink
	lastLoginAtLabel *widget.Label
	locationLink     *ui.InfoLink
	nameLink         *ui.InfoLink
	portraitImage    *xwidget.TappableImage
	raceLink         *ui.InfoLink
	securityLabel    *widget.Label
	shipLink         *ui.InfoLink
	skillpointsLabel *widget.Label
	tagsLabel        *widget.Label
	u                baseUI
	updateRun        latestRun
	wealthLabel      *widget.Label
}

func NewCharacterSheet(u baseUI) *CharacterSheet {
	makeLabel := func() *widget.Label {
		x := widget.NewLabel("?")
		x.Selectable = true
		x.Truncation = fyne.TextTruncateEllipsis
		return x
	}
	makeInfoLink := func() *ui.InfoLink {
		return ui.NewInfoLink(u.InfoViewer())
	}
	portrait := xwidget.NewTappableImage(icons.Characterplaceholder64Jpeg, nil)
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(fyne.NewSquareSize(128))
	portrait.SetToolTip("Show details")
	a := &CharacterSheet{
		bloodlineLink:    makeInfoLink(),
		bornLabel:        makeLabel(),
		factionLink:      makeInfoLink(),
		homeLink:         makeInfoLink(),
		lastLoginAtLabel: makeLabel(),
		locationLink:     makeInfoLink(),
		nameLink:         makeInfoLink(),
		portraitImage:    portrait,
		raceLink:         makeInfoLink(),
		securityLabel:    makeLabel(),
		shipLink:         makeInfoLink(),
		skillpointsLabel: makeLabel(),
		tagsLabel:        makeLabel(),
		u:                u,
		wealthLabel:      makeLabel(),
	}
	a.ExtendBaseWidget(a)

	// Signals
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
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
	a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		c := a.character.Load()
		if c == nil {
			return
		}
		characterID := c.IDOrZero()
		switch arg.Section {
		case app.SectionEveCharacters:
			if arg.Changed.Contains(characterID) {
				a.update(ctx)
			}
		case app.SectionEveCorporations:
			if c.EveCharacter != nil && arg.Changed.Contains(c.EveCharacter.Corporation.ID) {
				a.update(ctx)
			}
		case app.SectionEveMarketPrices:
			a.update(ctx)
		}
	})
	return a
}

// TODO: Group information

func (a *CharacterSheet) CreateRenderer() fyne.WidgetRenderer {
	main := widget.NewForm(
		widget.NewFormItem("Name", a.nameLink),
		widget.NewFormItem("Born", a.bornLabel),
		widget.NewFormItem("Race", a.raceLink),
		widget.NewFormItem("Bloodline", a.bloodlineLink),
		widget.NewFormItem("Faction", a.factionLink),
		widget.NewFormItem("Home Station", a.homeLink),
		widget.NewFormItem("Security Status", a.securityLabel),
		widget.NewFormItem("Wealth", a.wealthLabel),
		widget.NewFormItem("Total Skill Points", a.skillpointsLabel),
		widget.NewFormItem("Ship", a.shipLink),
		widget.NewFormItem("Location", a.locationLink),
		widget.NewFormItem("Last Login", a.lastLoginAtLabel),
		widget.NewFormItem("Tags", a.tagsLabel),
	)
	main.Orientation = widget.Adaptive

	portrait := container.NewVBox(
		container.NewPadded(a.portraitImage),
	)
	c := container.NewBorder(
		nil,
		nil,
		nil,
		portrait,
		container.NewVScroll(main),
	)
	if a.u.IsMobile() {
		portrait.Hide()
	}
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterSheet) update(ctx context.Context) {
	isLatest := a.updateRun.start()
	clearAll := func() {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.bloodlineLink.Clear()
			a.bornLabel.SetText("")
			a.factionLink.Clear()
			a.homeLink.Clear()
			a.lastLoginAtLabel.SetText("")
			a.locationLink.Clear()
			a.nameLink.Clear()
			a.portraitImage.SetResource(icons.BlankSvg)
			a.raceLink.Clear()
			a.securityLabel.SetText("")
			a.shipLink.Clear()
			a.skillpointsLabel.SetText("")
			a.tagsLabel.SetText("")
			a.wealthLabel.SetText("")
		})
	}
	old := a.character.Load()
	if old == nil || old.EveCharacter == nil {
		clearAll()
		return
	}
	c, err := a.u.Character().GetCharacter(ctx, old.ID)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to fetch character for sheet", "err", err)
		clearAll()
		return
	}
	if !a.character.CompareAndSwap(old, c) {
		return // character changed meanwhile; that change runs its own update
	}
	var home *app.EveLocation
	if v, ok := c.HomeLocationID.Value(); ok {
		el, err := a.u.EVEUniverse().GetLocation(ctx, v)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to fetch home for character", "characterID", c.ID, "err", err)
		} else {
			home = el
		}
	}
	var location *app.EveLocation
	if v, ok := c.LocationID.Value(); ok {
		el, err := a.u.EVEUniverse().GetLocation(ctx, v)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to fetch location for character", "characterID", c.ID, "err", err)
		} else {
			location = el
		}
	}
	var shipType *app.EveType
	if v, ok := c.ShipTypeID.Value(); ok {
		et, err := a.u.EVEUniverse().GetType(ctx, v)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to fetch ship type for character", "characterID", c.ID, "err", err)
		} else {
			shipType = et
		}
	}

	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.nameLink.Set(c.EveCharacter.ToEveEntity())
		a.portraitImage.OnTapped = func() {
			a.u.InfoViewer().Show(c.EveCharacter.ToEveEntity())
		}
		a.u.EVEImage().CharacterPortraitAsync(c.ID, 512, func(r fyne.Resource) {
			if !isLatest() {
				return
			}
			a.portraitImage.SetResource(r)
		})
		a.raceLink.SetRace(c.EveCharacter.Race)
		if v, ok := c.EveCharacter.Bloodline.Value(); ok {
			a.bloodlineLink.SetBloodline(v)
		} else {
			a.bloodlineLink.Clear()
		}
		if v, ok := c.EveCharacter.Faction.Value(); ok {
			a.factionLink.Set(v)
		} else {
			a.factionLink.Clear()
		}
		a.bornLabel.SetText(c.EveCharacter.Birthday.Format(app.DateTimeFormat))
		a.securityLabel.SetText(c.EveCharacter.SecurityStatus.StringFunc("?", func(v float64) string {
			return fmt.Sprintf("%.1f", v)
		}))
		a.lastLoginAtLabel.SetText(c.LastLoginAt.StringFunc("?", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))
		if home != nil {
			a.homeLink.SetLocation(home)
		} else {
			a.homeLink.Clear()
		}
		if location != nil {
			a.locationLink.SetLocation(location)
		} else {
			a.locationLink.Clear()
		}
		if shipType != nil {
			a.shipLink.Set(shipType.ToEveEntity())
		} else {
			a.shipLink.Clear()
		}
		a.skillpointsLabel.SetText(ihumanize.OptionalWithComma(c.TrainedSP, "?"))
		v := optional.SumNonEmpty(c.AssetValue, c.WalletBalance)
		a.wealthLabel.SetText(v.StringFunc("?", func(v float64) string {
			return humanize.Comma(int64(v))
		}) + " ISK")
	})

	var s string
	tags, err := a.u.Character().ListTagsForCharacter(ctx, c.ID)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("character sheet: update", "characterID", c.ID, "error", err)
		s = "?"
	} else {
		if tags.Size() == 0 {
			s = "-"
		} else {
			s = strings.Join(slices.Sorted(tags.All()), ", ")
		}
	}
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.tagsLabel.SetText(s)
	})
}

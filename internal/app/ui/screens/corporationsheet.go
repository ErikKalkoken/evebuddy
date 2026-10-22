package screens

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// CorporationSheet renders a corporation sheet.
type CorporationSheet struct {
	widget.BaseWidget

	allianceLink *ui.InfoLink
	ceoLink      *ui.InfoLink
	character    atomic.Pointer[app.Character]
	corporation  atomic.Pointer[app.Corporation]
	errorLabel   *widget.Label
	factionLink  *ui.InfoLink
	homeLink     *ui.InfoLink
	isCorpMode   bool
	logoImage    *xwidget.TappableImage
	membersLabel *widget.Label
	nameLink     *ui.InfoLink
	rolesLabel   *widget.Label
	taxRateLabel *widget.Label
	tickerLabel  *widget.Label
	u            baseUI
	updateRun    xsync.LatestRun
}

func NewCorporationSheet(u baseUI, isCorpMode bool) *CorporationSheet {
	logo := xwidget.NewTappableImage(icons.BlankSvg, nil)
	logo.SetFillMode(canvas.ImageFillContain)
	logo.SetMinSize(fyne.NewSquareSize(128))
	makeLabel := func() *widget.Label {
		x := xwidget.NewLabelWithSelection("?")
		x.Selectable = true
		x.Truncation = fyne.TextTruncateEllipsis
		return x
	}
	makeInfoLabel := func() *ui.InfoLink {
		return ui.NewInfoLink(u.InfoViewer())
	}
	a := &CorporationSheet{
		allianceLink: makeInfoLabel(),
		ceoLink:      makeInfoLabel(),
		errorLabel:   ui.NewLabelWithWrapping(""),
		factionLink:  makeInfoLabel(),
		homeLink:     makeInfoLabel(),
		isCorpMode:   isCorpMode,
		logoImage:    logo,
		membersLabel: makeLabel(),
		nameLink:     makeInfoLabel(),
		rolesLabel:   makeLabel(),
		taxRateLabel: makeLabel(),
		tickerLabel:  makeLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	a.errorLabel.Importance = widget.DangerImportance
	a.errorLabel.Hide()

	// signals
	if isCorpMode {
		a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			a.update(ctx)
		})
		a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
			corporationID := a.corporation.Load().IDOrZero()
			if corporationID == 0 {
				return
			}
			if arg.Section == app.SectionEveCorporations && arg.Changed.Contains(corporationID) {
				a.update(ctx)
			}
		})
	} else {
		a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
			a.character.Store(c)
			a.update(ctx)
		})
		a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
			if a.character.Load().IDOrZero() != arg.CharacterID {
				return
			}
			if arg.Section == app.SectionCharacterRoles {
				a.update(ctx)
			}
		})
		a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
			c := a.character.Load()
			if c == nil {
				return
			}
			switch arg.Section {
			case app.SectionEveCharacters:
				if arg.Changed.Contains(c.ID) {
					a.update(ctx) // character may have changed corporation
				}
			case app.SectionEveCorporations:
				if c.EveCharacter != nil && arg.Changed.Contains(c.EveCharacter.Corporation.ID) {
					a.update(ctx)
				}
			}
		})
	}
	return a
}

func (a *CorporationSheet) CreateRenderer() fyne.WidgetRenderer {
	items := []*widget.FormItem{
		widget.NewFormItem("Name", a.nameLink),
		widget.NewFormItem("Ticker", a.tickerLabel),
		widget.NewFormItem("Alliance", a.allianceLink),
		widget.NewFormItem("Faction", a.factionLink),
		widget.NewFormItem("CEO", a.ceoLink),
		widget.NewFormItem("Member Count", a.membersLabel),
		widget.NewFormItem("Home station", a.homeLink),
		widget.NewFormItem("Tax Rate", a.taxRateLabel),
	}
	if !a.isCorpMode {
		items = slices.Insert(items, 2, widget.NewFormItem("Roles", a.rolesLabel))
	}
	main := widget.NewForm(items...)
	main.Orientation = widget.Adaptive
	portraitDesktop := container.NewVBox(container.NewPadded(a.logoImage))
	c := container.NewBorder(
		nil,
		a.errorLabel,
		nil,
		portraitDesktop,
		main,
	)
	if a.u.IsMobile() {
		portraitDesktop.Hide()
	}
	return widget.NewSimpleRenderer(c)
}

func (a *CorporationSheet) update(ctx context.Context) {
	isLatest := a.updateRun.Start()
	clearAll := func() {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.errorLabel.Hide()
			a.allianceLink.Clear()
			a.ceoLink.Clear()
			a.factionLink.Clear()
			a.homeLink.Clear()
			a.logoImage.SetResource(icons.BlankSvg)
			a.membersLabel.SetText("")
			a.nameLink.Clear()
			a.rolesLabel.SetText("")
			a.taxRateLabel.SetText("")
			a.tickerLabel.SetText("")
		})
	}
	setError := func(err error) {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.errorLabel.SetText("ERROR: " + a.u.ErrorDisplay(err))
			a.errorLabel.Show()
			a.Refresh()
		})
	}
	var corporationID int64
	var roles string
	if a.isCorpMode {
		corporationID = a.corporation.Load().IDOrZero()
	} else {
		old := a.character.Load()
		if old == nil {
			clearAll()
			return
		}
		character, err := a.u.Character().GetCharacter(ctx, old.ID)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to fetch character for corporation sheet", "characterID", old.ID, "error", err)
			clearAll()
			setError(err)
			return
		}
		if !swapReloadedCharacter(&a.character, old, character) {
			return // character switched meanwhile; that switch runs its own update
		}
		if character.EveCharacter == nil {
			clearAll()
			return
		}
		corporationID = character.EveCharacter.Corporation.ID
		oo, err := a.u.Character().ListRoles(ctx, character.ID)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to fetch roles", "error", err)
			roles = "ERROR: " + a.u.ErrorDisplay(err)
		} else {
			x := slices.Sorted(xiter.Map(xiter.FilterSlice(oo, func(x app.CharacterRole) bool {
				return x.Granted
			}), func(x app.CharacterRole) string {
				return x.Role.Display()
			}))
			if len(x) == 0 {
				roles = "-"
			} else {
				roles = strings.Join(x, "\n")
			}
		}
	}
	if corporationID == 0 {
		clearAll()
		return
	}
	corporation, err := a.u.EVEUniverse().GetCorporation(ctx, corporationID)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			clearAll()
			return
		}
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to fetch eve corporation", "id", corporationID, "error", err)
		clearAll()
		setError(err)
		return
	}
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.errorLabel.Hide()
		a.nameLink.Set(corporation.ToEveEntity())
		a.logoImage.OnTapped = func() {
			a.u.InfoViewer().Show(corporation.ToEveEntity())
		}
		a.u.EVEImage().CorporationLogoAsync(corporation.ID, 512, func(r fyne.Resource) {
			if !isLatest() {
				return
			}
			a.logoImage.SetResource(r)
		})
		a.rolesLabel.SetText(roles)
		a.membersLabel.SetText(humanize.Comma(corporation.MemberCount))
		a.taxRateLabel.SetText(fmt.Sprintf("%.0f%%", corporation.TaxRate*100))
		a.tickerLabel.SetText(corporation.Ticker)
		if alliance, ok := corporation.Alliance.Value(); ok {
			a.allianceLink.Set(alliance)
		} else {
			a.allianceLink.Clear()
		}
		if faction, ok := corporation.Faction.Value(); ok {
			a.factionLink.Set(faction)
		} else {
			a.factionLink.Clear()
		}
		if ceo, ok := corporation.Ceo.Value(); ok {
			a.ceoLink.Set(ceo)
		} else {
			a.ceoLink.Clear()
		}
		if home, ok := corporation.HomeStation.Value(); ok {
			a.homeLink.Set(home)
		} else {
			a.homeLink.Clear()
		}
	})
}

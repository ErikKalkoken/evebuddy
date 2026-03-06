package corporationui

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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	services "github.com/ErikKalkoken/evebuddy/internal/app/services"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type CorporationSheet struct {
	widget.BaseWidget

	alliance    *widget.Hyperlink
	ceo         *widget.Hyperlink
	character   atomic.Pointer[app.Character]
	corporation atomic.Pointer[app.Corporation]
	faction     *widget.Hyperlink
	home        *widget.Hyperlink
	isCorpMode  bool
	logo        *xwidget.TappableImage
	members     *widget.Label
	name        *widget.Hyperlink
	roles       *widget.Label
	taxRate     *widget.Label
	ticker      *widget.Label
	s           services.Services
}

func NewCorporationSheet(s services.Services, isCorpMode bool) *CorporationSheet {
	if err := s.Check(); err != nil {
		panic(fmt.Errorf("corporation sheet: %w", err))
	}
	logo := xwidget.NewTappableImage(icons.BlankSvg, nil)
	logo.SetFillMode(canvas.ImageFillContain)
	logo.SetMinSize(fyne.NewSquareSize(128))
	makeHyperLink := func() *widget.Hyperlink {
		x := widget.NewHyperlink("?", nil)
		x.Truncation = fyne.TextTruncateEllipsis
		return x
	}
	makeLabel := func() *widget.Label {
		x := xwidget.NewLabelWithSelection("?")
		x.Selectable = true
		x.Truncation = fyne.TextTruncateEllipsis
		return x
	}
	a := &CorporationSheet{
		alliance:   makeHyperLink(),
		ceo:        makeHyperLink(),
		faction:    makeHyperLink(),
		home:       makeHyperLink(),
		isCorpMode: isCorpMode,
		logo:       logo,
		members:    makeLabel(),
		name:       makeHyperLink(),
		roles:      makeLabel(),
		taxRate:    makeLabel(),
		ticker:     makeLabel(),
		s:          s,
	}
	a.ExtendBaseWidget(a)

	// signals
	if isCorpMode {
		a.s.Signals.CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			a.update(ctx)
		})
		a.s.Signals.EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
			corporationID := corporationIDOrZero(a.corporation.Load())
			if corporationID == 0 {
				return
			}
			if arg.Section == app.SectionEveCorporations && arg.Changed.Contains(corporationID) {
				a.update(ctx)
			}
		})
	} else {
		a.s.Signals.CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
			a.character.Store(c)
			a.update(ctx)
		})
		a.s.Signals.EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
			c := a.character.Load()
			if c == nil {
				return
			}
			if arg.Section == app.SectionEveCorporations && arg.Changed.Contains(c.EveCharacter.Corporation.ID) {
				a.update(ctx)
			}
		})
	}
	return a
}

func (a *CorporationSheet) CreateRenderer() fyne.WidgetRenderer {
	items := []*widget.FormItem{
		widget.NewFormItem("Name", a.name),
		widget.NewFormItem("Ticker", a.ticker),
		widget.NewFormItem("Alliance", a.alliance),
		widget.NewFormItem("Faction", a.faction),
		widget.NewFormItem("CEO", a.ceo),
		widget.NewFormItem("Member Count", a.members),
		widget.NewFormItem("Home station", a.home),
		widget.NewFormItem("Tax Rate", a.taxRate),
	}
	if !a.isCorpMode {
		items = slices.Insert(items, 2, widget.NewFormItem("Roles", a.roles))
	}
	main := widget.NewForm(items...)
	main.Orientation = widget.Adaptive
	portraitDesktop := container.NewVBox(container.NewPadded(a.logo))
	c := container.NewBorder(
		nil,
		nil,
		nil,
		portraitDesktop,
		main,
	)
	if app.IsMobile() {
		portraitDesktop.Hide()
	}
	return widget.NewSimpleRenderer(c)
}

func (a *CorporationSheet) update(ctx context.Context) {
	var corporation *app.EveCorporation
	if a.isCorpMode {
		if c := a.corporation.Load(); c != nil {
			corporation = c.EveCorporation
		}
	} else {
		character := a.character.Load()
		if character != nil && character.EveCharacter != nil {
			var roles string
			oo, err := a.s.Character.ListRoles(ctx, character.ID)
			if err != nil {
				slog.Error("Failed to fetch roles", "error", err)
				roles = "ERROR: " + app.ErrorDisplay(err)
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
			fyne.Do(func() {
				a.roles.SetText(roles)
			})
			corporationID := character.EveCharacter.Corporation.ID
			c, err := a.s.EVEUniverse.GetCorporation(ctx, corporationID)
			if errors.Is(err, app.ErrNotFound) {
				// ignore
			} else if err != nil {
				slog.Error("Failed to fetch eve corporation", "id", corporationID, "error", err)
			} else {
				corporation = c
			}
		}
	}
	if corporation == nil {
		fyne.Do(func() {
			a.alliance.SetText("")
			a.ceo.SetText("")
			a.faction.SetText("")
			a.home.SetText("")
			a.logo.SetResource(icons.BlankSvg)
			a.members.SetText("")
			a.name.OnTapped = nil
			a.name.SetText("No corporation...")
			a.roles.SetText("")
			a.taxRate.SetText("")
			a.ticker.SetText("")
		})
		return
	}
	fyne.Do(func() {
		a.name.SetText(corporation.Name)
		a.name.OnTapped = func() {
			a.s.UI.InfoWindow().Show(app.EveEntityCorporation, corporation.ID)
		}
		a.logo.OnTapped = a.name.OnTapped
		a.members.SetText(humanize.Comma(corporation.MemberCount))
		a.taxRate.SetText(fmt.Sprintf("%.0f%%", corporation.TaxRate*100))
		a.ticker.SetText(corporation.Ticker)
		a.s.EVEImage.CorporationLogoAsync(corporation.ID, 512, func(r fyne.Resource) {
			a.logo.SetResource(r)
		})
	})
	fyne.Do(func() {
		if alliance, ok := corporation.Alliance.Value(); ok {
			a.alliance.SetText(alliance.Name)
			a.alliance.OnTapped = func() {
				a.s.UI.InfoWindow().ShowEntity(alliance)
			}
		} else {
			a.alliance.SetText("-")
			a.alliance.OnTapped = nil
		}
	})
	fyne.Do(func() {
		if faction, ok := corporation.Faction.Value(); ok {
			a.faction.SetText(faction.Name)
			a.faction.OnTapped = func() {
				a.s.UI.InfoWindow().ShowEntity(faction)
			}
		} else {
			a.faction.SetText("-")
			a.faction.OnTapped = nil
		}
	})
	fyne.Do(func() {
		if ceo, ok := corporation.Ceo.Value(); ok {
			a.ceo.SetText(ceo.Name)
			a.ceo.OnTapped = func() {
				a.s.UI.InfoWindow().ShowEntity(ceo)
			}
		} else {
			a.ceo.SetText("-")
			a.ceo.OnTapped = nil
		}
	})
	fyne.Do(func() {
		if home, ok := corporation.HomeStation.Value(); ok {
			a.home.SetText(home.Name)
			a.home.OnTapped = func() {
				a.s.UI.InfoWindow().ShowEntity(home)
			}
		} else {
			a.home.SetText("-")
			a.home.OnTapped = nil
		}
	})
}

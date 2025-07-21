package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

type corporationSheet struct {
	widget.BaseWidget

	alliance   *widget.Hyperlink
	ceo        *widget.Hyperlink
	faction    *widget.Hyperlink
	home       *widget.Hyperlink
	isCorpMode bool
	logo       *kxwidget.TappableImage
	members    *widget.Label
	name       *widget.Hyperlink
	roles      *widget.Label
	taxRate    *widget.Label
	ticker     *widget.Label
	u          *baseUI
}

func newCorporationSheet(u *baseUI, isCorpMode bool) *corporationSheet {
	roles := widget.NewLabel("")
	roles.Truncation = fyne.TextTruncateEllipsis
	logo := kxwidget.NewTappableImage(icons.BlankSvg, nil)
	logo.SetFillMode(canvas.ImageFillContain)
	logo.SetMinSize(fyne.NewSquareSize(128))
	w := &corporationSheet{
		alliance:   widget.NewHyperlink("", nil),
		ceo:        widget.NewHyperlink("", nil),
		faction:    widget.NewHyperlink("", nil),
		home:       widget.NewHyperlink("", nil),
		isCorpMode: isCorpMode,
		logo:       logo,
		members:    widget.NewLabel(""),
		name:       widget.NewHyperlink("", nil),
		roles:      roles,
		taxRate:    widget.NewLabel(""),
		ticker:     widget.NewLabel(""),
		u:          u,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (a *corporationSheet) CreateRenderer() fyne.WidgetRenderer {
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
	if !a.u.isDesktop {
		portraitDesktop.Hide()
	}
	return widget.NewSimpleRenderer(c)
}

func (a *corporationSheet) update() {
	var corporation *app.EveCorporation
	ctx := context.Background()
	if a.isCorpMode {
		x := a.u.currentCorporation()
		if x != nil {
			corporation = x.EveCorporation
		}
	} else {
		character := a.u.currentCharacter()
		if character != nil && character.EveCharacter != nil {
			var roles string
			oo, err := a.u.cs.ListRoles(ctx, character.ID)
			if err != nil {
				slog.Error("Failed to fetch roles", "error", err)
				roles = "?"
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
			c, err := a.u.eus.GetEveCorporation(ctx, corporationID)
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
			a.name.SetText("No corporation...")
			a.name.OnTapped = nil
		})
		return
	}
	fyne.Do(func() {
		a.name.SetText(corporation.Name)
		a.name.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityCorporation, corporation.ID)
		}
		a.logo.OnTapped = a.name.OnTapped
		a.members.SetText(humanize.Comma(corporation.MemberCount))
		a.taxRate.SetText(fmt.Sprintf("%.0f%%", corporation.TaxRate*100))
		a.ticker.SetText(corporation.Ticker)
	})
	iwidget.RefreshTappableImageAsync(a.logo, func() (fyne.Resource, error) {
		return a.u.eis.CorporationLogo(corporation.ID, 512)
	})
	fyne.Do(func() {
		if alliance := corporation.Alliance; alliance != nil {
			a.alliance.SetText(alliance.Name)
			a.alliance.OnTapped = func() {
				a.u.ShowEveEntityInfoWindow(alliance)
			}
		} else {
			a.alliance.SetText("-")
			a.alliance.OnTapped = nil
		}
	})
	fyne.Do(func() {
		if faction := corporation.Faction; faction != nil {
			a.faction.SetText(faction.Name)
			a.faction.OnTapped = func() {
				a.u.ShowEveEntityInfoWindow(faction)
			}
		} else {
			a.faction.SetText("-")
			a.faction.OnTapped = nil
		}
	})
	fyne.Do(func() {
		if ceo := corporation.Ceo; ceo != nil {
			a.ceo.SetText(ceo.Name)
			a.ceo.OnTapped = func() {
				a.u.ShowEveEntityInfoWindow(ceo)
			}
		} else {
			a.ceo.SetText("-")
			a.ceo.OnTapped = nil
		}
	})
	fyne.Do(func() {
		if home := corporation.HomeStation; home != nil {
			a.home.SetText(home.Name)
			a.home.OnTapped = func() {
				a.u.ShowEveEntityInfoWindow(home)
			}
		} else {
			a.home.SetText("-")
			a.home.OnTapped = nil
		}
	})
}

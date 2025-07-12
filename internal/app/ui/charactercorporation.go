package ui

import (
	"context"
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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

type characterCorporation struct {
	widget.BaseWidget

	alliance *widget.Hyperlink
	logo     *kxwidget.TappableImage
	name     *widget.Hyperlink
	roles    *widget.Label
	u        *baseUI
}

func newCharacterCorporation(u *baseUI) *characterCorporation {
	roles := widget.NewLabel("")
	roles.Truncation = fyne.TextTruncateEllipsis
	logo := kxwidget.NewTappableImage(icons.BlankSvg, nil)
	logo.SetFillMode(canvas.ImageFillContain)
	logo.SetMinSize(fyne.NewSquareSize(128))
	w := &characterCorporation{
		roles:    roles,
		name:     widget.NewHyperlink("", nil),
		logo:     logo,
		alliance: widget.NewHyperlink("", nil),
		u:        u,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (a *characterCorporation) update() {
	character := a.u.currentCharacter()
	if character == nil || character.EveCharacter == nil {
		fyne.Do(func() {
			a.name.SetText("No character...")
			a.name.OnTapped = nil
		})
		return
	}
	var roles string
	oo, err := a.u.cs.ListRoles(context.Background(), character.ID)
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
	corporation := character.EveCharacter.Corporation
	fyne.Do(func() {
		a.name.SetText(corporation.Name)
		a.name.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityCorporation, corporation.ID)
		}
		a.logo.OnTapped = a.name.OnTapped
		a.roles.SetText(roles)
	})
	fyne.Do(func() {
		if alliance := character.EveCharacter.Alliance; alliance != nil {
			a.alliance.SetText(alliance.Name)
			a.alliance.OnTapped = func() {
				a.u.ShowEveEntityInfoWindow(alliance)
			}
		} else {
			a.alliance.SetText("-")
			a.alliance.OnTapped = nil
		}
	})
	iwidget.RefreshTappableImageAsync(a.logo, func() (fyne.Resource, error) {
		return a.u.eis.CorporationLogo(corporation.ID, 512)
	})
}

func (a *characterCorporation) CreateRenderer() fyne.WidgetRenderer {
	const width = 140
	main := widget.NewForm(
		widget.NewFormItem("Name", a.name),
		widget.NewFormItem("Roles", a.roles),
		widget.NewFormItem("Alliance", a.alliance),
	)

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

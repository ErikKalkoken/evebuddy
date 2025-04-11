package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type overviewCharacter struct {
	alliance      *app.EveEntity
	assetValue    optional.Optional[float64]
	birthday      time.Time
	corporation   *app.EveEntity
	home          *app.EveLocation
	id            int32
	lastLoginAt   optional.Optional[time.Time]
	name          string
	security      float64
	unreadCount   optional.Optional[int]
	walletBalance optional.Optional[float64]
}

type OverviewCharacters struct {
	widget.BaseWidget

	rows []overviewCharacter
	body fyne.CanvasObject
	top  *widget.Label
	u    *BaseUI
}

func NewOverviewCharacters(u *BaseUI) *OverviewCharacters {
	a := &OverviewCharacters{
		rows: make([]overviewCharacter, 0),
		top:  appwidget.MakeTopLabel(),
		u:    u,
	}
	a.ExtendBaseWidget(a)
	headers := []iwidget.HeaderDef{
		{Text: "Character", Width: columnWidthCharacter},
		{Text: "Corporation", Width: 250},
		{Text: "Alliance", Width: 250},
		{Text: "Sec.", Width: 50},
		{Text: "Unread", Width: 100},
		{Text: "Wallet", Width: 100},
		{Text: "Assets", Width: 100},
		{Text: "Last Login", Width: 100},
		{Text: "Home", Width: columnWidthLocation},
		{Text: "Age", Width: 100},
	}
	makeCell := func(col int, c overviewCharacter) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(c.name)
		case 1:
			return iwidget.NewRichTextSegmentFromText(c.corporation.Name)
		case 2:
			var s string
			if c.alliance != nil {
				s = c.alliance.Name
			}
			return iwidget.NewRichTextSegmentFromText(s)
		case 3:
			var color fyne.ThemeColorName
			text := fmt.Sprintf("%.1f", c.security)
			if c.security > 0 {
				color = theme.ColorNameSuccess
			} else if c.security < 0 {
				color = theme.ColorNameError
			} else {
				color = theme.ColorNameForeground
			}
			return iwidget.NewRichTextSegmentFromText(text, widget.RichTextStyle{
				ColorName: color,
			})
		case 4:
			return iwidget.NewRichTextSegmentFromText(ihumanize.Optional(c.unreadCount, "?"))
		case 5:
			return iwidget.NewRichTextSegmentFromText(ihumanize.OptionalFloat(c.walletBalance, 1, "?"))
		case 6:
			return iwidget.NewRichTextSegmentFromText(ihumanize.OptionalFloat(c.assetValue, 1, "?"))
		case 7:
			return iwidget.NewRichTextSegmentFromText(ihumanize.Optional(c.lastLoginAt, "?"))
		case 8:
			if c.home != nil {
				return c.home.DisplayRichText()
			}
		case 9:
			return iwidget.NewRichTextSegmentFromText(humanize.RelTime(c.birthday, time.Now(), "", ""))
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop(headers, &a.rows, makeCell, func(c int, oc overviewCharacter) {
			switch c {
			case 0:
				u.ShowInfoWindow(app.EveEntityCharacter, oc.id)
			case 1:
				u.ShowInfoWindow(app.EveEntityCorporation, oc.corporation.ID)
			case 2:
				if oc.alliance != nil {
					u.ShowInfoWindow(app.EveEntityAlliance, oc.alliance.ID)
				}
			case 8:
				if oc.home != nil {
					u.ShowLocationInfoWindow(oc.home.ID)
				}
			}
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile(headers, &a.rows, makeCell, func(oc overviewCharacter) {
			u.ShowInfoWindow(app.EveEntityCharacter, oc.id)
		})
	}
	return a
}

func (a *OverviewCharacters) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *OverviewCharacters) Update() {
	t, i, err := func() (string, widget.Importance, error) {
		totals, err := a.updateCharacters()
		if err != nil {
			return "", 0, err
		}
		if len(a.rows) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		walletText := ihumanize.OptionalFloat(totals.wallet, 1, "?")
		assetsText := ihumanize.OptionalFloat(totals.assets, 1, "?")
		unreadText := ihumanize.Optional(totals.unread, "?")
		s := fmt.Sprintf(
			"%d characters • %s ISK wallet • %s ISK assets • %s unread",
			len(a.rows),
			walletText,
			assetsText,
			unreadText,
		)
		return s, widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh overview UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.body.Refresh()
}

type overviewTotals struct {
	unread optional.Optional[int]
	wallet optional.Optional[float64]
	assets optional.Optional[float64]
}

func (a *OverviewCharacters) updateCharacters() (overviewTotals, error) {
	var totals overviewTotals
	var err error
	ctx := context.Background()
	characters, err := a.u.CharacterService().ListCharacters(ctx)
	if err != nil {
		return totals, err
	}
	cc := xslices.Map(characters, func(m *app.Character) overviewCharacter {
		return overviewCharacter{
			alliance:      m.EveCharacter.Alliance,
			birthday:      m.EveCharacter.Birthday,
			corporation:   m.EveCharacter.Corporation,
			lastLoginAt:   m.LastLoginAt,
			id:            m.ID,
			name:          m.EveCharacter.Name,
			security:      m.EveCharacter.SecurityStatus,
			walletBalance: m.WalletBalance,
			home:          m.Home,
		}
	})
	for i, c := range cc {
		total, unread, err := a.u.CharacterService().GetMailCounts(ctx, c.id)
		if err != nil {
			return totals, err
		}
		if total > 0 {
			cc[i].unreadCount = optional.New(unread)
		}
	}
	for i, c := range cc {
		v, err := a.u.CharacterService().AssetTotalValue(ctx, c.id)
		if err != nil {
			return totals, err
		}
		cc[i].assetValue = v
	}
	for _, c := range cc {
		if !c.unreadCount.IsEmpty() {
			totals.unread.Set(totals.unread.ValueOrZero() + c.unreadCount.ValueOrZero())
		}
		if !c.walletBalance.IsEmpty() {
			totals.wallet.Set(totals.wallet.ValueOrZero() + c.walletBalance.ValueOrZero())
		}
		if !c.assetValue.IsEmpty() {
			totals.assets.Set(totals.assets.ValueOrZero() + c.assetValue.ValueOrZero())
		}
	}
	a.rows = cc
	return totals, nil
}

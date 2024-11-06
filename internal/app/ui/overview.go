package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/app/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type overviewCharacter struct {
	alliance       string
	assetValue     optional.Optional[float64]
	birthday       time.Time
	corporation    string
	home           *app.EntityShort[int64]
	id             int32
	lastLoginAt    optional.Optional[time.Time]
	location       *app.EntityShort[int64]
	name           string
	region         *app.EntityShort[int32]
	security       float64
	ship           *app.EntityShort[int32]
	solarSystem    *app.EntityShort[int32]
	systemSecurity optional.Optional[float32]
	totalSP        optional.Optional[int]
	training       optional.Optional[time.Duration]
	unallocatedSP  optional.Optional[int]
	unreadCount    optional.Optional[int]
	walletBalance  optional.Optional[float64]
}

// overviewArea is the UI area that shows an overview of all the user's characters.
type overviewArea struct {
	characters []overviewCharacter
	content    *fyne.Container
	table      *widget.Table
	top        *widget.Label
	u          *UI
}

func (u *UI) newOverviewArea() *overviewArea {
	a := overviewArea{
		characters: make([]overviewCharacter, 0),
		top:        widget.NewLabel(""),
		u:          u,
	}
	a.top.TextStyle.Bold = true

	top := container.NewVBox(a.top, widget.NewSeparator())
	a.table = a.makeTable()
	a.content = container.NewBorder(top, nil, nil, nil, a.table)
	return &a
}

func (a *overviewArea) makeTable() *widget.Table {
	var headers = []struct {
		text     string
		maxChars int
	}{
		{"Name", 20},
		{"Corporation", 20},
		{"Alliance", 20},
		{"Security", 10},
		{"Unread", 5},
		{"Total SP", 5},
		{"Unall. SP", 5},
		{"Training", 5},
		{"Wallet", 5},
		{"Assets", 5},
		{"Location", 15},
		{"System", 15},
		{"Region", 15},
		{"Ship", 15},
		{"Last Login", 10},
		{"Home", 15},
		{"Age", 10},
	}

	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.characters), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Template Template")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			if tci.Row >= len(a.characters) {
				return
			}
			c := a.characters[tci.Row]
			l.Importance = widget.MediumImportance
			var text string
			switch tci.Col {
			case 0:
				text = c.name
			case 1:
				text = c.corporation
			case 2:
				text = c.alliance
			case 3:
				text = fmt.Sprintf("%.1f", c.security)
				if c.security > 0 {
					l.Importance = widget.SuccessImportance
				} else if c.security < 0 {
					l.Importance = widget.DangerImportance
				}
			case 4:
				text = ihumanize.Optional(c.unreadCount, "?")
			case 5:
				text = ihumanize.Optional(c.totalSP, "?")
			case 6:
				text = ihumanize.Optional(c.unallocatedSP, "?")
			case 7:
				if c.training.IsEmpty() {
					text = "Inactive"
					l.Importance = widget.WarningImportance
				} else {
					text = ihumanize.Duration(c.training.MustValue())
				}
			case 8:
				text = ihumanize.OptionalFloat(c.walletBalance, 1, "?")
			case 9:
				text = ihumanize.OptionalFloat(c.assetValue, 1, "?")
			case 10:
				text = entityNameOrFallback(c.location, "?")
			case 11:
				if c.solarSystem == nil || c.systemSecurity.IsEmpty() {
					text = "?"
				} else {
					text = fmt.Sprintf("%s %.1f", c.solarSystem.Name, c.systemSecurity.MustValue())
				}
			case 12:
				text = entityNameOrFallback(c.region, "?")
			case 13:
				text = entityNameOrFallback(c.ship, "?")
			case 14:
				text = ihumanize.Optional(c.lastLoginAt, "?")
			case 15:
				text = entityNameOrFallback(c.home, "?")
			case 16:
				text = humanize.RelTime(c.birthday, time.Now(), "", "")
			}
			l.Text = text
			l.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.StickyColumnCount = 1
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		label := co.(*widget.Label)
		label.SetText(s.text)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
	}

	for i, h := range headers {
		x := widget.NewLabel(strings.Repeat("w", h.maxChars))
		w := x.MinSize().Width
		t.SetColumnWidth(i, w)
	}
	return t
}

func (u *UI) selectCharacterAndTab(characterID int32, tab *container.TabItem, subIndex int) {
	if err := u.loadCharacter(context.TODO(), characterID); err != nil {
		panic(err)
	}
	u.tabs.Select(tab)
	t := tab.Content.(*container.AppTabs)
	t.SelectIndex(subIndex)
}

func (a *overviewArea) refresh() {
	t, i, err := func() (string, widget.Importance, error) {
		totals, err := a.updateCharacters()
		if err != nil {
			return "", 0, err
		}
		if len(a.characters) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		walletText := ihumanize.OptionalFloat(totals.wallet, 1, "?")
		assetsText := ihumanize.OptionalFloat(totals.assets, 1, "?")
		spText := ihumanize.Optional(totals.sp, "?")
		unreadText := ihumanize.Optional(totals.unread, "?")
		s := fmt.Sprintf(
			"Total: %d characters • %s ISK wallet • %s ISK assets • %s SP  • %s unread",
			len(a.characters),
			walletText,
			assetsText,
			spText,
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
	a.table.Refresh()
}

type overviewTotals struct {
	sp     optional.Optional[int]
	unread optional.Optional[int]
	wallet optional.Optional[float64]
	assets optional.Optional[float64]
}

func (a *overviewArea) updateCharacters() (overviewTotals, error) {
	var totals overviewTotals
	var err error
	ctx := context.TODO()
	mycc, err := a.u.CharacterService.ListCharacters(ctx)
	if err != nil {
		return totals, fmt.Errorf("fetch characters: %w", err)
	}
	cc := make([]overviewCharacter, len(mycc))
	for i, m := range mycc {
		c := overviewCharacter{
			alliance:      m.EveCharacter.AllianceName(),
			birthday:      m.EveCharacter.Birthday,
			corporation:   m.EveCharacter.Corporation.Name,
			lastLoginAt:   m.LastLoginAt,
			id:            m.ID,
			name:          m.EveCharacter.Name,
			security:      m.EveCharacter.SecurityStatus,
			totalSP:       m.TotalSP,
			unallocatedSP: m.UnallocatedSP,
			walletBalance: m.WalletBalance,
		}
		if m.Home != nil {
			c.home = &app.EntityShort[int64]{
				ID:   m.Home.ID,
				Name: m.Home.DisplayName(),
			}
		}
		if m.Location != nil {
			c.location = &app.EntityShort[int64]{
				ID:   m.Location.ID,
				Name: m.Location.DisplayName(),
			}
			if m.Location.SolarSystem != nil {
				c.solarSystem = &app.EntityShort[int32]{
					ID:   m.Location.SolarSystem.ID,
					Name: m.Location.SolarSystem.Name,
				}
				c.systemSecurity = optional.New(m.Location.SolarSystem.SecurityStatus)
				c.region = &app.EntityShort[int32]{
					ID:   m.Location.SolarSystem.Constellation.Region.ID,
					Name: m.Location.SolarSystem.Constellation.Region.Name,
				}
			}
		}
		if m.Ship != nil {
			c.ship = &app.EntityShort[int32]{
				ID:   m.Ship.ID,
				Name: m.Ship.Name,
			}
		}
		cc[i] = c
	}
	for i, c := range cc {
		v, err := a.u.CharacterService.GetCharacterTotalTrainingTime(ctx, c.id)
		if err != nil {
			return totals, fmt.Errorf("fetch skill queue count for character %d, %w", c.id, err)
		}
		cc[i].training = v
	}
	for i, c := range cc {
		total, unread, err := a.u.CharacterService.GetCharacterMailCounts(ctx, c.id)
		if err != nil {
			return totals, fmt.Errorf("fetch mail counts for character %d, %w", c.id, err)
		}
		if total > 0 {
			cc[i].unreadCount = optional.New(unread)
		}
	}
	for i, c := range cc {
		v, err := a.u.CharacterService.CharacterAssetTotalValue(ctx, c.id)
		if err != nil {
			return totals, fmt.Errorf("fetch asset total value for character %d, %w", c.id, err)
		}
		cc[i].assetValue = v
	}
	for _, c := range cc {
		if !c.totalSP.IsEmpty() {
			totals.sp.Set(totals.sp.ValueOrZero() + c.totalSP.MustValue())
		}
		if !c.unreadCount.IsEmpty() {
			totals.unread.Set(totals.unread.ValueOrZero() + c.unreadCount.MustValue())
		}
		if !c.walletBalance.IsEmpty() {
			totals.wallet.Set(totals.wallet.ValueOrZero() + c.walletBalance.MustValue())
		}
		if !c.assetValue.IsEmpty() {
			totals.assets.Set(totals.assets.ValueOrZero() + c.assetValue.MustValue())
		}
	}
	a.characters = cc
	var hasUnread bool
	for _, c := range a.characters {
		if c.unreadCount.ValueOrZero() > 0 {
			hasUnread = true
			break
		}
	}
	if hasUnread {
		a.u.showMailIndicator()
	}
	return totals, nil
}

package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/app/humanize"
	"github.com/ErikKalkoken/evebuddy/pkg/optional"
)

type overviewCharacter struct {
	alliance       string
	assetValue     optional.Optional[float64]
	birthday       time.Time
	corporation    string
	id             int32
	home           *app.EntityShort[int64]
	lastLoginAt    optional.Optional[time.Time]
	location       *app.EntityShort[int64]
	name           string
	solarSystem    *app.EntityShort[int32]
	systemSecurity optional.Optional[float32]
	region         *app.EntityShort[int32]
	ship           *app.EntityShort[int32]
	security       float64
	totalSP        optional.Optional[int]
	training       optional.Optional[time.Duration]
	unallocatedSP  optional.Optional[int]
	unreadCount    optional.Optional[int]
	walletBalance  optional.Optional[float64]
}

// overviewArea is the UI area that shows an overview of all the user's characters.
type overviewArea struct {
	content    *fyne.Container
	characters []overviewCharacter
	table      *widget.Table
	top        *widget.Label
	ui         *ui
}

func (u *ui) newOverviewArea() *overviewArea {
	a := overviewArea{
		characters: make([]overviewCharacter, 0),
		top:        widget.NewLabel(""),
		ui:         u,
	}
	a.top.TextStyle.Bold = true

	top := container.NewVBox(a.top, widget.NewSeparator())
	a.table = a.makeTable()
	a.content = container.NewBorder(top, nil, nil, nil, a.table)
	return &a
}

func (a *overviewArea) makeTable() *widget.Table {
	var headers = []struct {
		text   string
		width  float32
		action func(overviewCharacter)
	}{
		{"Name", 200, nil},
		{"Corporation", 200, nil},
		{"Alliance", 200, nil},
		{"Security", 80, nil},
		{"Unread", 80, func(oc overviewCharacter) {
			a.ui.selectCharacterAndTab(oc.id, a.ui.mailTab, 0)
		}},
		{"Total SP", 80, func(oc overviewCharacter) {
			a.ui.selectCharacterAndTab(oc.id, a.ui.skillTab, 1)
		}},
		{"Unall. SP", 80, func(oc overviewCharacter) {
			a.ui.selectCharacterAndTab(oc.id, a.ui.skillTab, 1)
		}},
		{"Training", 80, func(oc overviewCharacter) {
			a.ui.selectCharacterAndTab(oc.id, a.ui.skillTab, 0)
		}},
		{"Wallet", 80, func(oc overviewCharacter) {
			a.ui.selectCharacterAndTab(oc.id, a.ui.walletTab, 0)
		}},
		{"Assets", 80, func(oc overviewCharacter) {
			a.ui.selectCharacterAndTab(oc.id, a.ui.assetTab, 0)
		}},
		{"Location", 150, func(oc overviewCharacter) {
			if oc.location != nil {
				a.ui.showLocationInfoWindow(oc.location.ID)
			}
		}},
		{"System", 150, nil},
		{"Region", 150, nil},
		{"Ship", 150, func(oc overviewCharacter) {
			if oc.ship != nil {
				a.ui.showTypeInfoWindow(oc.ship.ID, a.ui.characterID())
			}
		}},
		{"Last Login", 100, nil},
		{"Home", 150, func(oc overviewCharacter) {
			if oc.home != nil {
				a.ui.showLocationInfoWindow(oc.home.ID)
			}
		}},
		{"Age", 100, nil},
	}

	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.characters), len(headers)
		},
		func() fyne.CanvasObject {
			x := widget.NewLabel("Template")
			x.Truncation = fyne.TextTruncateEllipsis
			return x
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			if tci.Row >= len(a.characters) {
				return
			}
			c := a.characters[tci.Row]
			l.Importance = widget.MediumImportance
			switch tci.Col {
			case 0:
				l.Text = c.name
			case 1:
				l.Text = c.corporation
			case 2:
				l.Text = c.alliance
			case 3:
				l.Text = fmt.Sprintf("%.1f", c.security)
				if c.security > 0 {
					l.Importance = widget.SuccessImportance
				} else if c.security < 0 {
					l.Importance = widget.DangerImportance
				}
			case 4:
				l.Text = ihumanize.Optional(c.unreadCount, "?")
			case 5:
				l.Text = ihumanize.Optional(c.totalSP, "?")
			case 6:
				l.Text = ihumanize.Optional(c.unallocatedSP, "?")
			case 7:
				if c.training.IsEmpty() {
					l.Text = "Inactive"
					l.Importance = widget.WarningImportance
				} else {
					l.Text = ihumanize.Duration(c.training.MustValue())
				}
			case 8:
				l.Text = ihumanize.OptionalFloat(c.walletBalance, 1, "?")
			case 9:
				l.Text = ihumanize.OptionalFloat(c.assetValue, 1, "?")
			case 10:
				l.Text = entityNameOrFallback(c.location, "?")
			case 11:
				if c.solarSystem == nil || c.systemSecurity.IsEmpty() {
					l.Text = "?"
				} else {
					l.Text = fmt.Sprintf("%s %.1f", c.solarSystem.Name, c.systemSecurity.MustValue())
				}
			case 12:
				l.Text = entityNameOrFallback(c.region, "?")
			case 13:
				l.Text = entityNameOrFallback(c.ship, "?")
			case 14:
				l.Text = ihumanize.Optional(c.lastLoginAt, "?")
			case 15:
				l.Text = entityNameOrFallback(c.home, "?")
			case 16:
				l.Text = humanize.RelTime(c.birthday, time.Now(), "", "")
			}
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
		label.Text = s.text
		if headers[tci.Col].action != nil {
			label.Importance = widget.HighImportance
		} else {
			label.Importance = widget.MediumImportance
		}
		label.Refresh()
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
		if tci.Row <= len(a.characters) {
			return
		}
		c := a.characters[tci.Row]
		if action := headers[tci.Col].action; action != nil {
			action(c)
		}
	}

	for i, h := range headers {
		t.SetColumnWidth(i, h.width)
	}
	return t
}

func (u *ui) selectCharacterAndTab(characterID int32, tab *container.TabItem, subIndex int) {
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
	mycc, err := a.ui.CharacterService.ListCharacters(ctx)
	if err != nil {
		return totals, fmt.Errorf("failed to fetch characters: %w", err)
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
			c.region = &app.EntityShort[int32]{
				ID:   m.Location.SolarSystem.Constellation.Region.ID,
				Name: m.Location.SolarSystem.Constellation.Region.Name,
			}
			c.location = &app.EntityShort[int64]{
				ID:   m.Location.ID,
				Name: m.Location.DisplayName(),
			}
			c.solarSystem = &app.EntityShort[int32]{
				ID:   m.Location.SolarSystem.ID,
				Name: m.Location.SolarSystem.Name,
			}
			c.systemSecurity = optional.New(m.Location.SolarSystem.SecurityStatus)
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
		v, err := a.ui.CharacterService.GetCharacterTotalTrainingTime(ctx, c.id)
		if err != nil {
			return totals, fmt.Errorf("failed to fetch skill queue count for character %d, %w", c.id, err)
		}
		cc[i].training = v
	}
	for i, c := range cc {
		total, unread, err := a.ui.CharacterService.GetCharacterMailCounts(ctx, c.id)
		if err != nil {
			return totals, fmt.Errorf("failed to fetch mail counts for character %d, %w", c.id, err)
		}
		if total > 0 {
			cc[i].unreadCount = optional.New(unread)
		}
	}
	for i, c := range cc {
		v, err := a.ui.CharacterService.CharacterAssetTotalValue(ctx, c.id)
		if err != nil {
			return totals, fmt.Errorf("failed to fetch asset total value for character %d, %w", c.id, err)
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
		a.ui.showMailIndicator()
	}
	return totals, nil
}

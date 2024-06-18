package ui

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
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
	totalSP        optional.Optional[int64]
	training       optional.Optional[time.Duration]
	unallocatedSP  optional.Optional[int64]
	unreadCount    optional.Optional[int64]
	walletBalance  optional.Optional[float64]
}

// overviewArea is the UI area that shows an overview of all the user's characters.
type overviewArea struct {
	content    *fyne.Container
	characters binding.UntypedList // []overviewCharacter
	table      *widget.Table
	top        *widget.Label
	ui         *ui
}

func (u *ui) newOverviewArea() *overviewArea {
	a := overviewArea{
		characters: binding.NewUntypedList(),
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
		text  string
		width float32
	}{
		{"Name", 200},
		{"Corporation", 200},
		{"Alliance", 200},
		{"Security", 80},
		{"Unread", 80},
		{"Total SP", 80},
		{"Unall. SP", 80},
		{"Training", 80},
		{"Wallet", 80},
		{"Assets", 80},
		{"Location", 150},
		{"System", 150},
		{"Region", 150},
		{"Ship", 150},
		{"Last Login", 100},
		{"Home", 150},
		{"Age", 100},
	}

	t := widget.NewTable(
		func() (rows int, cols int) {
			return a.characters.Length(), len(headers)
		},
		func() fyne.CanvasObject {
			x := widget.NewLabel("Template")
			x.Truncation = fyne.TextTruncateEllipsis
			return x
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			c, err := getItemUntypedList[overviewCharacter](a.characters, tci.Row)
			if err != nil {
				slog.Error("failed to render cell in overview table", "err", err)
				l.Text = "failed to render"
				l.Importance = widget.DangerImportance
				l.Refresh()
				return
			}
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
				l.Text = humanizedNumericOption(c.unreadCount, 0, "?")
			case 5:
				l.Text = humanizedNumericOption(c.totalSP, 0, "?")
			case 6:
				l.Text = humanizedNumericOption(c.unallocatedSP, 0, "?")
			case 7:
				if c.training.IsNone() {
					l.Text = "Inactive"
					l.Importance = widget.WarningImportance
				} else {
					l.Text = ihumanize.Duration(c.training.MustValue())
				}
			case 8:
				l.Text = humanizedNumericMaybe(c.walletBalance, 1, "?")
			case 9:
				l.Text = humanizedNumericOption(c.assetValue, 1, "?")
			case 10:
				l.Text = entityNameOrFallback(c.location, "?")
			case 11:
				if c.solarSystem == nil || c.systemSecurity.IsNone() {
					l.Text = "?"
				} else {
					l.Text = fmt.Sprintf("%s %.1f", c.solarSystem.Name, c.systemSecurity.MustValue())
				}
			case 12:
				l.Text = entityNameOrFallback(c.region, "?")
			case 13:
				l.Text = entityNameOrFallback(c.ship, "?")
			case 14:
				l.Text = humanizedRelOptionTime(c.lastLoginAt, "?")
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
		co.(*widget.Label).SetText(s.text)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
		ctx := context.Background()
		c, err := getItemUntypedList[overviewCharacter](a.characters, tci.Row)
		if err != nil {
			slog.Error("Failed to select character", "err", err)
			return
		}
		m := map[int]struct {
			parent, child int
		}{
			4: {2, 0},
			5: {3, 1},
			6: {3, 1},
			7: {3, 0},
			8: {4, 0},
		}
		idx, ok := m[tci.Col]
		if ok {
			if err := a.ui.loadCharacter(ctx, c.id); err != nil {
				panic(err)
			}
			a.ui.tabs.SelectIndex(idx.parent)
			t := a.ui.tabs.Items[idx.parent].Content.(*container.AppTabs)
			t.SelectIndex(idx.child)
		}
		if tci.Col == 9 {
			if c.location != nil {
				a.ui.showLocationInfoWindow(c.location.ID)
			}
		}
		if tci.Col == 13 {
			if c.ship != nil {
				a.ui.showTypeInfoWindow(c.ship.ID, a.ui.characterID())
			}
		}
		if tci.Col == 16 {
			if c.home != nil {
				a.ui.showLocationInfoWindow(c.home.ID)
			}
		}
	}

	for i, h := range headers {
		t.SetColumnWidth(i, h.width)
	}
	return t
}

func (a *overviewArea) refresh() {
	t, i, err := func() (string, widget.Importance, error) {
		totals, err := a.updateEntries()
		if err != nil {
			return "", 0, err
		}
		if a.characters.Length() == 0 {
			return "No characters", widget.LowImportance, nil
		}
		walletText := humanizedNumericOption(totals.wallet, 1, "?")
		assetsText := humanizedNumericOption(totals.assets, 1, "?")
		spText := humanizedNumericOption(totals.sp, 0, "?")
		unreadText := humanizedNumericOption(totals.unread, 0, "?")
		s := fmt.Sprintf(
			"Total: %d characters • %s ISK wallet • %s ISK assets • %s SP  • %s unread",
			a.characters.Length(),
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
	sp     optional.Optional[int64]
	unread optional.Optional[int64]
	wallet optional.Optional[float64]
	assets optional.Optional[float64]
}

func (a *overviewArea) updateEntries() (overviewTotals, error) {
	var totals overviewTotals
	var err error
	ctx := context.Background()
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
			lastLoginAt:   optionFromNullTime(m.LastLoginAt),
			id:            m.ID,
			name:          m.EveCharacter.Name,
			security:      m.EveCharacter.SecurityStatus,
			totalSP:       optionFromNullInt64(m.TotalSP),
			unallocatedSP: optionFromNullInt64(m.UnallocatedSP),
			walletBalance: maybeFromNullFloat64(m.WalletBalance),
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
			cc[i].unreadCount = optional.New(int64(unread))
		}
	}
	for i, c := range cc {
		v, err := a.ui.CharacterService.CharacterAssetTotalValue(c.id)
		if err != nil {
			return totals, fmt.Errorf("failed to fetch asset total value for character %d, %w", c.id, err)
		}
		cc[i].assetValue = optionFromNullFloat64(v)
	}
	if err := a.characters.Set(copyToUntypedSlice(cc)); err != nil {
		return totals, err
	}
	for _, c := range cc {
		if c.totalSP.IsValue() {
			totals.sp.Set(totals.sp.ValueOrFallback(0) + c.totalSP.MustValue())
		}
		if c.unreadCount.IsValue() {
			totals.unread.Set(totals.unread.ValueOrFallback(0) + c.unreadCount.MustValue())
		}
		if c.walletBalance.IsValue() {
			totals.wallet.Set(totals.wallet.ValueOrFallback(0) + c.walletBalance.MustValue())
		}
		if c.assetValue.IsValue() {
			totals.assets.Set(totals.assets.ValueOrFallback(0) + c.assetValue.MustValue())
		}
	}
	return totals, nil
}

func maybeFromNullFloat64(v sql.NullFloat64) optional.Optional[float64] {
	if !v.Valid {
		return optional.NewNone[float64]()
	}
	return optional.New(v.Float64)
}

func optionFromNullFloat64(v sql.NullFloat64) optional.Optional[float64] {
	if !v.Valid {
		return optional.NewNone[float64]()
	}
	return optional.New(v.Float64)
}

func optionFromNullInt64(v sql.NullInt64) optional.Optional[int64] {
	if !v.Valid {
		return optional.NewNone[int64]()
	}
	return optional.New(v.Int64)
}

func optionFromNullTime(v sql.NullTime) optional.Optional[time.Time] {
	if !v.Valid {
		return optional.NewNone[time.Time]()
	}
	return optional.New(v.Time)
}

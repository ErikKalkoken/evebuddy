package ui

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

const overviewUpdateTicker = 60 * time.Second

type overviewCharacter struct {
	alliance       string
	birthday       time.Time
	corporation    string
	id             int32
	home           sql.NullString
	lastLoginAt    sql.NullTime
	location       sql.NullString
	name           string
	systemName     sql.NullString
	systemSecurity sql.NullFloat64
	region         sql.NullString
	ship           sql.NullString
	security       float64
	totalSP        sql.NullInt64
	training       types.NullDuration
	unallocatedSP  sql.NullInt64
	unreadCount    sql.NullInt64
	walletBalance  sql.NullFloat64
}

// overviewArea is the UI area that shows an overview of all the user's characters.
type overviewArea struct {
	content    *fyne.Container
	characters binding.UntypedList // []overviewCharacter
	table      *widget.Table
	totalLabel *widget.Label
	ui         *ui
}

func (u *ui) NewOverviewArea() *overviewArea {
	a := overviewArea{
		characters: binding.NewUntypedList(),
		totalLabel: widget.NewLabel(""),
		ui:         u,
	}
	a.totalLabel.TextStyle.Bold = true
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
			c, err := getFromBoundUntypedList[overviewCharacter](a.characters, tci.Row)
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
				l.Text = humanizedNullInt64(c.unreadCount, "?")
			case 5:
				l.Text = humanizedNullInt64(c.totalSP, "?")
			case 6:
				l.Text = humanizedNullInt64(c.unallocatedSP, "?")
			case 7:
				if !c.training.Valid {
					l.Text = "Inactive"
					l.Importance = widget.WarningImportance
				} else {
					l.Text = ihumanize.Duration(c.training.Duration)
				}
			case 8:
				l.Text = humanizedNullFloat64(c.walletBalance, 1, "?")
			case 9:
				l.Text = nullStringOrFallback(c.location, "?")
			case 10:
				if !c.systemName.Valid || !c.systemSecurity.Valid {
					l.Text = "?"
				} else {
					l.Text = fmt.Sprintf("%s %.1f", c.systemName.String, c.systemSecurity.Float64)
				}
			case 11:
				l.Text = nullStringOrFallback(c.region, "?")
			case 12:
				l.Text = nullStringOrFallback(c.ship, "?")
			case 13:
				l.Text = humanizedRelNullTime(c.lastLoginAt, "?")
			case 14:
				l.Text = nullStringOrFallback(c.home, "?")
			case 15:
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
		c, err := getFromBoundUntypedList[overviewCharacter](a.characters, tci.Row)
		if err != nil {
			panic(err)
		}
		switch tci.Col {
		case 4:
			a.ui.LoadCurrentCharacter(c.id)
			a.ui.tabs.SelectIndex(0)
		case 6:
			a.ui.LoadCurrentCharacter(c.id)
			a.ui.tabs.SelectIndex(1)
		case 7:
			a.ui.LoadCurrentCharacter(c.id)
			a.ui.tabs.SelectIndex(2)
		}
	}

	for i, h := range headers {
		t.SetColumnWidth(i, h.width)
	}

	top := container.NewVBox(a.totalLabel, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, t)
	a.table = t
	a.characters.AddListener(binding.NewDataListener(func() {
		a.table.Refresh()
	}))
	return &a
}

func (a *overviewArea) Refresh() {
	sp, unread, wallet, err := a.updateEntries()
	if err != nil {
		slog.Error("Failed to refresh overview", "err", err)
		return
	}
	walletText := humanizedNullFloat64(wallet, 1, "?")
	spText := humanizedNullInt64(sp, "?")
	unreadText := humanizedNullInt64(unread, "?")
	s := fmt.Sprintf(
		"Total: %d characters • %s ISK • %s SP  • %s unread",
		a.characters.Length(),
		walletText,
		spText,
		unreadText,
	)
	a.totalLabel.SetText(s)
}

func (a *overviewArea) updateEntries() (sql.NullInt64, sql.NullInt64, sql.NullFloat64, error) {
	var spTotal sql.NullInt64
	var unreadTotal sql.NullInt64
	var walletTotal sql.NullFloat64
	var err error
	mycc, err := a.ui.service.ListCharacters()
	if err != nil {
		return spTotal, unreadTotal, walletTotal, fmt.Errorf("failed to fetch characters: %w", err)
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
			c.home = sql.NullString{String: m.Home.Name, Valid: true}
		}
		if m.Location != nil {
			c.region = sql.NullString{String: m.Location.SolarSystem.Constellation.Region.Name, Valid: true}
			c.location = sql.NullString{String: m.Location.Name, Valid: true}
			c.systemName = sql.NullString{String: m.Location.SolarSystem.Name, Valid: true}
			c.systemSecurity = sql.NullFloat64{Float64: m.Location.SolarSystem.SecurityStatus, Valid: true}
		}
		if m.Ship != nil {
			c.ship = sql.NullString{String: m.Ship.Name, Valid: true}
		}
		cc[i] = c
	}
	for i, c := range cc {
		v, err := a.ui.service.GetCharacterTotalTrainingTime(c.id)
		if err != nil {
			return spTotal, unreadTotal, walletTotal, fmt.Errorf("failed to fetch skill queue count for character %d, %w", c.id, err)
		}
		cc[i].training = v
	}
	for i, c := range cc {
		total, unread, err := a.ui.service.GetCharacterMailCounts(c.id)
		if err != nil {
			return spTotal, unreadTotal, walletTotal, fmt.Errorf("failed to fetch mail counts for character %d, %w", c.id, err)
		}
		if total > 0 {
			cc[i].unreadCount.Int64 = int64(unread)
			cc[i].unreadCount.Valid = true
		}
	}
	if err := a.characters.Set(copyToUntypedSlice(cc)); err != nil {
		panic(err)
	}
	for _, c := range cc {
		if c.totalSP.Valid {
			spTotal.Valid = true
			spTotal.Int64 += c.totalSP.Int64
		}
		if c.unreadCount.Valid {
			unreadTotal.Valid = true
			unreadTotal.Int64 += c.unreadCount.Int64
		}
		if c.walletBalance.Valid {
			walletTotal.Valid = true
			walletTotal.Float64 += c.walletBalance.Float64
		}
	}
	return spTotal, unreadTotal, walletTotal, nil
}

func (a *overviewArea) StartUpdateTicker() {
	ticker := time.NewTicker(overviewUpdateTicker)
	go func() {
		for {
			func() {
				cc, err := a.ui.service.ListCharactersShort()
				if err != nil {
					slog.Error("Failed to fetch list of characters", "err", err)
					return
				}
				for _, c := range cc {
					a.MaybeUpdateAndRefresh(c.ID)
				}
			}()
			<-ticker.C
		}
	}()
}

func (a *overviewArea) MaybeUpdateAndRefresh(characterID int32) {
	sections := []model.CharacterSection{
		model.CharacterSectionHome,
		model.CharacterSectionLocation,
		model.CharacterSectionOnline,
		model.CharacterSectionShip,
		model.CharacterSectionSkills,
		model.CharacterSectionWalletBalance,
	}
	for _, s := range sections {
		go func(s model.CharacterSection) {
			changed, err := a.ui.service.UpdateCharacterSectionIfExpired(characterID, s)
			if err != nil {
				slog.Error("Failed to update character", "character", characterID, "section", s, "err", err)
				return
			}
			if changed {
				a.Refresh()
			}
		}(s)
	}

}

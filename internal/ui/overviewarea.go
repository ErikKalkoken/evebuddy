package ui

import (
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

type overviewCharacter struct {
	alliance       string
	birthday       time.Time
	corporation    string
	id             int32
	lastLoginAt    time.Time
	name           string
	systemName     string
	systemSecurity float64
	region         string
	ship           string
	security       float64
	sp             int
	training       types.NullDuration
	unreadCount    int
	walletBalance  float64
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
		{"SP", 80},
		{"Training", 80},
		{"Wallet", 80},
		{"System", 150},
		{"Region", 150},
		{"Ship", 150},
		{"Last Login", 100},
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
				l.Text = humanize.Comma(int64(c.unreadCount))
			case 5:
				l.Text = ihumanize.Number(float64(c.sp), 0)
			case 6:
				if !c.training.Valid {
					l.Text = "Inactive"
					l.Importance = widget.WarningImportance
				} else {
					l.Text = ihumanize.Duration(c.training.Duration)
				}
			case 7:
				l.Text = ihumanize.Number(c.walletBalance, 1)
			case 8:
				l.Text = fmt.Sprintf("%s %.1f", c.systemName, c.systemSecurity)
			case 9:
				l.Text = c.region
			case 10:
				l.Text = c.ship
			case 11:
				l.Text = humanize.RelTime(c.lastLoginAt, time.Now(), "", "")
			case 12:
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
	walletText := ihumanize.Number(wallet, 1)
	spText := ihumanize.Number(float64(sp), 0)
	unreadText := humanize.Comma(int64(unread))
	s := fmt.Sprintf(
		"Total: %d characters • %s ISK • %s SP  • %s unread",
		a.characters.Length(),
		walletText,
		spText,
		unreadText,
	)
	a.totalLabel.SetText(s)
}

func (a *overviewArea) updateEntries() (int, int, float64, error) {
	var err error
	mycc, err := a.ui.service.ListMyCharacters()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to fetch characters: %w", err)
	}
	cc := make([]overviewCharacter, len(mycc))
	for i, m := range mycc {
		var c overviewCharacter
		c.alliance = m.Character.AllianceName()
		c.birthday = m.Character.Birthday
		c.corporation = m.Character.Corporation.Name
		c.lastLoginAt = m.LastLoginAt
		c.id = m.ID
		c.name = m.Character.Name
		c.region = m.Location.Constellation.Region.Name
		c.security = m.Character.SecurityStatus
		c.ship = m.Ship.Name
		c.sp = m.SkillPoints
		c.systemName = m.Location.Name
		c.systemSecurity = m.Location.SecurityStatus
		c.walletBalance = m.WalletBalance
		cc[i] = c
	}
	for i, c := range cc {
		v, err := a.ui.service.GetTotalTrainingTime(c.id)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("failed to fetch skill queue count for character %d, %w", c.id, err)
		}
		cc[i].training = v
	}
	for i, c := range cc {
		v, err := a.ui.service.GetMailUnreadCount(c.id)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("failed to fetch unread count for character %d, %w", c.id, err)
		}
		cc[i].unreadCount = v
	}
	if err := a.characters.Set(copyToUntypedSlice(cc)); err != nil {
		panic(err)
	}
	var sp int
	var unread int
	var wallet float64
	for _, c := range cc {
		sp += c.sp
		unread += c.unreadCount
		wallet += c.walletBalance
	}
	return sp, unread, wallet, nil
}

func (a *overviewArea) StartUpdateTicker() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			func() {
				cc, err := a.ui.service.ListMyCharactersShort()
				if err != nil {
					slog.Error(err.Error())
					return
				}
				for _, c := range cc {
					go func(characterID int32) {
						isExpired, err := a.ui.service.SectionIsUpdateExpired(characterID, model.UpdateSectionMyCharacter)
						if err != nil {
							slog.Error(err.Error())
							return
						}
						if !isExpired {
							return
						}
						if err := a.ui.service.UpdateMyCharacter(characterID); err != nil {
							slog.Error(err.Error())
							return
						}
						a.Refresh()
					}(c.ID)
				}
			}()
			<-ticker.C
		}
	}()
}

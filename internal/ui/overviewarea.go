package ui

import (
	"fmt"
	"log/slog"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/dustin/go-humanize"
)

// overviewArea is the UI area that shows an overview of all the user's characters.
type overviewArea struct {
	content          *fyne.Container
	characters       []*model.MyCharacter
	skillqueueCounts []types.NullDuration
	unreadCounts     []int
	table            *widget.Table
	total            *widget.Label
	ui               *ui
}

func (u *ui) NewOverviewArea() *overviewArea {
	a := overviewArea{
		ui:               u,
		characters:       make([]*model.MyCharacter, 0),
		skillqueueCounts: make([]types.NullDuration, 0),
		unreadCounts:     make([]int, 0),
		total:            widget.NewLabel(""),
	}
	a.total.TextStyle.Bold = true
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

	table := widget.NewTable(
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
			c := a.characters[tci.Row]
			l.Importance = widget.MediumImportance
			switch tci.Col {
			case 0:
				l.Text = c.Character.Name
			case 1:
				l.Text = c.Character.Corporation.Name
			case 2:
				l.Text = c.Character.AllianceName()
			case 3:
				l.Text = fmt.Sprintf("%.1f", c.Character.SecurityStatus)
				if c.Character.SecurityStatus > 0 {
					l.Importance = widget.SuccessImportance
				} else if c.Character.SecurityStatus < 0 {
					l.Importance = widget.DangerImportance
				}
			case 4:
				l.Text = humanize.Comma(int64(a.unreadCounts[tci.Row]))
			case 5:
				l.Text = ihumanize.Number(float64(c.SkillPoints), 0)
			case 6:
				v := a.skillqueueCounts[tci.Row]
				if !v.Valid {
					l.Text = "Inactive"
					l.Importance = widget.WarningImportance
				} else {
					l.Text = ihumanize.Duration(v.Duration)
				}
			case 7:
				l.Text = ihumanize.Number(c.WalletBalance, 1)
			case 8:
				l.Text = fmt.Sprintf("%s %.1f", c.Location.Name, c.Location.SecurityStatus)
			case 9:
				l.Text = c.Location.Constellation.Region.Name
			case 10:
				l.Text = c.Ship.Name
			case 11:
				l.Text = humanize.RelTime(c.LastLoginAt, time.Now(), "", "")
			case 12:
				l.Text = humanize.RelTime(c.Character.Birthday, time.Now(), "", "")
			}
			l.Refresh()
		},
	)
	table.ShowHeaderRow = true
	table.StickyColumnCount = 1
	table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	table.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		co.(*widget.Label).SetText(s.text)
	}
	table.OnSelected = func(tci widget.TableCellID) {
		myC := a.characters[tci.Row]
		c, err := a.ui.service.GetMyCharacter(myC.ID)
		if err != nil {
			slog.Error("Failed to fetch character", "characterID", c.ID, "err", err)
			a.ui.statusArea.SetError("Failed to fetch character")
			return
		}
		switch tci.Col {
		case 4:
			a.ui.SetCurrentCharacter(c)
			a.ui.tabs.SelectIndex(0)
		case 6:
			a.ui.SetCurrentCharacter(c)
			a.ui.tabs.SelectIndex(1)
		case 7:
			a.ui.SetCurrentCharacter(c)
			a.ui.tabs.SelectIndex(2)
		}
	}

	for i, h := range headers {
		table.SetColumnWidth(i, h.width)
	}

	top := container.NewVBox(a.total, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, table)
	a.table = table
	return &a
}

func (a *overviewArea) Refresh() {
	a.updateEntries()
	a.table.Refresh()
	wallet, sp := a.makeWalletSPText()
	unread := a.makeUnreadText()
	s := fmt.Sprintf(
		"Total: %d characters • %s ISK • %s SP  • %s unread",
		len(a.characters),
		wallet,
		sp,
		unread,
	)
	a.total.SetText(s)
}

func (a *overviewArea) updateEntries() {
	a.characters = a.characters[0:0]
	var err error
	a.characters, err = a.ui.service.ListMyCharacters()
	if err != nil {
		slog.Error("failed to fetch characters", "err", err)
		return
	}

	a.skillqueueCounts = slices.Grow(a.skillqueueCounts, len(a.characters))
	a.skillqueueCounts = a.skillqueueCounts[0:len(a.characters)]
	for i, c := range a.characters {
		v, err := a.ui.service.GetTotalTrainingTime(c.ID)
		if err != nil {
			slog.Error("failed to fetch skill queue count", "characterID", c.ID, "err", err)
			continue
		}
		a.skillqueueCounts[i] = v
	}

	a.unreadCounts = slices.Grow(a.unreadCounts, len(a.characters))
	a.unreadCounts = a.unreadCounts[0:len(a.characters)]
	for i, c := range a.characters {
		v, err := a.ui.service.GetMailUnreadCount(c.ID)
		if err != nil {
			slog.Error("failed to fetch unread count", "characterID", c.ID, "err", err)
			continue
		}
		a.unreadCounts[i] = v
	}
}

func (a *overviewArea) makeWalletSPText() (string, string) {
	if len(a.unreadCounts) == 0 {
		return "?", "?"
	}
	var wallet float64
	var sp int
	for _, c := range a.characters {
		wallet += c.WalletBalance
		sp += c.SkillPoints
	}
	walletText := ihumanize.Number(wallet, 1)
	spText := ihumanize.Number(float64(sp), 0)
	return walletText, spText
}

func (a *overviewArea) makeUnreadText() string {
	if len(a.unreadCounts) == 0 {
		return "?"
	}
	var totalUnread int
	for _, x := range a.unreadCounts {
		totalUnread += x
	}
	unreadText := humanize.Comma(int64(totalUnread))
	return unreadText
}

func (a *overviewArea) StartUpdateTicker() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			func() {
				for _, c := range a.characters {
					go func(characterID int32) {
						isExpired, err := a.ui.service.SectionIsUpdateExpired(characterID, service.UpdateSectionMyCharacter)
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

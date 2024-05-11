package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
	"github.com/dustin/go-humanize"
)

// overviewArea is the UI area that shows the skillqueue
type overviewArea struct {
	content    *fyne.Container
	characters []*model.MyCharacter
	table      *widgets.StaticTable
	ui         *ui
}

func (u *ui) NewOverviewArea() *overviewArea {
	a := overviewArea{
		ui:         u,
		characters: make([]*model.MyCharacter, 0),
	}
	a.updateEntries()
	headers := []string{"Name", "Organization", "Security", "Wallet", "SP", "Location", "Ship", "Last Login", "Age"}
	table := widgets.NewStaticTable(
		func() (rows int, cols int) {
			return len(a.characters), len(headers)
		},
		func() fyne.CanvasObject {
			x := widget.NewLabel("PLACEHOLDER")
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
				l.Text = fmt.Sprintf("%s\n%s", c.Character.Corporation.Name, c.Character.AllianceName())
			case 2:
				l.Text = fmt.Sprintf("%.1f", c.Character.SecurityStatus)
				if c.Character.SecurityStatus > 0 {
					l.Importance = widget.SuccessImportance
				} else if c.Character.SecurityStatus < 0 {
					l.Importance = widget.DangerImportance
				}
			case 3:
				l.Text = ihumanize.Number(c.WalletBalance, 2)
			case 4:
				l.Text = ihumanize.Number(float64(c.SkillPoints), 0)
			case 5:
				l.Text = fmt.Sprintf("%s\n%s", c.Location.Name, c.Location.Constellation.Region.Name)
			case 6:
				l.Text = c.Ship.Name
			case 7:
				l.Text = humanize.Time(c.LastLoginAt)
			case 8:
				l.Text = humanize.RelTime(c.Character.Birthday, time.Now(), "", "")
			}
			l.Refresh()
		},
	)
	table.SetColumnWidth(0, 200)
	table.SetColumnWidth(1, 200)
	table.SetColumnWidth(2, 100)
	table.SetColumnWidth(3, 100)
	table.SetColumnWidth(4, 100)
	table.SetColumnWidth(5, 150)
	table.SetColumnWidth(6, 150)
	table.SetColumnWidth(7, 100)
	table.SetColumnWidth(8, 100)

	table.ShowHeaderRow = true
	table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	table.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		co.(*widget.Label).SetText(s)
	}

	a.content = container.NewBorder(nil, nil, nil, nil, table)
	a.table = table
	return &a
}

func (a *overviewArea) Refresh() {
	a.updateEntries()
	for i := range a.characters {
		a.table.SetRowHeight(i, 50)
	}
	a.table.Refresh()
}

func (a *overviewArea) updateEntries() {
	a.characters = a.characters[0:0]
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		return
	}
	var err error
	a.characters, err = a.ui.service.ListMyCharacters()
	if err != nil {
		slog.Error("failed to fetch characters", "err", err)
		return
	}
}

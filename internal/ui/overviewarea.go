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
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
	"github.com/dustin/go-humanize"
)

// overviewArea is the UI area that shows the skillqueue
type overviewArea struct {
	content          *fyne.Container
	characters       []*model.MyCharacter
	skillqueueCounts []types.NullDuration
	table            *widgets.StaticTable
	ui               *ui
}

func (u *ui) NewOverviewArea() *overviewArea {
	a := overviewArea{
		ui:               u,
		characters:       make([]*model.MyCharacter, 0),
		skillqueueCounts: make([]types.NullDuration, 0),
	}
	var headers = []struct {
		text  string
		width float32
	}{
		{"Name", 200},
		{"Corporation", 200},
		{"Alliance", 200},
		{"Security", 80},
		{"Wallet", 80},
		{"SP", 80},
		{"Training", 80},
		{"Location\nSystem", 150},
		{"Location\nRegion", 150},
		{"Ship", 150},
		{"Last Login", 100},
		{"Age", 100},
	}

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
				l.Text = ihumanize.Number(c.WalletBalance, 2)
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
				l.Text = fmt.Sprintf("%s %.1f", c.Location.Name, c.Location.SecurityStatus)
				// switch s := c.Location.SecurityStatus; {
				// case s < 0:
				// 	l.Importance = widget.DangerImportance
				// case s <= 0.5:
				// 	l.Importance = widget.WarningImportance
				// case s > 0.5:
				// 	l.Importance = widget.SuccessImportance
				// }
			case 8:
				l.Text = c.Location.Constellation.Region.Name
			case 9:
				l.Text = c.Ship.Name
			case 10:
				l.Text = humanize.Time(c.LastLoginAt)
			case 11:
				l.Text = humanize.RelTime(c.Character.Birthday, time.Now(), "", "")
			}
			l.Refresh()
		},
	)
	table.ShowHeaderRow = true
	table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template\nSecond")
	}
	table.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		co.(*widget.Label).SetText(s.text)
	}

	for i, h := range headers {
		table.SetColumnWidth(i, h.width)
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
}

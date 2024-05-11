package ui

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
)

// overviewArea is the UI area that shows the skillqueue
type overviewArea struct {
	content    *fyne.Container
	characters []*model.MyCharacterShort
	table      *widgets.StaticTable
	ui         *ui
}

func (u *ui) NewOverviewArea() *overviewArea {
	a := overviewArea{
		ui:         u,
		characters: make([]*model.MyCharacterShort, 0),
	}
	a.updateEntries()
	table := widgets.NewStaticTable(
		func() (rows int, cols int) {
			return len(a.characters), 2
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("PLACEHOLDER")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			e := a.characters[tci.Row]
			switch tci.Col {
			case 0:
				l.Text = e.Name
			case 1:
				l.Text = e.CorporationName
			}
			l.Refresh()
		},
	)
	table.SetColumnWidth(0, 200)
	table.SetColumnWidth(1, 200)

	table.ShowHeaderRow = true
	table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	table.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		var s string
		switch tci.Col {
		case 0:
			s = "Name"
		case 1:
			s = "Corporation"
		}
		co.(*widget.Label).SetText(s)
	}

	a.content = container.NewBorder(nil, nil, nil, nil, table)
	a.table = table
	return &a
}

func (a *overviewArea) Refresh() {
	a.updateEntries()
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

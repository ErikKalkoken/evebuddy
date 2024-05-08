package ui

import (
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
	"github.com/dustin/go-humanize"
)

// walletTransactionArea is the UI area that shows the skillqueue
type walletTransactionArea struct {
	content *fyne.Container
	ui      *ui
}

func (u *ui) NewWalletTransactionArea() *walletTransactionArea {
	c := walletTransactionArea{ui: u, content: container.NewStack()}
	return &c
}

func (a *walletTransactionArea) Redraw() {
	a.content.RemoveAll()
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		return
	}
	ee, err := a.ui.service.ListWalletJournalEntries(characterID)
	if err != nil {
		slog.Error("failed to fetch wallet journal", "err", err)
		a.content.Add(makeMessage("Failed to fetch wallet journal", widget.DangerImportance))
		return
	}
	t := widgets.NewStaticTable(
		func() (rows int, cols int) {
			return len(ee), 5
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			var s string
			e := ee[tci.Row]
			switch tci.Col {
			case 0:
				s = e.Date.Format(myDateTime)
			case 1:
				s = e.Type()
			case 2:
				s = humanize.FormatFloat("#,###.##", e.Amount)
			case 3:
				s = humanize.FormatFloat("#,###.##", e.Balance)
			case 4:
				s = e.Description
			}
			co.(*widget.Label).SetText(s)
		},
	)
	t.SetColumnWidth(0, 150)
	t.SetColumnWidth(1, 150)
	t.SetColumnWidth(2, 150)
	t.SetColumnWidth(3, 150)
	t.SetColumnWidth(4, 500)

	t.ShowHeaderRow = true
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		var s string
		switch tci.Col {
		case 0:
			s = "Date"
		case 1:
			s = "Type"
		case 2:
			s = "Amount"
		case 3:
			s = "Balance"
		case 4:
			s = "Description"
		}
		co.(*widget.Label).SetText(s)
	}

	a.content.Add(t)
}

func (a *walletTransactionArea) StartUpdateTicker() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			func() {
				characterID := a.ui.CurrentCharID()
				if characterID == 0 {
					return
				}
				if !a.ui.service.SectionUpdatedExpired(characterID, service.UpdateSectionWalletJournal) {
					return
				}
				if err := a.ui.service.UpdateWalletJournalEntryESI(characterID); err != nil {
					slog.Error(err.Error())
					return
				}
				a.Redraw()
			}()
			<-ticker.C
		}
	}()
}

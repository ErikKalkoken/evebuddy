package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
	"github.com/dustin/go-humanize"
)

const myFloatFormat = "#,###.##"

// walletTransactionArea is the UI area that shows the skillqueue
type walletTransactionArea struct {
	content   *fyne.Container
	entries   []*model.WalletJournalEntry
	errorText string
	table     *widgets.StaticTable
	total     *widget.Label
	ui        *ui
}

func (u *ui) NewWalletTransactionArea() *walletTransactionArea {
	a := walletTransactionArea{
		ui:      u,
		entries: make([]*model.WalletJournalEntry, 0),
	}
	a.updateEntries()
	table := widgets.NewStaticTable(
		func() (rows int, cols int) {
			return len(a.entries), 5
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("2024-05-08 18:59")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			l.Importance = widget.MediumImportance
			l.Alignment = fyne.TextAlignLeading
			e := a.entries[tci.Row]
			switch tci.Col {
			case 0:
				l.Text = e.Date.Format(myDateTime)
			case 1:
				l.Text = e.Type()
			case 2:
				l.Alignment = fyne.TextAlignTrailing
				l.Text = humanize.FormatFloat(myFloatFormat, e.Amount)
				switch {
				case e.Amount < 0:
					l.Importance = widget.DangerImportance
				case e.Amount > 0:
					l.Importance = widget.SuccessImportance
				default:
					l.Importance = widget.MediumImportance
				}
			case 3:
				l.Alignment = fyne.TextAlignTrailing
				l.Text = humanize.FormatFloat(myFloatFormat, e.Balance)
			case 4:
				l.Truncation = fyne.TextTruncateEllipsis
				l.Text = e.Description
			}
			l.Refresh()
		},
	)
	table.SetColumnWidth(0, 130)
	table.SetColumnWidth(1, 130)
	table.SetColumnWidth(2, 130)
	table.SetColumnWidth(3, 130)
	table.SetColumnWidth(4, 450)

	table.ShowHeaderRow = true
	table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	table.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
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

	s, i := a.makeBottomText()
	total := widget.NewLabel(s)
	total.Importance = i
	bottom := container.NewVBox(widget.NewSeparator(), total)

	a.content = container.NewBorder(nil, bottom, nil, nil, table)
	a.table = table
	a.total = total
	return &a
}

func (a *walletTransactionArea) Refresh() {
	a.updateEntries()
	a.table.Refresh()
	s, i := a.makeBottomText()
	a.total.Text = s
	a.total.Importance = i
	a.total.Refresh()
}

func (a *walletTransactionArea) updateEntries() {
	a.entries = a.entries[0:0]
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		return
	}
	var err error
	a.entries, err = a.ui.service.ListWalletJournalEntries(characterID)
	if err != nil {
		slog.Error("failed to fetch wallet journal", "err", err)
		c := a.ui.CurrentChar()
		a.errorText = fmt.Sprintf("Failed to fetch wallet journal for %s", c.Character.Name)
		return
	}
}

func (a *walletTransactionArea) makeBottomText() (string, widget.Importance) {
	var s string
	var i widget.Importance
	if len(a.entries) > 0 {
		s = fmt.Sprintf("Total: %s", humanize.FormatFloat(myFloatFormat, a.entries[0].Balance))
		i = widget.MediumImportance
	} else {
		s = "No entries"
		i = widget.WarningImportance
	}
	return s, i
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
				count, err := a.ui.service.UpdateWalletJournalEntryESI(characterID)
				if err != nil {
					slog.Error(err.Error())
					return
				}
				if count > 0 {
					a.Refresh()
				}
			}()
			<-ticker.C
		}
	}()
}
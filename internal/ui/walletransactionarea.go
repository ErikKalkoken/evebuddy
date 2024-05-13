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
		total:   widget.NewLabel(""),
	}
	a.total.TextStyle.Bold = true
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

	top := container.NewVBox(a.total, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, table)
	a.table = table
	return &a
}

func (a *walletTransactionArea) Refresh() {
	a.updateEntries()
	a.table.Refresh()
	s, i := a.makeTopText()
	a.total.Text = s
	a.total.Importance = i
	a.total.Refresh()
}

func (a *walletTransactionArea) makeTopText() (string, widget.Importance) {
	c := a.ui.CurrentChar()
	if c == nil {
		return "No data", widget.LowImportance
	}
	hasData, err := a.ui.service.SectionWasUpdated(c.ID, service.UpdateSectionWalletJournal)
	if err != nil {
		return "ERROR", widget.DangerImportance
	}
	if !hasData {
		return "No data", widget.LowImportance
	}
	s := fmt.Sprintf("Balance: %s", ihumanize.Number(c.WalletBalance, 1))
	return s, widget.MediumImportance
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

func (a *walletTransactionArea) StartUpdateTicker() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			func() {
				characterID := a.ui.CurrentCharID()
				if characterID == 0 {
					return
				}
				isExpired, err := a.ui.service.SectionIsUpdateExpired(characterID, service.UpdateSectionWalletJournal)
				if err != nil {
					slog.Error(err.Error())
					return
				}
				if !isExpired {
					return
				}
				_, err = a.ui.service.UpdateWalletJournalEntryESI(characterID)
				if err != nil {
					slog.Error(err.Error())
					return
				}
				a.Refresh()
			}()
			<-ticker.C
		}
	}()
}

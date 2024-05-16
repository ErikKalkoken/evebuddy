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

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
)

const myFloatFormat = "#,###.##"

type walletJournalEntry struct {
	date        time.Time
	refType     string
	amount      float64
	balance     float64
	description string
}

// walletTransactionArea is the UI area that shows the skillqueue
type walletTransactionArea struct {
	content *fyne.Container
	entries binding.UntypedList // []walletJournalEntry
	table   *widgets.StaticTable
	total   *widget.Label
	ui      *ui
}

func (u *ui) NewWalletTransactionArea() *walletTransactionArea {
	a := walletTransactionArea{
		ui:      u,
		entries: binding.NewUntypedList(),
		total:   widget.NewLabel(""),
	}
	a.total.TextStyle.Bold = true
	t := widgets.NewStaticTable(
		func() (rows int, cols int) {
			return a.entries.Length(), 5
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("2024-05-08 18:59")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			l.Importance = widget.MediumImportance
			l.Alignment = fyne.TextAlignLeading
			w, err := getFromBoundUntypedList[walletJournalEntry](a.entries, tci.Row)
			if err != nil {
				slog.Error("failed to render cell in wallet journal table", "err", err)
				l.Text = "failed to render"
				l.Importance = widget.DangerImportance
				l.Refresh()
				return
			}
			switch tci.Col {
			case 0:
				l.Text = w.date.Format(myDateTime)
			case 1:
				l.Text = w.refType
			case 2:
				l.Alignment = fyne.TextAlignTrailing
				l.Text = humanize.FormatFloat(myFloatFormat, w.amount)
				switch {
				case w.amount < 0:
					l.Importance = widget.DangerImportance
				case w.amount > 0:
					l.Importance = widget.SuccessImportance
				default:
					l.Importance = widget.MediumImportance
				}
			case 3:
				l.Alignment = fyne.TextAlignTrailing
				l.Text = humanize.FormatFloat(myFloatFormat, w.balance)
			case 4:
				l.Truncation = fyne.TextTruncateEllipsis
				l.Text = w.description
			}
			l.Refresh()
		},
	)
	t.SetColumnWidth(0, 130)
	t.SetColumnWidth(1, 130)
	t.SetColumnWidth(2, 130)
	t.SetColumnWidth(3, 130)
	t.SetColumnWidth(4, 450)

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

	top := container.NewVBox(a.total, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, t)
	a.table = t
	a.entries.AddListener(binding.NewDataListener(func() {
		a.table.Refresh()
	}))
	return &a
}

func (a *walletTransactionArea) Refresh() {
	a.updateEntries()
	s, i := a.makeTopText()
	a.total.Text = s
	a.total.Importance = i
	a.total.Refresh()
}

func (a *walletTransactionArea) makeTopText() (string, widget.Importance) {
	c := a.ui.CurrentChar()
	if c == nil {
		return "No data yet...", widget.LowImportance
	}
	hasData, err := a.ui.service.SectionWasUpdated(c.ID, model.UpdateSectionWalletJournal)
	if err != nil {
		return "ERROR", widget.DangerImportance
	}
	if !hasData {
		return "No data yet...", widget.LowImportance
	}
	s := fmt.Sprintf("Balance: %s", humanizedNullFloat64(c.WalletBalance, 1, "?"))
	return s, widget.MediumImportance
}

func (a *walletTransactionArea) updateEntries() error {
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		x := make([]any, 0)
		err := a.entries.Set(x)
		if err != nil {
			return err
		}
	}
	ww, err := a.ui.service.ListWalletJournalEntries(characterID)
	if err != nil {
		return fmt.Errorf("failed to fetch wallet journal for character %d: %w", characterID, err)
	}
	entries := make([]walletJournalEntry, len(ww))
	for i, w := range ww {
		var w2 walletJournalEntry
		w2.amount = w.Amount
		w2.balance = w.Balance
		w2.date = w.Date
		w2.description = w.Description
		w2.refType = w.Type()
		entries[i] = w2
	}
	if err := a.entries.Set(copyToUntypedSlice(entries)); err != nil {
		return err
	}
	return nil
}

func (a *walletTransactionArea) StartUpdateTicker() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			func() {
				cc, err := a.ui.service.ListMyCharactersShort()
				if err != nil {
					slog.Error("Failed to fetch list of my characters", "err", err)
					return
				}
				for _, c := range cc {
					a.UpdateAndRefresh(c.ID)
				}
			}()
			<-ticker.C
		}
	}()
}

func (a *walletTransactionArea) UpdateAndRefresh(characterID int32) {
	changed, err := a.ui.service.UpdateSectionIfExpired(characterID, model.UpdateSectionWalletJournal)
	if err != nil {
		slog.Error("Failed to update wallet transaction", "character", characterID, "err", err)
		return
	}
	if changed && characterID == a.ui.CurrentCharID() {
		a.Refresh()
	}
}

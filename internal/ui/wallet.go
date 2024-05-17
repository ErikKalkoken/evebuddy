package ui

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

const walletTransactionUpdateTicker = 60 * time.Second

type walletJournalEntry struct {
	amount      float64
	balance     float64
	date        time.Time
	description string
	refType     string
	reason      string
}

func (e walletJournalEntry) hasReason() bool {
	return e.reason != ""
}

func (e walletJournalEntry) refTypeOutput() string {
	s := strings.ReplaceAll(e.refType, "_", " ")
	c := cases.Title(language.English)
	s = c.String(s)
	return s
}

func (e walletJournalEntry) descriptionWithReason() string {
	if e.reason == "" {
		return e.description
	}
	return fmt.Sprintf("[r] %s", e.description)
}

// walletTransactionArea is the UI area that shows the skillqueue
type walletTransactionArea struct {
	content *fyne.Container
	entries binding.UntypedList // []walletJournalEntry
	table   *widget.Table
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
	t := widget.NewTable(
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
				l.Text = w.refTypeOutput()
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
				l.Text = w.descriptionWithReason()
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
	t.OnSelected = func(tci widget.TableCellID) {
		e, err := getFromBoundUntypedList[walletJournalEntry](a.entries, tci.Row)
		if err != nil {
			slog.Error("failed to access entries in list", "err", err)
			return
		}
		if e.hasReason() {
			c := widget.NewLabel(e.reason)
			dlg := dialog.NewCustom("Reason", "OK", c, u.window)
			dlg.Show()
		}
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
		w2.reason = w.Reason
		w2.refType = w.RefType
		entries[i] = w2
	}
	if err := a.entries.Set(copyToUntypedSlice(entries)); err != nil {
		return err
	}
	return nil
}

func (a *walletTransactionArea) StartUpdateTicker() {
	ticker := time.NewTicker(walletTransactionUpdateTicker)
	go func() {
		for {
			func() {
				cc, err := a.ui.service.ListMyCharactersShort()
				if err != nil {
					slog.Error("Failed to fetch list of my characters", "err", err)
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

func (a *walletTransactionArea) MaybeUpdateAndRefresh(characterID int32) {
	changed, err := a.ui.service.UpdateSectionIfExpired(characterID, model.UpdateSectionWalletJournal)
	if err != nil {
		slog.Error("Failed to update wallet transaction", "character", characterID, "err", err)
		return
	}
	if changed && characterID == a.ui.CurrentCharID() {
		a.Refresh()
	}
}

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

// walletJournalArea is the UI area that shows the skillqueue
type walletJournalArea struct {
	content *fyne.Container
	entries binding.UntypedList // []walletJournalEntry
	table   *widget.Table
	top     *widget.Label
	ui      *ui
}

func (u *ui) newWalletJournalArea() *walletJournalArea {
	a := walletJournalArea{
		ui:      u,
		entries: binding.NewUntypedList(),
		top:     widget.NewLabel(""),
	}
	a.top.TextStyle.Bold = true
	var headers = []struct {
		text  string
		width float32
	}{
		{"Date", 130},
		{"Type", 130},
		{"Amount", 130},
		{"Balance", 130},
		{"Description", 450},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return a.entries.Length(), len(headers)
		},
		func() fyne.CanvasObject {
			x := widget.NewLabel("Template")
			x.Truncation = fyne.TextTruncateEllipsis
			return x
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			l.Importance = widget.MediumImportance
			l.Alignment = fyne.TextAlignLeading
			w, err := getItemUntypedList[walletJournalEntry](a.entries, tci.Row)
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
	t.ShowHeaderRow = true
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		co.(*widget.Label).SetText(s.text)
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.width)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		e, err := getItemUntypedList[walletJournalEntry](a.entries, tci.Row)
		if err != nil {
			t := "Failed to select wallet journal entry"
			slog.Error(t, "err", err)
			a.ui.statusBarArea.SetError(t)
			return
		}
		if e.hasReason() {
			c := widget.NewLabel(e.reason)
			dlg := dialog.NewCustom("Reason", "OK", c, u.window)
			dlg.Show()
		}
	}

	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, t)
	a.table = t
	return &a
}

func (a *walletJournalArea) refresh() {
	t, i, err := func() (string, widget.Importance, error) {
		if err := a.updateEntries(); err != nil {
			return "", 0, err
		}
		return a.makeTopText()
	}()
	if err != nil {
		slog.Error("Failed to refresh wallet journal UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.table.Refresh()
}

func (a *walletJournalArea) makeTopText() (string, widget.Importance, error) {
	if !a.ui.hasCharacter() {
		return "No character", widget.LowImportance, nil
	}
	c := a.ui.currentChar()
	hasData, err := a.ui.service.CharacterSectionWasUpdated(c.ID, model.CharacterSectionWalletJournal)
	if err != nil {
		return "", 0, err
	}
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	s := fmt.Sprintf("Balance: %s", humanizedNullFloat64(c.WalletBalance, 1, "?"))
	return s, widget.MediumImportance, nil
}

func (a *walletJournalArea) updateEntries() error {
	if !a.ui.hasCharacter() {
		x := make([]any, 0)
		err := a.entries.Set(x)
		if err != nil {
			return err
		}
	}
	characterID := a.ui.currentCharID()
	ww, err := a.ui.service.ListCharacterWalletJournalEntries(characterID)
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

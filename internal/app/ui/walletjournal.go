package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/app/humanize"
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
	entries []walletJournalEntry
	table   *widget.Table
	top     *widget.Label
	u       *UI
}

func (u *UI) newWalletJournalArea() *walletJournalArea {
	a := walletJournalArea{
		entries: make([]walletJournalEntry, 0),
		top:     widget.NewLabel(""),
		u:       u,
	}

	a.top.TextStyle.Bold = true
	a.table = a.makeTable()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, a.table)
	return &a
}

func (a *walletJournalArea) makeTable() *widget.Table {
	var headers = []struct {
		text  string
		width float32
	}{
		{"Date", 150},
		{"Type", 150},
		{"Amount", 200},
		{"Balance", 200},
		{"Description", 450},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.entries), len(headers)
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
			if tci.Row >= len(a.entries) {
				return
			}
			w := a.entries[tci.Row]
			switch tci.Col {
			case 0:
				l.Text = w.date.Format(app.TimeDefaultFormat)
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
		defer t.UnselectAll()
		if tci.Row >= len(a.entries) {
			return
		}
		e := a.entries[tci.Row]
		if e.hasReason() {
			c := widget.NewLabel(e.reason)
			dlg := dialog.NewCustom("Reason", "OK", c, a.u.window)
			dlg.Show()
		}
	}
	return t
}

func (a *walletJournalArea) refresh() {
	var t string
	var i widget.Importance
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh wallet journal UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeTopText()
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.table.Refresh()
}

func (a *walletJournalArea) makeTopText() (string, widget.Importance) {
	if !a.u.hasCharacter() {
		return "No character", widget.LowImportance
	}
	c := a.u.currentCharacter()
	hasData := a.u.StatusCacheService.CharacterSectionExists(c.ID, app.SectionWalletJournal)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	b := ihumanize.OptionalFloat(c.WalletBalance, 1, "?")
	t := humanize.Comma(int64(len(a.entries)))
	s := fmt.Sprintf("Balance: %s • Entries: %s", b, t)
	return s, widget.MediumImportance
}

func (a *walletJournalArea) updateEntries() error {
	if !a.u.hasCharacter() {
		a.entries = make([]walletJournalEntry, 0)
		return nil
	}
	characterID := a.u.characterID()
	ww, err := a.u.CharacterService.ListCharacterWalletJournalEntries(context.TODO(), characterID)
	if err != nil {
		return fmt.Errorf("fetch wallet journal for character %d: %w", characterID, err)
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
	a.entries = entries
	return nil
}

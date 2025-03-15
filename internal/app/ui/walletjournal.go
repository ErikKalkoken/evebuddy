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
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
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
	titler := cases.Title(language.English)
	return titler.String(strings.ReplaceAll(e.refType, "_", " "))
}

func (e walletJournalEntry) descriptionWithReason() string {
	if e.reason == "" {
		return e.description
	}
	return fmt.Sprintf("[r] %s", e.description)
}

// WalletJournalArea is the UI area that shows the skillqueue
type WalletJournalArea struct {
	Content *fyne.Container

	OnRefresh func(balance string)

	rows []walletJournalEntry
	body fyne.CanvasObject
	top  *widget.Label
	u    *BaseUI
}

func NewWalletJournalArea(u *BaseUI) *WalletJournalArea {
	a := WalletJournalArea{
		rows: make([]walletJournalEntry, 0),
		top:  MakeTopLabel(),
		u:    u,
	}
	var headers = []headerDef{
		{"Date", 150},
		{"Type", 150},
		{"Amount", 200},
		{"Balance", 200},
		{"Description", 450},
	}
	makeDataLabel := func(col int, w walletJournalEntry) (string, fyne.TextAlign, widget.Importance) {
		var align fyne.TextAlign
		var importance widget.Importance
		var text string
		switch col {
		case 0:
			text = w.date.Format(app.DateTimeFormat)
		case 1:
			text = w.refTypeOutput()
		case 2:
			align = fyne.TextAlignTrailing
			text = humanize.FormatFloat(app.FloatFormat, w.amount)
			switch {
			case w.amount < 0:
				importance = widget.DangerImportance
			case w.amount > 0:
				importance = widget.SuccessImportance
			default:
				importance = widget.MediumImportance
			}
		case 3:
			align = fyne.TextAlignTrailing
			text = humanize.FormatFloat(app.FloatFormat, w.balance)
		case 4:
			text = w.descriptionWithReason()
		}
		return text, align, importance
	}
	showReasonDialog := func(r walletJournalEntry) {
		if r.hasReason() {
			dlg := dialog.NewCustom("Reason", "OK", widget.NewLabel(r.reason), a.u.Window)
			dlg.Show()
		}
	}
	if a.u.IsDesktop() {
		a.body = makeDataTableForDesktop(headers, &a.rows, makeDataLabel, func(_ int, r walletJournalEntry) {
			showReasonDialog(r)
		})
	} else {
		a.body = makeDataTableForMobile(headers, &a.rows, makeDataLabel, showReasonDialog)
	}
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.Content = container.NewBorder(top, nil, nil, nil, a.body)
	return &a
}

func (a *WalletJournalArea) Refresh() {
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
	a.body.Refresh()
}

func (a *WalletJournalArea) makeTopText() (string, widget.Importance) {
	if !a.u.HasCharacter() {
		return "No character", widget.LowImportance
	}
	c := a.u.CurrentCharacter()
	hasData := a.u.StatusCacheService.CharacterSectionExists(c.ID, app.SectionWalletJournal)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	b := ihumanize.OptionalFloat(c.WalletBalance, 1, "?")
	t := humanize.Comma(int64(len(a.rows)))
	s := fmt.Sprintf("Balance: %s • Entries: %s", b, t)
	if a.OnRefresh != nil {
		a.OnRefresh(b)
	}
	return s, widget.MediumImportance
}

func (a *WalletJournalArea) updateEntries() error {
	if !a.u.HasCharacter() {
		a.rows = make([]walletJournalEntry, 0)
		return nil
	}
	characterID := a.u.CurrentCharacterID()
	ww, err := a.u.CharacterService.ListCharacterWalletJournalEntries(context.TODO(), characterID)
	if err != nil {
		return err
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
	a.rows = entries
	return nil
}

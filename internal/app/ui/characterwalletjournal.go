package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
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

type CharacterWalletJournal struct {
	widget.BaseWidget

	OnUpdate func(balance string)

	rows []walletJournalEntry
	body fyne.CanvasObject
	top  *widget.Label
	u    *BaseUI
}

func NewCharacterWalletJournal(u *BaseUI) *CharacterWalletJournal {
	a := &CharacterWalletJournal{
		rows: make([]walletJournalEntry, 0),
		top:  appwidget.MakeTopLabel(),
		u:    u,
	}
	a.ExtendBaseWidget(a)
	headers := []iwidget.HeaderDef{
		{Text: "Date", Width: 150},
		{Text: "Type", Width: 150},
		{Text: "Amount", Width: 200},
		{Text: "Balance", Width: 200},
		{Text: "Description", Width: 450},
	}
	makeCell := func(col int, w walletJournalEntry) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(w.date.Format(app.DateTimeFormat))
		case 1:
			return iwidget.NewRichTextSegmentFromText(w.refTypeOutput())
		case 2:
			var color fyne.ThemeColorName
			switch {
			case w.amount < 0:
				color = theme.ColorNameError
			case w.amount > 0:
				color = theme.ColorNameSuccess
			default:
				color = theme.ColorNameForeground
			}
			return iwidget.NewRichTextSegmentFromText(
				humanize.FormatFloat(app.FloatFormat, w.amount),
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					ColorName: color,
				},
			)
		case 3:
			return iwidget.NewRichTextSegmentFromText(
				humanize.FormatFloat(app.FloatFormat, w.balance),
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				},
			)
		case 4:
			return iwidget.NewRichTextSegmentFromText(w.descriptionWithReason())
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	showReasonDialog := func(r walletJournalEntry) {
		if r.hasReason() {
			a.u.ShowInformationDialog("Reason", r.reason, a.u.MainWindow())
		}
	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop(headers, &a.rows, makeCell, func(_ int, r walletJournalEntry) {
			showReasonDialog(r)
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile(headers, &a.rows, makeCell, showReasonDialog)
	}
	return a
}

func (a *CharacterWalletJournal) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterWalletJournal) Update() {
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

func (a *CharacterWalletJournal) makeTopText() (string, widget.Importance) {
	if !a.u.HasCharacter() {
		return "No character", widget.LowImportance
	}
	c := a.u.CurrentCharacter()
	hasData := a.u.StatusCacheService().CharacterSectionExists(c.ID, app.SectionWalletJournal)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	b := ihumanize.OptionalFloat(c.WalletBalance, 1, "?")
	t := humanize.Comma(int64(len(a.rows)))
	s := fmt.Sprintf("Balance: %s • Entries: %s", b, t)
	if a.OnUpdate != nil {
		a.OnUpdate(b)
	}
	return s, widget.MediumImportance
}

func (a *CharacterWalletJournal) updateEntries() error {
	if !a.u.HasCharacter() {
		a.rows = make([]walletJournalEntry, 0)
		return nil
	}
	characterID := a.u.CurrentCharacterID()
	ww, err := a.u.CharacterService().ListWalletJournalEntries(context.TODO(), characterID)
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

package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type walletJournalRow struct {
	amount         float64
	amountDisplay  []widget.RichTextSegment
	balance        float64
	date           time.Time
	description    string
	reason         string
	refType        string
	refTypeDisplay string
}

func (e walletJournalRow) hasReason() bool {
	return e.reason != ""
}

func (e walletJournalRow) descriptionWithReason() string {
	if e.reason == "" {
		return e.description
	}
	return fmt.Sprintf("[r] %s", e.description)
}

type characterWalletJournal struct {
	widget.BaseWidget

	OnUpdate func(balance string)

	body         fyne.CanvasObject
	columnSorter *columnSorter
	rows         []walletJournalRow
	rowsFiltered []walletJournalRow
	selectType   *iwidget.FilterChipSelect
	sortButton   *sortButton
	top          *widget.Label
	u            *baseUI
}

func newCharacterWalletJournal(u *baseUI) *characterWalletJournal {
	headers := []headerDef{
		{Text: "Date", Width: 150},
		{Text: "Type", Width: 150},
		{Text: "Amount", Width: 200},
		{Text: "Balance", Width: 200, NotSortable: true},
		{Text: "Description", Width: 450, NotSortable: true},
	}
	a := &characterWalletJournal{
		columnSorter: newColumnSorterWithInit(headers, 0, sortDesc),
		rows:         make([]walletJournalRow, 0),
		top:          makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r walletJournalRow) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(r.date.Format(app.DateTimeFormat))
		case 1:
			return iwidget.NewRichTextSegmentFromText(r.refTypeDisplay)
		case 2:
			return r.amountDisplay
		case 3:
			return iwidget.NewRichTextSegmentFromText(
				humanize.FormatFloat(app.FloatFormat, r.balance),
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				},
			)
		case 4:
			return iwidget.NewRichTextSegmentFromText(r.descriptionWithReason())
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	showReasonDialog := func(r walletJournalRow) {
		if r.hasReason() {
			a.u.ShowInformationDialog("Reason", r.reason, a.u.MainWindow())
		}
	}
	if a.u.isDesktop {
		a.body = makeDataTable(headers, &a.rowsFiltered, makeCell, a.columnSorter, a.filterRows, func(_ int, r walletJournalRow) {
			showReasonDialog(r)
		})
	} else {
		a.body = makeDataList(headers, &a.rowsFiltered, makeCell, showReasonDialog)
	}
	a.selectType = iwidget.NewFilterChipSelectWithSearch("Type", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)
	return a
}

func (a *characterWalletJournal) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectType)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewHScroll(filter),
		nil,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterWalletJournal) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectType.Selected; x != "" {
		rows = xslices.Filter(rows, func(r walletJournalRow) bool {
			return r.refTypeDisplay == x
		})
	}
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b walletJournalRow) int {
			var x int
			switch sortCol {
			case 0:
				x = a.date.Compare(b.date)
			case 1:
				x = strings.Compare(a.refType, b.refType)
			case 2:
				x = cmp.Compare(a.amount, b.amount)
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	// update filters
	a.selectType.SetOptions(xslices.Map(rows, func(r walletJournalRow) string {
		return r.refTypeDisplay
	}))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *characterWalletJournal) update() {
	var err error
	rows := make([]walletJournalRow, 0)
	characterID := a.u.currentCharacterID()
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionWalletJournal)
	if hasData {
		rows2, err2 := a.fetchRows(characterID, a.u.services())
		if err2 != nil {
			slog.Error("Failed to refresh wallet journal UI", "err", err2)
			err = err2
		} else {
			rows = rows2
		}
	}
	t, i := a.u.makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		character := a.u.currentCharacter()
		b := ihumanize.OptionalFloat(character.WalletBalance, 1, "?")
		s := fmt.Sprintf("Balance: %s", b)
		if a.OnUpdate != nil {
			a.OnUpdate(b)
		}
		return s, widget.MediumImportance
	})
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRows(-1)
	})
}

func (*characterWalletJournal) fetchRows(characterID int32, s services) ([]walletJournalRow, error) {
	entries, err := s.cs.ListWalletJournalEntries(context.Background(), characterID)
	if err != nil {
		return nil, err
	}
	rows := make([]walletJournalRow, len(entries))
	for i, o := range entries {
		r := walletJournalRow{
			amount:         o.Amount,
			balance:        o.Balance,
			date:           o.Date,
			description:    o.Description,
			reason:         o.Reason,
			refType:        o.RefType,
			refTypeDisplay: o.RefTypeDisplay(),
		}
		var color fyne.ThemeColorName
		switch {
		case o.Amount < 0:
			color = theme.ColorNameError
		case o.Amount > 0:
			color = theme.ColorNameSuccess
		default:
			color = theme.ColorNameForeground
		}
		r.amountDisplay = iwidget.NewRichTextSegmentFromText(
			humanize.FormatFloat(app.FloatFormat, o.Amount),
			widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: color,
			},
		)
		rows[i] = r
	}
	return rows, nil
}

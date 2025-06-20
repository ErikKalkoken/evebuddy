package ui

import (
	"cmp"
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/antihax/goesi"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type walletJournalRow struct {
	amount           float64
	amountDisplay    []widget.RichTextSegment
	amountFormatted  string
	balance          float64
	balanceFormatted string
	characterID      int32
	date             time.Time
	dateFormatted    string
	description      string
	reason           string
	refID            int64
	refType          string
	refTypeDisplay   string
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
	selectType   *kxwidget.FilterChipSelect
	sortButton   *sortButton
	top          *widget.Label
	u            *baseUI
}

func newCharacterWalletJournal(u *baseUI) *characterWalletJournal {
	headers := []headerDef{
		{Label: "Date", Width: 150},
		{Label: "Type", Width: 150},
		{Label: "Amount", Width: 200},
		{Label: "Balance", Width: 200, NotSortable: true},
		{Label: "Description", Width: 450, NotSortable: true},
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
			return iwidget.NewRichTextSegmentFromText(r.dateFormatted)
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
	if a.u.isDesktop {
		a.body = makeDataTable(headers, &a.rowsFiltered, makeCell, a.columnSorter, a.filterRows, func(_ int, r walletJournalRow) {
			showCharacterWalletJournalEntry(a.u, r.characterID, r.refID)
		})
	} else {
		a.body = a.makeDataList()
	}
	a.selectType = kxwidget.NewFilterChipSelectWithSearch("Type", []string{}, func(string) {
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

func (a *characterWalletJournal) makeDataList() *widget.List {
	p := theme.Padding()
	bgColor := theme.Color(theme.ColorNameInputBackground)
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			date := widget.NewLabel("Template")
			date.Truncation = fyne.TextTruncateClip
			balance := widget.NewLabel("Template")
			balance.Alignment = fyne.TextAlignTrailing
			refType := widget.NewLabel("Template")
			refType.Truncation = fyne.TextTruncateClip
			value := widget.NewLabel("Template")
			value.Alignment = fyne.TextAlignTrailing
			description := widget.NewLabel("Template")
			description.Truncation = fyne.TextTruncateClip
			return container.NewStack(
				canvas.NewRectangle(color.Transparent),
				container.New(layout.NewCustomPaddedVBoxLayout(-p),
					container.NewBorder(nil, nil, nil, value, date),
					container.NewBorder(nil, nil, nil, balance, refType),
					description,
				))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]

			x := co.(*fyne.Container).Objects

			bg := x[0].(*canvas.Rectangle)
			if id%2 == 0 {
				bg.FillColor = bgColor
			} else {
				bg.FillColor = color.Transparent
			}
			bg.Refresh()

			c := x[1].(*fyne.Container).Objects

			b0 := c[0].(*fyne.Container).Objects
			b0[0].(*widget.Label).SetText(r.dateFormatted)
			amount := b0[1].(*widget.Label)
			amount.Text = r.amountFormatted
			amount.Importance = importanceISKAmount(r.amount)
			amount.Refresh()

			b1 := c[1].(*fyne.Container).Objects
			b1[0].(*widget.Label).SetText(r.refTypeDisplay)
			b1[1].(*widget.Label).SetText(r.balanceFormatted)

			c[2].(*widget.Label).SetText(r.description)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll() // TODO: Show detail window
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		showCharacterWalletJournalEntry(a.u, r.characterID, r.refID)
	}
	l.HideSeparators = true
	return l
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
		b := ihumanize.OptionalWithDecimals(character.WalletBalance, 1, "?")
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
			amount:           o.Amount,
			amountFormatted:  humanize.FormatFloat(app.FloatFormat, o.Amount),
			balance:          o.Balance,
			balanceFormatted: humanize.FormatFloat(app.FloatFormat, o.Balance),
			characterID:      characterID,
			date:             o.Date,
			dateFormatted:    o.Date.Format(app.DateTimeFormat),
			description:      o.Description,
			reason:           o.Reason,
			refID:            o.RefID,
			refType:          o.RefType,
			refTypeDisplay:   o.RefTypeDisplay(),
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
			r.amountFormatted,
			widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: color,
			},
		)
		rows[i] = r
	}
	return rows, nil
}

// showCharacterWalletJournalEntry shows a wallet journal entry for a character in a new window.
func showCharacterWalletJournalEntry(u *baseUI, characterID int32, refID int64) {
	o, err := u.cs.GetWalletJournalEntry(context.Background(), characterID, refID)
	if err != nil {
		u.showErrorDialog("Failed to fetch wallet journal entry", err, u.window)
		return
	}

	f := widget.NewForm()
	f.Orientation = widget.Adaptive

	amount := widget.NewLabel(formatISKAmount(o.Amount))
	amount.Importance = importanceISKAmount(o.Amount)
	reason := o.Reason
	if reason == "" {
		reason = "-"
	}

	contextDefaultWidget := widget.NewLabel("?")
	contextItem := widget.NewFormItem("Related item", contextDefaultWidget)
	ctx := context.Background()
	reportError := func(o *app.CharacterWalletJournalEntry, err error) {
		slog.Error("Failed to fetch related context", "contextIDType", o.ContextIDType, "contextID", o.ContextID, "error", err)
		contextDefaultWidget.SetText("Failed to load related item: " + u.humanizeError(err))
	}
	// TODO: Add support for industry jobs
	switch o.ContextIDType {
	case "alliance_id", "character_id", "corporation_id", "system_id", "type_id":
		go func() {
			ee, err := u.eus.GetOrCreateEntityESI(ctx, int32(o.ContextID))
			if err != nil {
				reportError(o, err)
				return
			}
			contextItem.Text = "Related " + ee.CategoryDisplay()
			contextItem.Widget = makeEveEntityActionLabel(ee, u.ShowEveEntityInfoWindow)
			f.Refresh()
		}()
	case "contract_id":
		c, err := u.cs.GetContract(ctx, characterID, int32(o.ContextID))
		if err != nil {
			reportError(o, err)
			break
		}
		contextItem.Text = "Related contract"
		contextItem.Widget = makeLinkLabelWithWrap(c.NameDisplay(), func() {
			showContract(u, c.CharacterID, c.ContractID)
		})
	case "market_transaction_id":
		contextItem.Text = "Related market transaction"
		contextItem.Widget = makeLinkLabelWithWrap(fmt.Sprintf("#%d", o.ContextID), func() {
			showCharacterWalletTransaction(u, o.CharacterID, o.ContextID)
		})
	case "station_id", "structure_id":
		contextItem.Text = "Related location"
		go func() {
			token, err := u.cs.GetValidCharacterToken(ctx, characterID)
			if err != nil {
				reportError(o, err)
				return
			}
			ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
			el, err := u.eus.GetOrCreateLocationESI(ctx, o.ContextID)
			if err != nil {
				reportError(o, err)
				return
			}
			contextItem.Widget = makeLocationLabel(el.ToShort(), u.ShowLocationInfoWindow)
			f.Refresh()
		}()
	}
	items := []*widget.FormItem{
		widget.NewFormItem("Owner", makeOwnerActionLabel(
			characterID,
			u.scs.CharacterName(characterID),
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Date", widget.NewLabel(o.Date.Format(app.DateTimeFormatWithSeconds))),
		widget.NewFormItem("Type", makeLabelWithWrap(o.RefTypeDisplay())),
		widget.NewFormItem("Amount", amount),
		widget.NewFormItem("Balance", widget.NewLabel(formatISKAmount(o.Balance))),
		widget.NewFormItem("Description", makeLabelWithWrap(o.Description)),
		widget.NewFormItem("Reason", makeLabelWithWrap(reason)),
	}
	if o.FirstParty != nil {
		items = append(items, widget.NewFormItem(
			"First Party",
			makeEveEntityActionLabel(o.FirstParty, u.ShowEveEntityInfoWindow),
		))
	}
	if o.SecondParty != nil {
		items = append(items, widget.NewFormItem(
			"Second Party",
			makeEveEntityActionLabel(o.SecondParty, u.ShowEveEntityInfoWindow),
		))
	}
	if o.TaxReceiver != nil {
		items = append(items, widget.NewFormItem(
			"Tax",
			widget.NewLabel(formatISKAmount(o.Tax))),
		)
		items = append(items, widget.NewFormItem(
			"Tax Receiver",
			makeEveEntityActionLabel(o.TaxReceiver, u.ShowEveEntityInfoWindow)),
		)
	}
	items = append(items, contextItem)
	if u.IsDeveloperMode() {
		items = append(items, widget.NewFormItem("Ref ID", u.makeCopyToClipboardLabel(fmt.Sprint(refID))))
	}

	for _, it := range items {
		f.AppendItem(it)
	}
	title := fmt.Sprintf("Wallet Journal #%d", refID)
	w := u.makeDetailWindow("Wallet Journal Entry", title, f)
	w.Show()
}

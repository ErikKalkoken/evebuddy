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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// Options for industry job select widgets
const (
	marketTransactionActivityBuy  = "Buy"
	marketTransactionActivitySell = "Sell"
)

type walletTransactionRow struct {
	categoryName     string
	characterID      int32
	client           *app.EveEntity
	clientName       string
	date             time.Time
	dateFormatted    string
	corporationID    int32
	division         app.Division
	isBuy            bool
	locationDisplay  []widget.RichTextSegment
	locationID       int64
	locationName     string
	quantity         int
	quantityDisplay  string
	regionName       string
	total            float64
	totalColor       fyne.ThemeColorName
	totalFormatted   string
	totalImportance  widget.Importance
	transactionID    int64
	typeID           int32
	typeName         string
	unitPrice        float64
	unitPriceDisplay string
}

type walletTransactions struct {
	widget.BaseWidget

	body           fyne.CanvasObject
	bottom         *widget.Label
	columnSorter   *columnSorter
	division       app.Division
	rows           []walletTransactionRow
	rowsFiltered   []walletTransactionRow
	selectActivity *kxwidget.FilterChipSelect
	selectCategory *kxwidget.FilterChipSelect
	selectClient   *kxwidget.FilterChipSelect
	selectLocation *kxwidget.FilterChipSelect
	selectRegion   *kxwidget.FilterChipSelect
	selectType     *kxwidget.FilterChipSelect
	sortButton     *sortButton
	u              *baseUI
}

func newCharacterWalletTransaction(u *baseUI) *walletTransactions {
	return newWalletTransaction(u, app.DivisionZero)
}

func newCorporationWalletTransactions(u *baseUI, d app.Division) *walletTransactions {
	return newWalletTransaction(u, d)
}

func newWalletTransaction(u *baseUI, d app.Division) *walletTransactions {
	headers := []headerDef{
		{label: "Date", width: columnWidthDateTime},
		{label: "Qty.", width: 75},
		{label: "Type", width: 200},
		{label: "Unit Price", width: 150},
		{label: "Total", width: 150},
		{label: "Client", width: columnWidthEntity},
		{label: "Where", width: columnWidthLocation},
	}
	a := &walletTransactions{
		bottom:       widget.NewLabel(""),
		columnSorter: newColumnSorterWithInit(headers, 0, sortDesc),
		division:     d,
		rows:         make([]walletTransactionRow, 0),
		rowsFiltered: make([]walletTransactionRow, 0),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r walletTransactionRow) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.RichTextSegmentsFromText(r.dateFormatted)
		case 1:
			return iwidget.RichTextSegmentsFromText(r.quantityDisplay,
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				})
		case 2:
			return iwidget.RichTextSegmentsFromText(r.typeName)
		case 3:
			return iwidget.RichTextSegmentsFromText(
				r.unitPriceDisplay,
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				})
		case 4:
			return iwidget.RichTextSegmentsFromText(r.totalFormatted, widget.RichTextStyle{
				ColorName: r.totalColor,
			})
		case 5:
			return iwidget.RichTextSegmentsFromText(r.clientName)
		case 6:
			return r.locationDisplay
		}
		return iwidget.RichTextSegmentsFromText("?")
	}
	if a.u.isDesktop {
		a.body = makeDataTable(
			headers,
			&a.rowsFiltered,
			makeCell,
			a.columnSorter,
			a.filterRows,
			func(_ int, r walletTransactionRow) {
				if a.isCorporation() {
					showCorporationWalletTransactionWindow(a.u, r.characterID, r.division, r.transactionID)
				} else {
					showCharacterWalletTransactionWindow(a.u, r.characterID, r.transactionID)
				}
			})
	} else {
		a.body = a.makeDataList()
	}

	a.selectActivity = kxwidget.NewFilterChipSelect("Activity", []string{
		marketTransactionActivityBuy,
		marketTransactionActivitySell,
	}, func(_ string) {
		a.filterRows(-1)
	})
	a.selectCategory = kxwidget.NewFilterChipSelectWithSearch("Category", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectClient = kxwidget.NewFilterChipSelectWithSearch(
		"Client",
		[]string{},
		func(_ string) {
			a.filterRows(-1)
		},
		a.u.window,
	)
	a.selectLocation = kxwidget.NewFilterChipSelectWithSearch(
		"Location",
		[]string{},
		func(_ string) {
			a.filterRows(-1)
		},
		a.u.window,
	)
	a.selectType = kxwidget.NewFilterChipSelectWithSearch(
		"Type",
		[]string{},
		func(_ string) {
			a.filterRows(-1)
		},
		a.u.window,
	)
	a.selectRegion = kxwidget.NewFilterChipSelectWithSearch("Region",
		[]string{}, func(string) {
			a.filterRows(-1)
		},
		a.u.window,
	)
	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)
	return a
}

func (a *walletTransactions) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectActivity, a.selectCategory, a.selectType, a.selectClient, a.selectRegion, a.selectLocation)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewHScroll(filter),
		a.bottom,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *walletTransactions) isCorporation() bool {
	return a.division != app.DivisionZero
}

func (a *walletTransactions) makeDataList() *iwidget.StripedList {
	p := theme.Padding()
	l := iwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			date := widget.NewLabel("Template")
			date.Truncation = fyne.TextTruncateClip
			total := widget.NewLabel("Template")
			total.Alignment = fyne.TextAlignTrailing
			invType := widget.NewLabel("Template")
			invType.Truncation = fyne.TextTruncateClip
			amount := widget.NewLabel("Template")
			amount.Alignment = fyne.TextAlignTrailing
			location := widget.NewLabel("Template")
			location.Truncation = fyne.TextTruncateClip
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				container.NewBorder(nil, nil, nil, amount, date),
				container.NewBorder(nil, nil, nil, total, invType),
				location,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			c := co.(*fyne.Container).Objects

			b0 := c[0].(*fyne.Container).Objects
			b0[0].(*widget.Label).SetText(r.dateFormatted)
			total := b0[1].(*widget.Label)
			total.Text = r.totalFormatted
			total.Importance = r.totalImportance
			total.Refresh()

			b1 := c[1].(*fyne.Container).Objects
			b1[0].(*widget.Label).SetText(r.typeName)
			b1[1].(*widget.Label).SetText("x" + r.quantityDisplay)

			c[2].(*widget.Label).SetText(r.locationName)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		if a.isCorporation() {
			showCorporationWalletTransactionWindow(a.u, r.characterID, r.division, r.transactionID)
		} else {
			showCharacterWalletTransactionWindow(a.u, r.characterID, r.transactionID)
		}
	}
	l.HideSeparators = true
	return l
}

func (a *walletTransactions) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectActivity.Selected; x != "" {
		rows = xslices.Filter(rows, func(r walletTransactionRow) bool {
			switch x {
			case marketTransactionActivityBuy:
				return r.isBuy
			case marketTransactionActivitySell:
				return !r.isBuy
			}
			return false
		})
	}
	if x := a.selectCategory.Selected; x != "" {
		rows = xslices.Filter(rows, func(r walletTransactionRow) bool {
			return r.categoryName == x
		})
	}
	if x := a.selectClient.Selected; x != "" {
		rows = xslices.Filter(rows, func(r walletTransactionRow) bool {
			return r.clientName == x
		})
	}
	if x := a.selectLocation.Selected; x != "" {
		rows = xslices.Filter(rows, func(r walletTransactionRow) bool {
			return r.locationName == x
		})
	}
	if x := a.selectRegion.Selected; x != "" {
		rows = xslices.Filter(rows, func(r walletTransactionRow) bool {
			return r.regionName == x
		})
	}
	if x := a.selectType.Selected; x != "" {
		rows = xslices.Filter(rows, func(r walletTransactionRow) bool {
			return r.typeName == x
		})
	}
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b walletTransactionRow) int {
			var x int
			switch sortCol {
			case 0:
				x = a.date.Compare(b.date)
			case 1:
				x = cmp.Compare(a.quantity, b.quantity)
			case 2:
				x = strings.Compare(a.typeName, b.typeName)
			case 3:
				x = cmp.Compare(a.unitPrice, b.unitPrice)
			case 4:
				x = cmp.Compare(a.total, b.total)
			case 5:
				x = strings.Compare(a.clientName, b.clientName)
			case 6:
				x = strings.Compare(a.locationName, b.locationName)
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	// update filters
	a.selectCategory.SetOptions(xslices.Map(rows, func(r walletTransactionRow) string {
		return r.categoryName
	}))
	a.selectClient.SetOptions(xslices.Map(rows, func(r walletTransactionRow) string {
		return r.clientName
	}))
	a.selectLocation.SetOptions(xslices.Map(rows, func(r walletTransactionRow) string {
		return r.locationName
	}))
	a.selectRegion.SetOptions(xslices.Map(rows, func(r walletTransactionRow) string {
		return r.regionName
	}))
	a.selectType.SetOptions(xslices.Map(rows, func(r walletTransactionRow) string {
		return r.typeName
	}))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *walletTransactions) update() {
	if a.isCorporation() {
		a.updateCorporation()
	} else {
		a.updateCharacter()
	}
}

func (a *walletTransactions) updateCharacter() {
	var err error
	rows := make([]walletTransactionRow, 0)
	characterID := a.u.currentCharacterID()
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterWalletTransactions)
	if hasData {
		rows2, err2 := a.fetchCharacterRows(characterID, a.u.services())
		if err2 != nil {
			slog.Error("Failed to refresh wallet transaction UI", "err", err2)
			err = err2
		} else {
			rows = rows2
		}
	}
	t, i := a.u.makeTopText(characterID, hasData, err, nil)
	fyne.Do(func() {
		if t != "" {
			a.bottom.Text = t
			a.bottom.Importance = i
			a.bottom.Refresh()
			a.bottom.Show()
		} else {
			a.bottom.Hide()
		}
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRows(-1)
	})
}

func (a *walletTransactions) fetchCharacterRows(characterID int32, s services) ([]walletTransactionRow, error) {
	entries, err := s.cs.ListWalletTransactions(context.Background(), characterID)
	if err != nil {
		return nil, err
	}
	rows := make([]walletTransactionRow, len(entries))
	for i, o := range entries {
		total := o.Total()
		r := walletTransactionRow{
			categoryName:     o.Type.Group.Category.Name,
			characterID:      characterID,
			client:           o.Client,
			clientName:       o.Client.Name,
			date:             o.Date,
			dateFormatted:    o.Date.Format(app.DateTimeFormat),
			isBuy:            o.IsBuy,
			locationDisplay:  o.Location.DisplayRichText(),
			locationID:       o.Location.ID,
			locationName:     o.Location.DisplayName(),
			quantity:         int(o.Quantity),
			quantityDisplay:  humanize.Comma(int64(o.Quantity)),
			total:            total,
			totalFormatted:   humanize.FormatFloat(app.FloatFormat, total),
			transactionID:    o.TransactionID,
			typeID:           o.Type.ID,
			typeName:         o.Type.Name,
			unitPrice:        o.UnitPrice,
			unitPriceDisplay: humanize.FormatFloat(app.FloatFormat, o.UnitPrice),
		}
		if o.IsBuy {
			r.totalColor = theme.ColorNameError
			r.totalImportance = widget.DangerImportance
		} else {
			r.totalColor = theme.ColorNameSuccess
			r.totalImportance = widget.SuccessImportance
		}
		if o.Region != nil {
			r.regionName = o.Region.Name
		}
		rows[i] = r
	}
	return rows, nil
}

func (a *walletTransactions) updateCorporation() {
	var err error
	rows := make([]walletTransactionRow, 0)
	corporationID := a.u.currentCorporationID()
	hasData := a.u.scs.HasCorporationSection(corporationID, app.CorporationSectionWalletTransactions(a.division))
	if hasData {
		rows2, err2 := a.fetchCorporationRows(corporationID, a.division, a.u.services())
		if err2 != nil {
			slog.Error("Failed to refresh wallet transaction UI", "err", err2)
			err = err2
		} else {
			rows = rows2
		}
	}
	t, i := a.u.makeTopText(corporationID, hasData, err, nil)
	fyne.Do(func() {
		if t != "" {
			a.bottom.Text = t
			a.bottom.Importance = i
			a.bottom.Refresh()
			a.bottom.Show()
		} else {
			a.bottom.Hide()
		}
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRows(-1)
	})
}

func (a *walletTransactions) fetchCorporationRows(corporationID int32, division app.Division, s services) ([]walletTransactionRow, error) {
	entries, err := s.rs.ListWalletTransactions(context.Background(), corporationID, division)
	if err != nil {
		return nil, err
	}
	rows := make([]walletTransactionRow, len(entries))
	for i, o := range entries {
		total := o.Total()
		r := walletTransactionRow{
			categoryName:     o.Type.Group.Category.Name,
			client:           o.Client,
			clientName:       o.Client.Name,
			corporationID:    corporationID,
			date:             o.Date,
			dateFormatted:    o.Date.Format(app.DateTimeFormat),
			division:         division,
			isBuy:            o.IsBuy,
			locationDisplay:  o.Location.DisplayRichText(),
			locationID:       o.Location.ID,
			locationName:     o.Location.DisplayName(),
			quantity:         int(o.Quantity),
			quantityDisplay:  humanize.Comma(int64(o.Quantity)),
			total:            total,
			totalFormatted:   humanize.FormatFloat(app.FloatFormat, total),
			transactionID:    o.TransactionID,
			typeID:           o.Type.ID,
			typeName:         o.Type.Name,
			unitPrice:        o.UnitPrice,
			unitPriceDisplay: humanize.FormatFloat(app.FloatFormat, o.UnitPrice),
		}
		if o.IsBuy {
			r.totalColor = theme.ColorNameError
			r.totalImportance = widget.DangerImportance
		} else {
			r.totalColor = theme.ColorNameSuccess
			r.totalImportance = widget.SuccessImportance
		}
		if o.Region != nil {
			r.regionName = o.Region.Name
		}
		rows[i] = r
	}
	return rows, nil
}

// showCharacterWalletTransactionWindow shows the detail of a character wallet transaction in a window.
func showCharacterWalletTransactionWindow(u *baseUI, characterID int32, transactionID int64) {
	title := fmt.Sprintf("Character Market Transaction #%d", transactionID)
	w, ok := u.getOrCreateWindow(fmt.Sprintf("wallettransaction-%d-%d", characterID, transactionID), title, u.scs.CharacterName(characterID))
	if !ok {
		w.Show()
		return
	}
	o, err := u.cs.GetWalletTransactions(context.Background(), characterID, transactionID)
	if err != nil {
		u.showErrorDialog("Failed to show market transaction", err, u.window)
		return
	}
	var activity string
	total := widget.NewLabel(formatISKAmount(o.Total()))
	if o.IsBuy {
		total.Importance = widget.DangerImportance
		activity = "Buy"
	} else {
		total.Importance = widget.SuccessImportance
		activity = "Sell"
	}
	items := []*widget.FormItem{
		widget.NewFormItem("Owner", makeOwnerActionLabel(
			characterID,
			u.scs.CharacterName(characterID),
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Date", widget.NewLabel(o.Date.Format(app.DateTimeFormatWithSeconds))),
		widget.NewFormItem("Activity", widget.NewLabel(activity)),
		widget.NewFormItem("Quantity", widget.NewLabel(humanize.Comma(int64(o.Quantity)))),
		widget.NewFormItem("Type", makeLinkLabelWithWrap(o.Type.Name, func() {
			u.ShowInfoWindow(app.EveEntityInventoryType, o.Type.ID)
		})),
		widget.NewFormItem("Unit price", widget.NewLabel(formatISKAmount(o.UnitPrice))),
		widget.NewFormItem("Total", total),
		widget.NewFormItem("Client", makeEveEntityActionLabel(o.Client, u.ShowEveEntityInfoWindow)),
		widget.NewFormItem("Location", makeLocationLabel(o.Location, u.ShowLocationInfoWindow)),
		widget.NewFormItem("Related Journal Entry", makeLinkLabelWithWrap(
			fmt.Sprintf("#%d", o.JournalRefID), func() {
				showCharacterWalletJournalEntryWindow(u, characterID, o.JournalRefID)
			},
		)),
	}

	if u.IsDeveloperMode() {
		items = append(items, widget.NewFormItem(
			"Transaction ID",
			u.makeCopyToClipboardLabel(fmt.Sprint(transactionID)),
		))
	}
	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	setDetailWindow(detailWindowParams{
		title:   title,
		content: f,
		window:  w,
	})
	w.Show()
}

// showCorporationWalletTransactionWindow shows the detail of a corporation wallet transaction in a window.
func showCorporationWalletTransactionWindow(u *baseUI, corporationID int32, division app.Division, transactionID int64) {
	title := fmt.Sprintf("Corporation Market Transaction #%d", transactionID)
	w, ok := u.getOrCreateWindow(fmt.Sprintf("wallettransaction-%d-%d", corporationID, transactionID), title, u.scs.CorporationName(corporationID))
	if !ok {
		w.Show()
		return
	}
	o, err := u.rs.GetWalletTransaction(context.Background(), corporationID, division, transactionID)
	if err != nil {
		u.showErrorDialog("Failed to show market transaction", err, u.window)
		return
	}
	totalAmount := o.Total()
	items := []*widget.FormItem{
		widget.NewFormItem("Owner", makeOwnerActionLabel(
			corporationID,
			u.scs.CorporationName(corporationID),
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Date", widget.NewLabel(o.Date.Format(app.DateTimeFormatWithSeconds))),
		widget.NewFormItem("Quantity", widget.NewLabel(humanize.Comma(int64(o.Quantity)))),
		widget.NewFormItem("Type", makeLinkLabelWithWrap(o.Type.Name, func() {
			u.ShowInfoWindow(app.EveEntityInventoryType, o.Type.ID)
		})),
		widget.NewFormItem("Unit price", widget.NewLabel(formatISKAmount(o.UnitPrice))),
		widget.NewFormItem("Total", widget.NewLabel(formatISKAmount(totalAmount))),
		widget.NewFormItem("Client", makeEveEntityActionLabel(o.Client, u.ShowEveEntityInfoWindow)),
		widget.NewFormItem("Location", makeLocationLabel(o.Location, u.ShowLocationInfoWindow)),
		widget.NewFormItem("Related Journal Entry", makeLinkLabelWithWrap(
			fmt.Sprintf("#%d", o.JournalRefID), func() {
				showCorporationWalletJournalEntryWindow(u, corporationID, division, o.JournalRefID)
			},
		)),
	}

	if u.IsDeveloperMode() {
		items = append(items, widget.NewFormItem(
			"Transaction ID",
			u.makeCopyToClipboardLabel(fmt.Sprint(transactionID)),
		))
	}
	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	setDetailWindow(detailWindowParams{
		title:   title,
		content: f,
		window:  w,
	})
	w.Show()
}

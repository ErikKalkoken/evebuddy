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

type corporationWalletTransactionRow struct {
	categoryName     string
	corporationID    int32
	client           *app.EveEntity
	clientName       string
	date             time.Time
	division         app.Division
	dateFormatted    string
	locationDisplay  []widget.RichTextSegment
	locationID       int64
	locationName     string
	unitPrice        float64
	quantity         int
	quantityDisplay  string
	regionName       string
	total            float64
	totalColor       fyne.ThemeColorName
	totalImportance  widget.Importance
	totalFormatted   string
	transactionID    int64
	typeID           int32
	typeName         string
	unitPriceDisplay string
}

type corporationWalletTransactions struct {
	widget.BaseWidget

	body           fyne.CanvasObject
	bottom         *widget.Label
	columnSorter   *columnSorter
	division       app.Division
	rows           []corporationWalletTransactionRow
	rowsFiltered   []corporationWalletTransactionRow
	selectCategory *kxwidget.FilterChipSelect
	selectClient   *kxwidget.FilterChipSelect
	selectLocation *kxwidget.FilterChipSelect
	selectRegion   *kxwidget.FilterChipSelect
	selectType     *kxwidget.FilterChipSelect
	sortButton     *sortButton
	u              *baseUI
}

func newCorporationWalletTransactions(u *baseUI, d app.Division) *corporationWalletTransactions {
	headers := []headerDef{
		{label: "Date", width: columnWidthDateTime},
		{label: "Qty.", width: 75},
		{label: "Type", width: 200},
		{label: "Unit Price", width: 150},
		{label: "Total", width: 150},
		{label: "Client", width: columnWidthEntity},
		{label: "Where", width: columnWidthLocation},
	}
	a := &corporationWalletTransactions{
		columnSorter: newColumnSorterWithInit(headers, 0, sortDesc),
		division:     d,
		rows:         make([]corporationWalletTransactionRow, 0),
		rowsFiltered: make([]corporationWalletTransactionRow, 0),
		bottom:       widget.NewLabel(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r corporationWalletTransactionRow) []widget.RichTextSegment {
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
			return iwidget.RichTextSegmentsFromText(r.totalFormatted)
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
			func(_ int, r corporationWalletTransactionRow) {
				showCorporationWalletTransaction(a.u, r.corporationID, r.division, r.transactionID)
			})
	} else {
		a.body = a.makeDataList()
	}

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

func (a *corporationWalletTransactions) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectCategory, a.selectType, a.selectClient, a.selectRegion, a.selectLocation)
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

func (a *corporationWalletTransactions) makeDataList() *iwidget.StripedList {
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
			b0[1].(*widget.Label).SetText(r.totalFormatted)

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
		showCorporationWalletTransaction(a.u, r.corporationID, r.division, r.transactionID)
	}
	l.HideSeparators = true
	return l
}

func (a *corporationWalletTransactions) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectCategory.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationWalletTransactionRow) bool {
			return r.categoryName == x
		})
	}
	if x := a.selectClient.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationWalletTransactionRow) bool {
			return r.clientName == x
		})
	}
	if x := a.selectLocation.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationWalletTransactionRow) bool {
			return r.locationName == x
		})
	}
	if x := a.selectRegion.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationWalletTransactionRow) bool {
			return r.regionName == x
		})
	}
	if x := a.selectType.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationWalletTransactionRow) bool {
			return r.typeName == x
		})
	}
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b corporationWalletTransactionRow) int {
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
	a.selectCategory.SetOptions(xslices.Map(rows, func(r corporationWalletTransactionRow) string {
		return r.categoryName
	}))
	a.selectClient.SetOptions(xslices.Map(rows, func(r corporationWalletTransactionRow) string {
		return r.clientName
	}))
	a.selectLocation.SetOptions(xslices.Map(rows, func(r corporationWalletTransactionRow) string {
		return r.locationName
	}))
	a.selectRegion.SetOptions(xslices.Map(rows, func(r corporationWalletTransactionRow) string {
		return r.regionName
	}))
	a.selectType.SetOptions(xslices.Map(rows, func(r corporationWalletTransactionRow) string {
		return r.typeName
	}))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *corporationWalletTransactions) update() {
	var err error
	rows := make([]corporationWalletTransactionRow, 0)
	corporationID := a.u.currentCorporationID()
	section := map[app.Division]app.CorporationSection{
		app.Division1: app.SectionCorporationWalletTransactions1,
		app.Division2: app.SectionCorporationWalletTransactions2,
		app.Division3: app.SectionCorporationWalletTransactions3,
		app.Division4: app.SectionCorporationWalletTransactions4,
		app.Division5: app.SectionCorporationWalletTransactions5,
		app.Division6: app.SectionCorporationWalletTransactions6,
		app.Division7: app.SectionCorporationWalletTransactions7,
	}
	hasData := a.u.scs.HasCorporationSection(corporationID, section[a.division])
	if hasData {
		rows2, err2 := a.fetchRows(corporationID, a.division, a.u.services())
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

func (a *corporationWalletTransactions) fetchRows(corporationID int32, division app.Division, s services) ([]corporationWalletTransactionRow, error) {
	entries, err := s.rs.ListWalletTransactions(context.Background(), corporationID, division)
	if err != nil {
		return nil, err
	}
	rows := make([]corporationWalletTransactionRow, len(entries))
	for i, o := range entries {
		total := o.Total()
		r := corporationWalletTransactionRow{
			categoryName:     o.Type.Group.Category.Name,
			corporationID:    corporationID,
			client:           o.Client,
			clientName:       o.Client.Name,
			date:             o.Date,
			division:         division,
			dateFormatted:    o.Date.Format(app.DateTimeFormat),
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
		if o.Region != nil {
			r.regionName = o.Region.Name
		}
		rows[i] = r
	}
	return rows, nil
}

func showCorporationWalletTransaction(u *baseUI, corporationID int32, division app.Division, transactionID int64) {
	title := fmt.Sprintf("Market Transaction #%d", transactionID)
	w, ok := u.getOrCreateWindow(fmt.Sprintf("%d-%d", corporationID, transactionID), title, u.scs.CorporationName(corporationID))
	if !ok {
		w.Show()
		return
	}
	o, err := u.rs.GetWalletTransactions(context.Background(), corporationID, division, transactionID)
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
				showCorporationWalletJournalEntry(u, corporationID, division, o.JournalRefID)
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
	setDetailWindow(title, f, w)
	w.Show()
}

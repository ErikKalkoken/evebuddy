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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

type walletTransactionRow struct {
	categoryName     string
	client           *app.EveEntity
	clientName       string
	date             time.Time
	dateDisplay      string
	locationDisplay  []widget.RichTextSegment
	locationID       int64
	locationName     string
	price            float64
	quantity         int
	quantityDisplay  string
	regionName       string
	total            float64
	totalColor       fyne.ThemeColorName
	totalText        string
	transactionID    int64
	typeID           int32
	typeName         string
	unitPriceDisplay string
}

type characterWalletTransaction struct {
	widget.BaseWidget

	body           fyne.CanvasObject
	bottom         *widget.Label
	columnSorter   *columnSorter
	rows           []walletTransactionRow
	rowsFiltered   []walletTransactionRow
	selectCategory *iwidget.FilterChipSelect
	selectClient   *iwidget.FilterChipSelect
	selectLocation *iwidget.FilterChipSelect
	selectRegion   *iwidget.FilterChipSelect
	selectType     *iwidget.FilterChipSelect
	sortButton     *sortButton
	u              *baseUI
}

func newCharacterWalletTransaction(u *baseUI) *characterWalletTransaction {
	headers := []headerDef{
		{Text: "Date", Width: 150},
		{Text: "Quantity", Width: 130},
		{Text: "Type", Width: 200},
		{Text: "Unit Price", Width: 200},
		{Text: "Total", Width: 200},
		{Text: "Client", Width: 250},
		{Text: "Where", Width: 350},
	}
	a := &characterWalletTransaction{
		columnSorter: newColumnSorterWithInit(headers, 0, sortDesc),
		rows:         make([]walletTransactionRow, 0),
		rowsFiltered: make([]walletTransactionRow, 0),
		bottom:       widget.NewLabel(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r walletTransactionRow) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(r.dateDisplay)
		case 1:
			return iwidget.NewRichTextSegmentFromText(r.quantityDisplay,
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				})
		case 2:
			return iwidget.NewRichTextSegmentFromText(r.typeName)
		case 3:
			return iwidget.NewRichTextSegmentFromText(
				r.unitPriceDisplay,
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				})
		case 4:
			return iwidget.NewRichTextSegmentFromText(r.totalText, widget.RichTextStyle{
				ColorName: r.totalColor,
				Alignment: fyne.TextAlignTrailing,
			})
		case 5:
			return iwidget.NewRichTextSegmentFromText(r.clientName)
		case 6:
			return r.locationDisplay
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	if a.u.isDesktop {
		a.body = makeDataTable(
			headers,
			&a.rowsFiltered,
			makeCell,
			a.columnSorter,
			a.filterRows,
			func(_ int, r walletTransactionRow) {
				a.showEntry(r)
			})
	} else {
		a.body = makeDataList(headers, &a.rowsFiltered, makeCell, a.showEntry)
	}

	a.selectCategory = iwidget.NewFilterChipSelectWithSearch("Category", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectClient = iwidget.NewFilterChipSelectWithSearch(
		"Client",
		[]string{},
		func(_ string) {
			a.filterRows(-1)
		},
		a.u.window,
	)
	a.selectLocation = iwidget.NewFilterChipSelectWithSearch(
		"Location",
		[]string{},
		func(_ string) {
			a.filterRows(-1)
		},
		a.u.window,
	)
	a.selectType = iwidget.NewFilterChipSelectWithSearch(
		"Type",
		[]string{},
		func(_ string) {
			a.filterRows(-1)
		},
		a.u.window,
	)
	a.selectRegion = iwidget.NewFilterChipSelectWithSearch("Region",
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

func (a *characterWalletTransaction) CreateRenderer() fyne.WidgetRenderer {
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

func (a *characterWalletTransaction) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
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
				x = cmp.Compare(a.price, b.price)
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

func (a *characterWalletTransaction) update() {
	var err error
	rows := make([]walletTransactionRow, 0)
	characterID := a.u.currentCharacterID()
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionWalletTransactions)
	if hasData {
		rows2, err2 := a.fetchRows(characterID, a.u.services())
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

func (a *characterWalletTransaction) fetchRows(characterID int32, s services) ([]walletTransactionRow, error) {
	entries, err := s.cs.ListWalletTransactions(context.Background(), characterID)
	if err != nil {
		return nil, err
	}
	rows := make([]walletTransactionRow, len(entries))
	for i, o := range entries {
		total := o.UnitPrice * float64(o.Quantity)
		var color fyne.ThemeColorName
		switch {
		case total < 0:
			color = theme.ColorNameError
		case total > 0:
			color = theme.ColorNameSuccess
		default:
			color = theme.ColorNameForeground
		}
		totalText := humanize.FormatFloat(app.FloatFormat, total)
		totalColor := color
		r := walletTransactionRow{
			categoryName:     o.Type.Group.Category.Name,
			client:           o.Client,
			clientName:       o.Client.Name,
			date:             o.Date,
			dateDisplay:      o.Date.Format(app.DateTimeFormat),
			locationDisplay:  o.Location.DisplayRichText(),
			locationID:       o.Location.ID,
			locationName:     o.Location.DisplayName(),
			price:            o.UnitPrice,
			quantity:         int(o.Quantity),
			quantityDisplay:  humanize.Comma(int64(o.Quantity)),
			total:            total,
			totalColor:       totalColor,
			totalText:        totalText,
			transactionID:    o.TransactionID,
			typeID:           o.Type.ID,
			typeName:         o.Type.Name,
			unitPriceDisplay: humanize.FormatFloat(app.FloatFormat, o.UnitPrice),
		}
		if o.Region != nil {
			r.regionName = o.Region.Name
		}
		rows[i] = r
	}
	return rows, nil
}

func (a *characterWalletTransaction) showEntry(r walletTransactionRow) {
	newTappableLabelWithWrap := func(text string, f func()) *kxwidget.TappableLabel {
		x := kxwidget.NewTappableLabel(text, f)
		x.Wrapping = fyne.TextWrapWord
		return x
	}
	location := iwidget.NewTappableRichText(r.locationDisplay, func() {
		a.u.ShowLocationInfoWindow(r.locationID)
	})
	location.Wrapping = fyne.TextWrapWord
	client := newTappableLabelWithWrap(r.clientName, func() {
		a.u.ShowEveEntityInfoWindow(r.client)
	})
	client.Wrapping = fyne.TextWrapWord
	items := []*widget.FormItem{
		widget.NewFormItem("Date", widget.NewLabel(r.dateDisplay)),
		widget.NewFormItem("Quantity", widget.NewLabel(r.quantityDisplay)),
		widget.NewFormItem("Type", newTappableLabelWithWrap(r.typeName, func() {
			a.u.ShowInfoWindow(app.EveEntityInventoryType, r.typeID)
		})),
		widget.NewFormItem("Unit price", widget.NewLabel(r.unitPriceDisplay)),
		widget.NewFormItem("Total", widget.NewRichText(
			iwidget.NewRichTextSegmentFromText(
				r.totalText, widget.RichTextStyle{
					ColorName: r.totalColor,
				})...,
		)),
		widget.NewFormItem("Client", client),
		widget.NewFormItem("Location", location),
	}

	if a.u.IsDeveloperMode() {
		items = append(items, widget.NewFormItem(
			"Transaction ID",
			a.u.makeCopyToClipboardLabel(fmt.Sprint(r.transactionID)),
		))
	}
	title := fmt.Sprintf("Entry #%d", r.transactionID)
	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	w := a.u.makeDetailWindow("Wallet Transaction Entry", title, f)
	w.Show()
}

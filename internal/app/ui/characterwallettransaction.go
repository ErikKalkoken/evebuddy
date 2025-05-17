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
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

type walletTransactionRow struct {
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
	total            float64
	totalText        string
	totalColor       fyne.ThemeColorName
	transactionID    int64
	typeID           int32
	typeName         string
	unitPriceDisplay string
}

type characterWalletTransaction struct {
	widget.BaseWidget

	body           fyne.CanvasObject
	columnSorter   *columnSorter
	rows           []walletTransactionRow
	rowsFiltered   []walletTransactionRow
	selectClient   *selectFilter
	selectLocation *selectFilter
	sortButton     *sortButton
	top            *widget.Label
	u              *BaseUI
}

func NewCharacterWalletTransaction(u *BaseUI) *characterWalletTransaction {
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
		top:          makeTopLabel(),
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

	a.selectClient = newSelectFilter("Any client", func() {
		a.filterRows(-1)
	})
	a.selectLocation = newSelectFilter("Any location", func() {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)
	return a
}

func (a *characterWalletTransaction) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectClient, a.selectLocation)
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

func (a *characterWalletTransaction) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	a.selectClient.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(r walletTransactionRow) bool {
			return r.clientName == selected
		})
	})
	a.selectLocation.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(r walletTransactionRow) bool {
			return r.locationName == selected
		})
	})
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
	a.selectClient.setOptions(xiter.MapSlice(rows, func(r walletTransactionRow) string {
		return r.clientName
	}))
	a.selectLocation.setOptions(xiter.MapSlice(rows, func(r walletTransactionRow) string {
		return r.locationName
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
	t, i := makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		t := humanize.Comma(int64(len(rows)))
		s := fmt.Sprintf("Entries: %s", t)
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
			client:           o.Client,
			clientName:       o.Client.Name,
			date:             o.Date,
			dateDisplay:      o.Date.Format(app.DateTimeFormat),
			locationDisplay:  o.Location.DisplayRichText(),
			locationID:       o.Location.ID,
			locationName:     o.Location.DisplayName(),
			price:            o.UnitPrice,
			unitPriceDisplay: humanize.FormatFloat(app.FloatFormat, o.UnitPrice),
			quantity:         int(o.Quantity),
			quantityDisplay:  humanize.Comma(int64(o.Quantity)),
			total:            total,
			totalText:        totalText,
			totalColor:       totalColor,
			typeID:           o.EveType.ID,
			typeName:         o.EveType.Name,
			transactionID:    o.TransactionID,
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
	location := iwidget.NewTappableRichText(
		func() {
			a.u.ShowLocationInfoWindow(r.locationID)
		},
		r.locationDisplay...,
	)
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

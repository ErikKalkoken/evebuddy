package wallet

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/xwindow"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// Options for industry job select widgets
const (
	marketTransactionActivityBuy  = "Buy"
	marketTransactionActivitySell = "Sell"
)

type walletTransactionRow struct {
	categoryName     string
	characterID      int64
	client           *app.EveEntity
	clientName       string
	date             time.Time
	dateFormatted    string
	corporationID    int64
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
	typeID           int64
	typeName         string
	unitPrice        float64
	unitPriceDisplay string
}

type WalletTransactions struct {
	widget.BaseWidget

	body           fyne.CanvasObject
	footer         *widget.Label
	character      atomic.Pointer[app.Character]
	columnSorter   *xwidget.ColumnSorter[walletTransactionRow]
	corporation    atomic.Pointer[app.Corporation]
	division       app.Division
	rows           []walletTransactionRow
	rowsFiltered   []walletTransactionRow
	selectActivity *kxwidget.FilterChipSelect
	selectCategory *kxwidget.FilterChipSelect
	selectClient   *kxwidget.FilterChipSelect
	selectLocation *kxwidget.FilterChipSelect
	selectRegion   *kxwidget.FilterChipSelect
	selectType     *kxwidget.FilterChipSelect
	sortButton     *xwidget.SortButton
	u              ui
}

func NewCharacterWalletTransaction(u ui) *WalletTransactions {
	a := newWalletTransaction(u, app.DivisionZero)
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.Update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterWalletTransactions {
			a.Update(ctx)
		}
	})
	return a
}

func NewCorporationWalletTransactions(u ui, d app.Division) *WalletTransactions {
	a := newWalletTransaction(u, d)
	a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		a.Update(ctx)
	})
	a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		if a.corporation.Load().IDOrZero() != arg.CorporationID {
			return
		}
		if arg.Section == app.CorporationSectionWalletTransactions(d) {
			a.Update(ctx)
		}
	})
	return a
}

const (
	walletTransactionColDate = iota + 1
	walletTransactionColQuantity
	walletTransactionColType
	walletTransactionColPrice
	walletTransactionColTotal
	walletTransactionColClient
	walletTransactionColLocation
)

func newWalletTransaction(u ui, d app.Division) *WalletTransactions {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[walletTransactionRow]{{
		ID:    walletTransactionColDate,
		Label: "Date",
		Width: app.ColumnWidthDateTime,
		Sort: func(a, b walletTransactionRow) int {
			return a.date.Compare(b.date)
		},
		Update: func(r walletTransactionRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.dateFormatted)
		},
	}, {
		ID:    walletTransactionColQuantity,
		Label: "Qty.",
		Width: 75,
		Sort: func(a, b walletTransactionRow) int {
			return cmp.Compare(a.quantity, b.quantity)
		},
		Update: func(r walletTransactionRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.quantityDisplay, widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		},
	}, {
		ID:    walletTransactionColType,
		Label: "Type",
		Width: 200,
		Sort: func(a, b walletTransactionRow) int {
			return strings.Compare(a.typeName, b.typeName)
		},
		Update: func(r walletTransactionRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.typeName)
		},
	}, {
		ID:    walletTransactionColPrice,
		Label: "Unit Price",
		Width: 150,
		Sort: func(a, b walletTransactionRow) int {
			return cmp.Compare(a.unitPrice, b.unitPrice)
		},
		Update: func(r walletTransactionRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.unitPriceDisplay, widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		},
	}, {
		ID:    walletTransactionColTotal,
		Label: "Total",
		Width: 150,
		Sort: func(a, b walletTransactionRow) int {
			return cmp.Compare(a.total, b.total)
		},
		Update: func(r walletTransactionRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.totalFormatted, widget.RichTextStyle{
				ColorName: r.totalColor,
			})
		},
	}, {
		ID:    walletTransactionColClient,
		Label: "Client",
		Width: app.ColumnWidthEntity,
		Sort: func(a, b walletTransactionRow) int {
			return xstrings.CompareIgnoreCase(a.clientName, b.clientName)
		},
		Update: func(r walletTransactionRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.clientName)
		},
	}, {
		ID:    walletTransactionColLocation,
		Label: "Where",
		Width: app.ColumnWidthLocation,
		Sort: func(a, b walletTransactionRow) int {
			return strings.Compare(a.locationName, b.locationName)
		},
		Update: func(r walletTransactionRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.locationDisplay)
		},
	}})
	a := &WalletTransactions{
		footer:       awidget.NewLabelWithTruncation(""),
		columnSorter: xwidget.NewColumnSorter(columns, walletTransactionColDate, xwidget.SortDesc),
		division:     d,
		u:            u,
	}
	a.ExtendBaseWidget(a)

	if !a.u.IsMobile() {
		a.body = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync,
			func(_ int, r walletTransactionRow) {
				if a.isCorporation() {
					ShowCorporationWalletTransactionWindowAsync(a.u, r.corporationID, r.division, r.transactionID)
				} else {
					ShowCharacterWalletTransactionWindowAsync(a.u, r.characterID, r.transactionID)
				}
			})
	} else {
		a.body = a.makeDataList()
	}

	a.selectActivity = kxwidget.NewFilterChipSelect("Activity", []string{
		marketTransactionActivityBuy,
		marketTransactionActivitySell,
	}, func(_ string) {
		a.filterRowsAsync(-1)
	})
	a.selectCategory = kxwidget.NewFilterChipSelectWithSearch("Category", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectClient = kxwidget.NewFilterChipSelectWithSearch(
		"Client",
		[]string{},
		func(_ string) {
			a.filterRowsAsync(-1)
		},
		a.u.MainWindow(),
	)
	a.selectLocation = kxwidget.NewFilterChipSelectWithSearch(
		"Location",
		[]string{},
		func(_ string) {
			a.filterRowsAsync(-1)
		},
		a.u.MainWindow(),
	)
	a.selectType = kxwidget.NewFilterChipSelectWithSearch(
		"Type",
		[]string{},
		func(_ string) {
			a.filterRowsAsync(-1)
		},
		a.u.MainWindow(),
	)
	a.selectRegion = kxwidget.NewFilterChipSelectWithSearch("Region",
		[]string{}, func(string) {
			a.filterRowsAsync(-1)
		},
		a.u.MainWindow(),
	)
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	return a
}

func (a *WalletTransactions) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectActivity, a.selectCategory, a.selectType, a.selectClient, a.selectRegion, a.selectLocation)
	if a.u.IsMobile() {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewHScroll(filter),
		a.footer,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *WalletTransactions) isCorporation() bool {
	return a.division != app.DivisionZero
}

func (a *WalletTransactions) makeDataList() *xwidget.StripedList {
	p := theme.Padding()
	l := xwidget.NewStripedList(
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
			ShowCorporationWalletTransactionWindowAsync(a.u, r.corporationID, r.division, r.transactionID)
		} else {
			ShowCharacterWalletTransactionWindowAsync(a.u, r.characterID, r.transactionID)
		}
	}
	l.HideSeparators = true
	return l
}

func (a *WalletTransactions) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	category := a.selectCategory.Selected
	client := a.selectClient.Selected
	location := a.selectLocation.Selected
	region := a.selectRegion.Selected
	et := a.selectType.Selected
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		// filter
		if activity := a.selectActivity.Selected; activity != "" {
			rows = slices.DeleteFunc(rows, func(r walletTransactionRow) bool {
				switch activity {
				case marketTransactionActivityBuy:
					return !r.isBuy
				case marketTransactionActivitySell:
					return r.isBuy
				}
				return true
			})
		}
		if category != "" {
			rows = slices.DeleteFunc(rows, func(r walletTransactionRow) bool {
				return r.categoryName != category
			})
		}
		if client != "" {
			rows = slices.DeleteFunc(rows, func(r walletTransactionRow) bool {
				return r.clientName != client
			})
		}
		if location != "" {
			rows = slices.DeleteFunc(rows, func(r walletTransactionRow) bool {
				return r.locationName != location
			})
		}
		if region != "" {
			rows = slices.DeleteFunc(rows, func(r walletTransactionRow) bool {
				return r.regionName != region
			})
		}
		if et != "" {
			rows = slices.DeleteFunc(rows, func(r walletTransactionRow) bool {
				return r.typeName != et
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		// update filters
		categoryOptions := xslices.Map(rows, func(r walletTransactionRow) string {
			return r.categoryName
		})
		clientOptions := xslices.Map(rows, func(r walletTransactionRow) string {
			return r.clientName
		})
		locationOPtions := xslices.Map(rows, func(r walletTransactionRow) string {
			return r.locationName
		})
		regionOptions := xslices.Map(rows, func(r walletTransactionRow) string {
			return r.regionName
		})
		typeOptions := xslices.Map(rows, func(r walletTransactionRow) string {
			return r.typeName
		})
		footer := fmt.Sprintf("Showing %s / %s transactions", ihumanize.Comma(len(rows)), ihumanize.Comma(totalRows))

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectCategory.SetOptions(categoryOptions)
			a.selectClient.SetOptions(clientOptions)
			a.selectLocation.SetOptions(locationOPtions)
			a.selectRegion.SetOptions(regionOptions)
			a.selectType.SetOptions(typeOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	}()
}

func (a *WalletTransactions) Update(ctx context.Context) {
	if a.isCorporation() {
		a.updateCorporation(ctx)
	} else {
		a.updateCharacter(ctx)
	}
}

func (a *WalletTransactions) updateCharacter(ctx context.Context) {
	var err error
	var rows []walletTransactionRow
	characterID := a.character.Load().IDOrZero()
	hasData := a.u.StatusCache().HasCharacterSection(characterID, app.SectionCharacterWalletTransactions)
	if hasData {
		rows2, err2 := a.fetchCharacterRows(ctx, characterID)
		if err2 != nil {
			slog.Error("Failed to refresh wallet transaction UI", "err", err2)
			err = err2
		} else {
			rows = rows2
		}
	}
	t, i := makeTopText(characterID, hasData, err, nil)
	fyne.Do(func() {
		if t != "" {
			a.footer.Text = t
			a.footer.Importance = i
			a.footer.Refresh()
		}
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync(-1)
	})
}

func (a *WalletTransactions) fetchCharacterRows(ctx context.Context, characterID int64) ([]walletTransactionRow, error) {
	entries, err := a.u.Character().ListWalletTransactions(ctx, characterID)
	if err != nil {
		return nil, err
	}
	var rows []walletTransactionRow
	for _, o := range entries {
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
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *WalletTransactions) updateCorporation(ctx context.Context) {
	var err error
	var rows []walletTransactionRow
	corporationID := a.corporation.Load().IDOrZero()
	hasData := a.u.StatusCache().HasCorporationSection(corporationID, app.CorporationSectionWalletTransactions(a.division))
	if hasData {
		rows2, err2 := a.fetchCorporationRows(ctx, corporationID, a.division)
		if err2 != nil {
			slog.Error("Failed to refresh wallet transaction UI", "err", err2)
			err = err2
		} else {
			rows = rows2
		}
	}
	t, i := makeTopText(corporationID, hasData, err, nil)
	fyne.Do(func() {
		if t != "" {
			a.footer.Text = t
			a.footer.Importance = i
			a.footer.Refresh()
			a.footer.Show()
		} else {
			a.footer.Hide()
		}
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync(-1)
	})
}

func (a *WalletTransactions) fetchCorporationRows(ctx context.Context, corporationID int64, division app.Division) ([]walletTransactionRow, error) {
	entries, err := a.u.Corporation().ListWalletTransactions(ctx, corporationID, division)
	if err != nil {
		return nil, err
	}
	var rows []walletTransactionRow
	for _, o := range entries {
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
		rows = append(rows, r)
	}
	return rows, nil
}

// ShowCharacterWalletTransactionWindowAsync shows the detail of a character wallet transaction in a window.
func ShowCharacterWalletTransactionWindowAsync(u ui, characterID int64, transactionID int64) {
	title := fmt.Sprintf("Character Market Transaction #%d", transactionID)
	w, created := u.GetOrCreateWindow(
		fmt.Sprintf("wallettransaction-%d-%d", characterID, transactionID),
		title,
		u.StatusCache().CharacterName(characterID),
	)
	if !created {
		w.Show()
		return
	}

	go func() {
		o, err := u.Character().GetWalletTransactions(context.Background(), characterID, transactionID)
		if err != nil {
			xdialog.ShowErrorAndLog("Failed to show market transaction", err, u.IsDeveloperMode(), u.MainWindow())
			return
		}
		fyne.Do(func() {
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
				widget.NewFormItem("Owner", makeCharacterActionLabel(
					characterID,
					u.StatusCache().CharacterName(characterID),
					u.InfoWindow().Show,
				)),
				widget.NewFormItem("Date", widget.NewLabel(o.Date.Format(app.DateTimeFormatWithSeconds))),
				widget.NewFormItem("Activity", widget.NewLabel(activity)),
				widget.NewFormItem("Quantity", widget.NewLabel(humanize.Comma(int64(o.Quantity)))),
				widget.NewFormItem("Type", makeLinkLabelWithWrap(o.Type.Name, func() {
					u.InfoWindow().Show(o.Type.EveEntity())
				})),
				widget.NewFormItem("Unit price", widget.NewLabel(formatISKAmount(o.UnitPrice))),
				widget.NewFormItem("Total", total),
				widget.NewFormItem("Client", makeEveEntityActionLabel(o.Client, u.InfoWindow().Show)),
				widget.NewFormItem("Location", makeLocationLabel(o.Location, u.InfoWindow().ShowLocation)),
				// widget.NewFormItem("Related Journal Entry", makeLinkLabelWithWrap(
				// 	fmt.Sprintf("#%d", o.JournalRefID), func() {
				// 		showCharacterWalletJournalEntryWindow(u, characterID, o.JournalRefID)
				// 	},
				// )),
			}

			if u.IsDeveloperMode() {
				items = append(items, widget.NewFormItem(
					"Transaction ID",
					xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(transactionID)),
				))
			}
			f := widget.NewForm(items...)
			f.Orientation = widget.Adaptive
			xwindow.Set(xwindow.Params{
				Content: f,
				ImageAction: func() {
					u.InfoWindow().ShowType(o.Type.ID, 0)
				},
				ImageLoader: func(setter func(r fyne.Resource)) {
					u.EVEImage().InventoryTypeIconAsync(o.Type.ID, 256, setter)
				},
				Title:  title,
				Window: w,
			})
			w.Show()
		})
	}()
}

// ShowCorporationWalletTransactionWindowAsync shows the detail of a corporation wallet transaction in a window.
func ShowCorporationWalletTransactionWindowAsync(u ui, corporationID int64, division app.Division, transactionID int64) {
	title := fmt.Sprintf("Corporation Market Transaction #%d", transactionID)
	w, created := u.GetOrCreateWindow(
		fmt.Sprintf("wallettransaction-%d-%d", corporationID, transactionID),
		title,
		u.StatusCache().CorporationName(corporationID),
	)
	if !created {
		w.Show()
		return
	}

	go func() {
		o, err := u.Corporation().GetWalletTransaction(context.Background(), corporationID, division, transactionID)
		if err != nil {
			xdialog.ShowErrorAndLog("Failed to show market transaction", err, u.IsDeveloperMode(), u.MainWindow())
			return
		}
		fyne.Do(func() {
			totalAmount := o.Total()
			items := []*widget.FormItem{
				widget.NewFormItem("Owner", makeCharacterActionLabel(
					corporationID,
					u.StatusCache().CorporationName(corporationID),
					u.InfoWindow().Show,
				)),
				widget.NewFormItem("Date", widget.NewLabel(o.Date.Format(app.DateTimeFormatWithSeconds))),
				widget.NewFormItem("Quantity", widget.NewLabel(humanize.Comma(int64(o.Quantity)))),
				widget.NewFormItem("Type", makeLinkLabelWithWrap(o.Type.Name, func() {
					u.InfoWindow().Show(o.Type.EveEntity())
				})),
				widget.NewFormItem("Unit price", widget.NewLabel(formatISKAmount(o.UnitPrice))),
				widget.NewFormItem("Total", widget.NewLabel(formatISKAmount(totalAmount))),
				widget.NewFormItem("Client", makeEveEntityActionLabel(o.Client, u.InfoWindow().Show)),
				widget.NewFormItem("Location", makeLocationLabel(o.Location, u.InfoWindow().ShowLocation)),
				widget.NewFormItem("Related Journal Entry", makeLinkLabelWithWrap(
					fmt.Sprintf("#%d", o.JournalRefID), func() {
						go ShowCorporationWalletJournalEntryWindowAsync(u, corporationID, division, o.JournalRefID)
					},
				)),
			}

			if u.IsDeveloperMode() {
				items = append(items, widget.NewFormItem(
					"Transaction ID",
					xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(transactionID)),
				))
			}
			f := widget.NewForm(items...)
			f.Orientation = widget.Adaptive
			xwindow.Set(xwindow.Params{
				Content: f,
				ImageAction: func() {
					u.InfoWindow().ShowType(o.Type.ID, 0)
				},
				ImageLoader: func(setter func(r fyne.Resource)) {
					u.EVEImage().InventoryTypeIconAsync(o.Type.ID, 256, setter)
				},
				Title:  title,
				Window: w,
			})
			w.Show()
		})
	}()
}

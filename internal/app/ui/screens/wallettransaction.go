package screens

import (
	"cmp"
	"context"
	"fmt"
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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// Options for industry job select widgets
const (
	marketTransactionActivityBuy  = "Buy"
	marketTransactionActivitySell = "Sell"
)

func walletTransactionActivityOptions() []string {
	return []string{marketTransactionActivityBuy, marketTransactionActivitySell}
}

// Names of the wallet transaction filters, used as labels on desktop and as option names on mobile.
const (
	walletTransactionFilterActivity = "Activity"
	walletTransactionFilterCategory = "Category"
	walletTransactionFilterClient   = "Client"
	walletTransactionFilterDivision = "Division"
	walletTransactionFilterLocation = "Location"
	walletTransactionFilterRegion   = "Region"
	walletTransactionFilterType     = "Type"
)

// walletTransactionFilter is the selected value of each wallet transaction filter. Empty means not filtered.
type walletTransactionFilter struct {
	activity string
	category string
	client   string
	division string
	location string
	region   string
	typeName string
}

// match reports whether row r passes all filters.
func (f walletTransactionFilter) match(r walletTransactionRow) bool {
	switch {
	case f.activity == marketTransactionActivityBuy && !r.isBuy,
		f.activity == marketTransactionActivitySell && r.isBuy,
		f.category != "" && r.categoryName != f.category,
		f.client != "" && r.clientName != f.client,
		f.division != "" && r.divisionName != f.division,
		f.location != "" && r.locationName != f.location,
		f.region != "" && r.regionName != f.region,
		f.typeName != "" && r.typeName != f.typeName:
		return false
	}
	return true
}

type walletTransactionRow struct {
	categoryName     string
	ownerID          int64
	ownerName        string
	client           *app.EveEntity
	clientName       string
	date             time.Time
	dateFormatted    string
	division         app.Division
	divisionName     string
	isBuy            bool
	locationDisplay  []widget.RichTextSegment
	locationID       int64
	locationName     string
	quantity         int
	quantityDisplay  string
	regionName       string
	searchTarget     string
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

// setDisplayFields sets the fields derived from the row's data.
func (r *walletTransactionRow) setDisplayFields() {
	r.dateFormatted = r.date.Format(app.DateTimeFormat)
	r.quantityDisplay = humanize.Comma(int64(r.quantity))
	r.totalFormatted = humanize.FormatFloat(ui.FloatFormatISK, r.total)
	r.unitPriceDisplay = humanize.FormatFloat(ui.FloatFormatISK, r.unitPrice)
	if r.isBuy {
		r.totalColor = theme.ColorNameError
		r.totalImportance = widget.DangerImportance
	} else {
		r.totalColor = theme.ColorNameSuccess
		r.totalImportance = widget.SuccessImportance
	}
	r.setSearchTarget()
}

// setSearchTarget sets the text the search entry matches against.
func (r *walletTransactionRow) setSearchTarget() {
	r.searchTarget = strings.ToLower(r.clientName + "\n" + r.typeName) // separator prevents matches across names
}

type WalletTransactions struct {
	widget.BaseWidget

	activitySelect *kxwidget.FilterChipSelect // select chips only on desktop
	body           fyne.CanvasObject
	categorySelect *kxwidget.FilterChipSelect
	character      atomic.Pointer[app.Character]
	clientSelect   *kxwidget.FilterChipSelect
	columnSorter   *xwidget.ColumnSorter[walletTransactionRow]
	corporation    atomic.Pointer[app.Corporation]
	divisionSelect *kxwidget.FilterChipSelect
	errorLabel     *widget.Label
	filterChip     *xwidget.FilterChipCompact // only on mobile
	filterRun      xsync.LatestRun
	footerLabel    *widget.Label
	forCorporation bool
	locationSelect *kxwidget.FilterChipSelect
	regionSelect   *kxwidget.FilterChipSelect
	rows           []walletTransactionRow
	rowsFiltered   []walletTransactionRow
	searchEntry    *xwidget.SearchEntry
	sortChip       *kxwidget.SortChip
	typeSelect     *kxwidget.FilterChipSelect
	u              baseUI
	updateRun      xsync.LatestRun
}

func NewCharacterWalletTransaction(u baseUI) *WalletTransactions {
	return newWalletTransaction(u, false)
}

func NewCorporationWalletTransactions(u baseUI) *WalletTransactions {
	return newWalletTransaction(u, true)
}

func newWalletTransaction(u baseUI, forCorporation bool) *WalletTransactions {
	cols := []xwidget.DataColumn[walletTransactionRow]{{
		Label: "Date",
		Width: ui.ColumnWidthDateTime,
		Sort: func(a, b walletTransactionRow) int {
			return a.date.Compare(b.date)
		},
		Update: func(r walletTransactionRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.dateFormatted)
		},
	}, {
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
		Label: "Type",
		Width: 200,
		Sort: func(a, b walletTransactionRow) int {
			return strings.Compare(a.typeName, b.typeName)
		},
		Update: func(r walletTransactionRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.typeName)
		},
	}, {
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
		Label: "Total",
		Width: 150,
		Sort: func(a, b walletTransactionRow) int {
			return cmp.Compare(a.total, b.total)
		},
		Update: func(r walletTransactionRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.totalFormatted, widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: r.totalColor,
			})
		},
	}, {
		Label: "Client",
		Width: ui.ColumnWidthEntity,
		Sort: func(a, b walletTransactionRow) int {
			return xstrings.CompareIgnoreCase(a.clientName, b.clientName)
		},
		Update: func(r walletTransactionRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.clientName)
		},
	}, {
		Label: "Where",
		Width: ui.ColumnWidthLocation,
		Sort: func(a, b walletTransactionRow) int {
			return strings.Compare(a.locationName, b.locationName)
		},
		Update: func(r walletTransactionRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.locationDisplay)
		},
	}}
	if forCorporation {
		cols = append(cols, xwidget.DataColumn[walletTransactionRow]{
			Label: "Division",
			Width: ui.ColumnWidthEntity,
			Sort: func(a, b walletTransactionRow) int {
				return cmp.Compare(a.division, b.division)
			},
			Update: func(r walletTransactionRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.divisionName)
			},
		})
	}
	columns := xwidget.NewDataColumns(cols)
	a := &WalletTransactions{
		columnSorter:   xwidget.NewColumnSorter(columns, "Date", xwidget.SortDesc),
		errorLabel:     ui.NewLabelWithWrapping(""),
		footerLabel:    ui.NewLabelWithTruncation(""),
		forCorporation: forCorporation,
		u:              u,
	}
	a.ExtendBaseWidget(a)

	a.errorLabel.Importance = widget.DangerImportance
	a.errorLabel.Hide()

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
				if a.forCorporation {
					ShowCorporationWalletTransactionWindowAsync(a.u, r.ownerID, r.ownerName, r.division, r.transactionID)
				} else {
					ShowCharacterWalletTransactionWindowAsync(a.u, r.ownerID, r.ownerName, r.transactionID)
				}
			})
	} else {
		a.body = a.makeDataList()
	}

	placeholder := "Search clients and types"
	if a.u.IsMobile() {
		placeholder = "Search" // shares the row with the chips
	}
	a.searchEntry = xwidget.NewSearchEntry(placeholder, func(_ string) {
		a.filterRowsAsync("")
	})

	if a.u.IsMobile() {
		a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
			a.filterRowsAsync("")
		})
	} else {
		makeSelectWithSearch := func(label string) *kxwidget.FilterChipSelect {
			return kxwidget.NewFilterChipSelectWithSearch(label, []string{}, func(string) {
				a.filterRowsAsync("")
			}, a.u.MainWindow())
		}
		a.activitySelect = kxwidget.NewFilterChipSelect(walletTransactionFilterActivity, walletTransactionActivityOptions(), func(_ string) {
			a.filterRowsAsync("")
		})
		a.activitySelect.SortDisabled = true
		a.categorySelect = makeSelectWithSearch(walletTransactionFilterCategory)
		a.clientSelect = makeSelectWithSearch(walletTransactionFilterClient)
		a.locationSelect = makeSelectWithSearch(walletTransactionFilterLocation)
		a.typeSelect = makeSelectWithSearch(walletTransactionFilterType)
		a.regionSelect = makeSelectWithSearch(walletTransactionFilterRegion)
		a.divisionSelect = kxwidget.NewFilterChipSelect(walletTransactionFilterDivision, []string{}, func(_ string) {
			a.filterRowsAsync("")
		})
	}
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	})

	// signals
	if a.forCorporation {
		a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			fyne.Do(a.resetFilters)
			a.updateCorporationRows(ctx)
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if a.corporation.Load().IDOrZero() != arg.CorporationID {
				return
			}
			if arg.Section == app.SectionCorporationDivisions || app.CorporationSectionsWalletTransactions.Contains(arg.Section) {
				a.updateCorporationRows(ctx)
			}
		})
	} else {
		a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
			a.character.Store(c)
			fyne.Do(a.resetFilters)
			a.updateCharacter(ctx)
		})
		a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
			if a.character.Load().IDOrZero() != arg.CharacterID {
				return
			}
			if arg.Section == app.SectionCharacterWalletTransactions {
				a.updateCharacter(ctx)
			}
		})
	}
	return a
}

func (a *WalletTransactions) CreateRenderer() fyne.WidgetRenderer {
	var top fyne.CanvasObject
	if a.u.IsMobile() {
		controls := container.NewHBox(a.filterChip, a.sortChip)
		top = container.NewBorder(
			nil,
			nil,
			nil,
			controls,
			a.searchEntry,
		)
	} else {
		filter := make([]fyne.CanvasObject, 0, 8)
		if a.forCorporation {
			filter = append(filter, a.divisionSelect)
		}
		filter = append(
			filter,
			a.activitySelect,
			a.categorySelect,
			a.typeSelect,
			a.clientSelect,
			a.regionSelect,
			a.locationSelect,
		)
		top = container.NewBorder(
			nil,
			nil,
			container.NewHBox(filter...),
			nil,
			a.searchEntry,
		)
	}
	c := container.NewBorder(
		top,
		container.NewVBox(a.errorLabel, a.footerLabel),
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chips on desktop.
func (a *WalletTransactions) currentFilter() walletTransactionFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		return walletTransactionFilter{
			activity: s[walletTransactionFilterActivity],
			category: s[walletTransactionFilterCategory],
			client:   s[walletTransactionFilterClient],
			division: s[walletTransactionFilterDivision],
			location: s[walletTransactionFilterLocation],
			region:   s[walletTransactionFilterRegion],
			typeName: s[walletTransactionFilterType],
		}
	}
	return walletTransactionFilter{
		activity: a.activitySelect.Selected,
		category: a.categorySelect.Selected,
		client:   a.clientSelect.Selected,
		division: a.divisionSelect.Selected,
		location: a.locationSelect.Selected,
		region:   a.regionSelect.Selected,
		typeName: a.typeSelect.Selected,
	}
}

// resetFilters clears search and filters without filtering again.
func (a *WalletTransactions) resetFilters() {
	a.searchEntry.ClearSilent()
	if a.filterChip != nil {
		a.filterChip.ResetSilent()
		return
	}
	clearSelectsSilent(
		a.activitySelect,
		a.categorySelect,
		a.clientSelect,
		a.divisionSelect,
		a.locationSelect,
		a.regionSelect,
		a.typeSelect,
	)
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
			division := widget.NewLabel("Template")
			if !a.forCorporation {
				division.Hide()
			}
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				container.NewBorder(nil, nil, nil, amount, date),
				container.NewBorder(nil, nil, nil, total, invType),
				container.NewBorder(nil, nil, nil, division, location),
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

			b2 := c[2].(*fyne.Container).Objects
			b2[0].(*widget.Label).SetText(r.locationName)
			b2[1].(*widget.Label).SetText(r.divisionName)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		if a.forCorporation {
			ShowCorporationWalletTransactionWindowAsync(a.u, r.ownerID, r.ownerName, r.division, r.transactionID)
		} else {
			ShowCharacterWalletTransactionWindowAsync(a.u, r.ownerID, r.ownerName, r.transactionID)
		}
	}
	l.HideSeparators = true
	return l
}

func (a *WalletTransactions) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.Start()
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	filter := a.currentFilter()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
		rows = slices.DeleteFunc(rows, func(r walletTransactionRow) bool {
			return !filter.match(r)
		})
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r walletTransactionRow) bool {
				return !strings.Contains(r.searchTarget, search)
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
		locationOptions := xslices.Map(rows, func(r walletTransactionRow) string {
			return r.locationName
		})
		regionOptions := xslices.Map(rows, func(r walletTransactionRow) string {
			return r.regionName
		})
		typeOptions := xslices.Map(rows, func(r walletTransactionRow) string {
			return r.typeName
		})
		divisionOptions := xslices.Map(rows, func(r walletTransactionRow) string {
			return r.divisionName
		})
		footer := fmt.Sprintf("Showing %s / %s transactions", ihumanize.Comma(len(rows)), ihumanize.Comma(totalRows))

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footerLabel.Text = footer
			a.footerLabel.Importance = widget.MediumImportance
			a.footerLabel.Refresh()
			if a.filterChip != nil {
				options := make([]xwidget.FilterOption, 0, 8)
				if a.forCorporation {
					options = append(
						options,
						xwidget.NewFilterOptionMultiChoice(walletTransactionFilterDivision, divisionOptions),
					)
				}
				options = append(
					options,
					xwidget.NewFilterOptionMultiChoiceOrdered(walletTransactionFilterActivity, walletTransactionActivityOptions()),
					xwidget.NewFilterOptionMultiChoiceWithSearch(walletTransactionFilterCategory, categoryOptions),
					xwidget.NewFilterOptionMultiChoiceWithSearch(walletTransactionFilterType, typeOptions),
					xwidget.NewFilterOptionMultiChoiceWithSearch(walletTransactionFilterClient, clientOptions),
					xwidget.NewFilterOptionMultiChoiceWithSearch(walletTransactionFilterRegion, regionOptions),
					xwidget.NewFilterOptionMultiChoiceWithSearch(walletTransactionFilterLocation, locationOptions),
				)
				a.filterChip.SetOptions(options...)
			} else {
				a.divisionSelect.SetOptions(divisionOptions)
				a.categorySelect.SetOptions(categoryOptions)
				a.clientSelect.SetOptions(clientOptions)
				a.locationSelect.SetOptions(locationOptions)
				a.regionSelect.SetOptions(regionOptions)
				a.typeSelect.SetOptions(typeOptions)
			}
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	})
}

func (a *WalletTransactions) updateCharacter(ctx context.Context) {
	a.updateRows(ctx, func(ctx context.Context) ([]walletTransactionRow, bool, bool, error) {
		c := a.character.Load() // read after Start, so the latest run has the current owner
		if c == nil {
			return nil, false, false, nil
		}
		rows, hasData, err := a.fetchCharacterRows(ctx, c)
		return rows, true, hasData, err
	})
}

func (a *WalletTransactions) updateCorporationRows(ctx context.Context) {
	a.updateRows(ctx, func(ctx context.Context) ([]walletTransactionRow, bool, bool, error) {
		c := a.corporation.Load() // read after Start, so the latest run has the current owner
		if c == nil {
			return nil, false, false, nil
		}
		rows, hasData, err := a.fetchCorporationRows(ctx, c)
		return rows, true, hasData, err
	})
}

func (a *WalletTransactions) updateRows(ctx context.Context, fetch func(context.Context) (rows []walletTransactionRow, hasOwner, hasData bool, err error)) {
	isLatest := a.updateRun.Start()
	setInfo := func(s string, i widget.Importance) {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footerLabel.Text, a.footerLabel.Importance = s, i
			a.footerLabel.Refresh()
		})
	}
	setError := func(err error) {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.errorLabel.SetText("ERROR: " + a.u.ErrorDisplay(err))
			a.errorLabel.Show()
			a.Refresh()
		})
	}
	reset := func() {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.filterRun.Discard()
			a.errorLabel.Hide()
			xslices.Clear(&a.rows)
			xslices.Clear(&a.rowsFiltered)
			a.footerLabel.SetText("")
			a.body.Refresh()
		})
	}
	rows, hasOwner, hasData, err := fetch(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		reset()
		setError(err)
		return
	}
	if !hasOwner {
		reset()
		var s string
		if a.forCorporation {
			s = "No corporation"
		} else {
			s = "No character"
		}
		setInfo(s, widget.LowImportance)
		return
	}
	if !hasData {
		reset()
		setInfo("No data", widget.WarningImportance)
		return
	}
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.errorLabel.Hide()
		a.rows = rows
		a.filterRowsAsync("")
	})
}

func (a *WalletTransactions) fetchCharacterRows(ctx context.Context, character *app.Character) ([]walletTransactionRow, bool, error) {
	hasData, err := a.u.Character().HasSection(ctx, character.ID, app.SectionCharacterWalletTransactions)
	if err != nil || !hasData {
		return nil, false, err
	}
	oo, err := a.u.Character().ListWalletTransactions(ctx, character.ID)
	if err != nil {
		return nil, false, err
	}
	rows := make([]walletTransactionRow, 0, len(oo))
	for _, o := range oo {
		r := walletTransactionRow{
			categoryName:    o.Type.Group.Category.Name,
			ownerID:         character.ID,
			ownerName:       character.NameOrZero(),
			client:          o.Client,
			clientName:      o.Client.Name,
			date:            o.Date,
			isBuy:           o.IsBuy,
			locationDisplay: o.Location.DisplayRichText(),
			locationID:      o.Location.ID,
			locationName:    o.Location.DisplayName(),
			quantity:        int(o.Quantity),
			total:           o.Total(),
			transactionID:   o.TransactionID,
			typeID:          o.Type.ID,
			typeName:        o.Type.Name,
			unitPrice:       o.UnitPrice,
		}
		if o.Region != nil {
			r.regionName = o.Region.Name
		}
		r.setDisplayFields()
		rows = append(rows, r)
	}
	return rows, true, nil
}

func (a *WalletTransactions) fetchCorporationRows(ctx context.Context, corporation *app.Corporation) ([]walletTransactionRow, bool, error) {
	completed, err := a.u.Corporation().CompletedSections(ctx, corporation.ID)
	if err != nil || !completed.ContainsAny(app.CorporationSectionsWalletTransactions.All()) {
		return nil, false, err
	}
	names := a.u.Corporation().ListWalletNames(ctx, corporation.ID)
	oo, err := a.u.Corporation().ListWalletTransactionsAll(ctx, corporation.ID)
	if err != nil {
		return nil, false, err
	}
	rows := make([]walletTransactionRow, 0, len(oo))
	for _, o := range oo {
		r := walletTransactionRow{
			categoryName:    o.Type.Group.Category.Name,
			client:          o.Client,
			clientName:      o.Client.Name,
			ownerID:         corporation.ID,
			ownerName:       corporation.NameOrZero(),
			date:            o.Date,
			division:        o.Division,
			divisionName:    corporationWalletLabel(o.Division, names[o.Division]),
			isBuy:           o.IsBuy,
			locationDisplay: o.Location.DisplayRichText(),
			locationID:      o.Location.ID,
			locationName:    o.Location.DisplayName(),
			quantity:        int(o.Quantity),
			total:           o.Total(),
			transactionID:   o.TransactionID,
			typeID:          o.Type.ID,
			typeName:        o.Type.Name,
			unitPrice:       o.UnitPrice,
		}
		if o.Region != nil {
			r.regionName = o.Region.Name
		}
		r.setDisplayFields()
		rows = append(rows, r)
	}
	return rows, true, nil
}

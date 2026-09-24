package screens

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"
	"github.com/nathabonfim59/fyneline"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	wealthMaxCharacters        = 10
	wealthMinSliceCount        = 1 // not included others
	wealthMinSliceShare        = 0.05
	wealthMultiplier           = 1_000_000_000
	wealthNameTruncationLimit  = 20
	wealthNameTruncationSuffix = 0
)

// characterWealthRow is the shared data for all tabs of the wealth screen.
type characterWealthRow struct {
	alliance        optional.Optional[*app.EveEntity]
	characterID     int64
	characterName   string
	combinedAssets  optional.Optional[float64]
	contractsEscrow optional.Optional[float64]
	corporation     *app.EveEntity
	ordersEscrow    optional.Optional[float64]
	skillPoints     optional.Optional[float64]
	tags            set.Set[string]
	total           optional.Optional[float64]
	walletBalance   optional.Optional[float64]
}

func (r characterWealthRow) allianceName() string {
	return optional.Map(r.alliance, "", func(v *app.EveEntity) string {
		return v.Name
	})
}

func (r characterWealthRow) corporationName() string {
	if r.corporation == nil {
		return ""
	}
	return r.corporation.Name
}

func (r characterWealthRow) chartName() string {
	return xstrings.TruncateWithSuffix(r.characterName, wealthNameTruncationLimit, wealthNameTruncationSuffix)
}

// filterWealthRows returns the rows matching all non-empty filter values.
func filterWealthRows(rows []characterWealthRow, tag, corporation, alliance string) []characterWealthRow {
	return slices.DeleteFunc(slices.Clone(rows), func(r characterWealthRow) bool {
		return (tag != "" && !r.tags.Contains(tag)) ||
			(corporation != "" && r.corporationName() != corporation) ||
			(alliance != "" && r.allianceName() != alliance)
	})
}

// wealthFilterOptions returns the sorted options for the filter chips.
func wealthFilterOptions(rows []characterWealthRow) (tags, corporations, alliances []string) {
	sortedNonEmpty := func(s set.Set[string]) []string {
		s.Delete("")
		return slices.Sorted(s.All())
	}
	tags = sortedNonEmpty(set.Union(xslices.Map(rows, func(r characterWealthRow) set.Set[string] {
		return r.tags
	})...))
	corporations = sortedNonEmpty(set.Of(xslices.Map(rows, characterWealthRow.corporationName)...))
	alliances = sortedNonEmpty(set.Of(xslices.Map(rows, characterWealthRow.allianceName)...))
	return tags, corporations, alliances
}

// wealthBillions returns v in billion ISK for the charts.
func wealthBillions(v optional.Optional[float64]) float64 {
	return v.ValueOrZero() / wealthMultiplier
}

// characterWealthValue holds one character's wealth breakdown.
type characterWealthValue struct {
	name      string
	assets    float64
	wallet    float64
	contracts float64
	orders    float64
}

type CharacterWealth struct {
	widget.BaseWidget

	OnUpdate func(totalNetWorth optional.Optional[float64])

	breakdownEmpty               *widget.Label
	characterBreakdownCard       *chartCard
	characterBreakdownChart      *fyneline.BarChart[characterWealthValue]
	characterBreakdownTitleLabel *widget.Label
	characterSplitCard           *chartCard
	characterSplitChart          *fyneline.ArcChart[namedValue]
	characterSplitTitleLabel     *widget.Label
	details                      *characterWealthDetails
	filterRun                    latestRun
	footer                       *widget.Label
	overviewEmpty                *widget.Label
	overviewGrid                 *fyne.Container
	rows                         []characterWealthRow
	selectAlliance               *kxwidget.FilterChipSelect
	selectCorporation            *kxwidget.FilterChipSelect
	selectTag                    *kxwidget.FilterChipSelect
	showHelp                     *xwidget.IconButton
	topLabel                     *widget.Label
	totalSplitCard               *chartCard
	totalSplitChart              *fyneline.ArcChart[namedValue]
	totalSplitTitleLabel         *widget.Label
	u                            baseUI
}

func NewCharacterWealth(u baseUI) *CharacterWealth {
	sliceLabel := func(v namedValue) string { return fmt.Sprintf("%.1f", v.value) }
	a := &CharacterWealth{
		characterBreakdownChart: fyneline.NewBarChart([]characterWealthValue(nil),
			func(v characterWealthValue) string { return v.name },
			fyneline.NewBarSeries("Assets", func(v characterWealthValue) float64 { return v.assets }),
			fyneline.NewBarSeries("Wallet", func(v characterWealthValue) float64 { return v.wallet }),
			fyneline.NewBarSeries("Contracts", func(v characterWealthValue) float64 { return v.contracts }),
			fyneline.NewBarSeries("Orders", func(v characterWealthValue) float64 { return v.orders }),
		),
		characterBreakdownTitleLabel: newChartTitleLabel(),
		characterSplitChart: fyneline.NewArcChart([]namedValue(nil),
			func(v namedValue) float64 { return v.value },
			sliceLabel,
		),
		characterSplitTitleLabel: newChartTitleLabel(),
		footer:                   widget.NewLabel(""),
		topLabel:                 ui.NewLabelWithWrapping(""),
		totalSplitChart: fyneline.NewArcChart([]namedValue(nil),
			func(v namedValue) float64 { return v.value },
			sliceLabel,
		),
		totalSplitTitleLabel: newChartTitleLabel(),
		u:                    u,
		details:              newCharacterWealthDetails(u),
	}
	a.ExtendBaseWidget(a)
	a.topLabel.Hide()

	a.characterBreakdownChart.SetValueAxis(fyneline.NewNumericAxis().WithFormatter(wealthAxisValueFormatter))
	a.characterBreakdownChart.SetOrientation(fyneline.BarHorizontal)
	a.characterBreakdownChart.SetSeriesLayout(fyneline.SeriesStack)
	configureArcChart(a.characterSplitChart)
	configureArcChart(a.totalSplitChart)

	legend := newSeriesLegend(
		newLegendEntry("Assets", wealthArcColor(0)),
		newLegendEntry("Wallet", wealthArcColor(1)),
		newLegendEntry("Contracts", wealthArcColor(2)),
		newLegendEntry("Orders", wealthArcColor(3)),
	)

	totalLegend := newSeriesLegend(
		newLegendEntry("Assets", wealthArcColor(0)),
		newLegendEntry("Wallet", wealthArcColor(1)),
		newLegendEntry("Contracts", wealthArcColor(2)),
		newLegendEntry("Orders", wealthArcColor(3)),
	)

	a.characterBreakdownCard = newChartCard(a.characterBreakdownTitleLabel, legend, a.characterBreakdownChart)
	a.characterSplitCard = newChartCard(a.characterSplitTitleLabel, newSeriesLegend(), a.characterSplitChart)
	a.totalSplitCard = newChartCard(a.totalSplitTitleLabel, totalLegend, a.totalSplitChart)
	a.overviewGrid = container.NewAdaptiveGrid(2, a.totalSplitCard, a.characterSplitCard)

	newEmptyLabel := func() *widget.Label {
		l := widget.NewLabel("")
		l.Importance = widget.LowImportance
		l.Hide()
		return l
	}
	a.overviewEmpty = newEmptyLabel()
	a.breakdownEmpty = newEmptyLabel()

	a.selectAlliance = kxwidget.NewFilterChipSelect("Alliance", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.selectCorporation = kxwidget.NewFilterChipSelect("Corporation", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync()
	})

	a.showHelp = xwidget.NewIconButton(theme.QuestionIcon(), func() {
		showHelpPopUp(characterWealthHelpText, a.u.IsMobile(), a.showHelp)
	})
	a.showHelp.SetToolTip("Show explanation for values")

	// Signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		switch arg.Section {
		case
			app.SectionCharacterAssets,
			app.SectionCharacterContracts,
			app.SectionCharacterMarketOrders,
			app.SectionCharacterSkills,
			app.SectionCharacterWalletBalance:
			a.update(ctx)
		}
	})
	a.u.Signals().CharacterChanged.AddListener(func(ctx context.Context, _ int64) {
		a.update(ctx)
	})
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	return a
}

func (a *CharacterWealth) CreateRenderer() fyne.WidgetRenderer {
	tabs := container.NewAppTabs(
		container.NewTabItem(
			"Overview",
			container.NewStack(a.overviewGrid, container.NewCenter(a.overviewEmpty)),
		),
		container.NewTabItem(
			"Characters",
			container.NewStack(a.characterBreakdownCard, container.NewCenter(a.breakdownEmpty)),
		),
		container.NewTabItem("Details", a.details),
	)
	filterBar := container.NewHScroll(container.NewHBox(a.selectCorporation, a.selectAlliance, a.selectTag))
	var top fyne.CanvasObject
	if !a.u.IsMobile() {
		top = container.NewVBox(a.topLabel, filterBar)
	} else {
		top = filterBar
	}
	c := container.NewBorder(
		top,
		container.NewHBox(a.footer, layout.NewSpacer(), a.showHelp),
		nil,
		nil,
		tabs,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterWealth) update(ctx context.Context) {
	rows, total, err := a.fetchRows(ctx)
	if err != nil {
		slog.Error("Failed to fetch data for wealth", "err", err)
		fyne.Do(func() {
			a.topLabel.Text = fmt.Sprintf("Failed to fetch data for charts: %s", a.u.ErrorDisplay(err))
			a.topLabel.Importance = widget.DangerImportance
			a.topLabel.Refresh()
			a.topLabel.Show()
			a.footer.Text = "ERROR: " + a.u.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.topLabel.Hide()
		a.rows = rows
		a.filterRowsAsync()
	})

	// Only report when there is wealth data to show.
	hasWealth := slices.ContainsFunc(rows, func(r characterWealthRow) bool {
		return !r.total.IsEmpty()
	})
	if !hasWealth {
		return
	}
	fyne.Do(func() {
		if a.OnUpdate != nil {
			a.OnUpdate(total)
		}
	})
}

// filterRowsAsync applies the filters and updates all tabs.
// Must be called on the main thread.
func (a *CharacterWealth) filterRowsAsync() {
	isLatest := a.filterRun.start()
	rows := slices.Clone(a.rows)
	alliance := a.selectAlliance.Selected
	corporation := a.selectCorporation.Selected
	tag := a.selectTag.Selected
	isFiltered := alliance != "" || corporation != "" || tag != ""

	runAsync(func() {
		filtered := filterWealthRows(rows, tag, corporation, alliance)
		tagOptions, corporationOptions, allianceOptions := wealthFilterOptions(filtered)
		footer := fmt.Sprintf("Showing %d / %d characters", len(filtered), len(rows))

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectAlliance.SetOptions(allianceOptions)
			a.selectCorporation.SetOptions(corporationOptions)
			a.selectTag.SetOptions(tagOptions)
			a.details.setRows(filtered)
		})

		// Characters without any wealth data are shown in details only.
		chartRows := slices.DeleteFunc(slices.Clone(filtered), func(r characterWealthRow) bool {
			return r.total.IsEmpty()
		})
		if text := wealthEmptyText(isFiltered, len(filtered), len(chartRows)); text != "" {
			fyne.Do(func() {
				if !isLatest() {
					return
				}
				a.setChartsEmpty(text)
			})
			return
		}

		if !isLatest() {
			return // skip chart updates for outdated results
		}
		a.updateCharacterBreakdown(chartRows)
		a.updateCharacterSplit(chartRows)
		a.updateTotalSplit(chartRows)
		// Queued after the chart updates, so no stale data is shown.
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.setChartsEmpty("")
		})
	})
}

// setChartsEmpty replaces the charts with text, or shows them again when text is empty.
// Must be called on the main thread.
func (a *CharacterWealth) setChartsEmpty(text string) {
	for _, l := range []*widget.Label{a.overviewEmpty, a.breakdownEmpty} {
		l.SetText(text)
		l.Hidden = text == ""
		l.Refresh()
	}
	if text == "" {
		a.overviewGrid.Show()
		a.characterBreakdownCard.Show()
	} else {
		a.overviewGrid.Hide()
		a.characterBreakdownCard.Hide()
	}
}

// wealthEmptyText returns the text to show instead of the charts
// or an empty string when there is data to chart.
func wealthEmptyText(isFiltered bool, filteredCount, chartCount int) string {
	switch {
	case chartCount > 0:
		return ""
	case filteredCount == 0 && isFiltered:
		return "No characters match the filter"
	case filteredCount == 0:
		return "No characters"
	default:
		return "No wealth data yet"
	}
}

func (a *CharacterWealth) updateCharacterBreakdown(rows []characterWealthRow) {
	var total float64
	d := make([]characterWealthValue, 0, len(rows))
	for _, r := range rows {
		d = append(d, characterWealthValue{
			name:      r.chartName(),
			assets:    wealthBillions(r.combinedAssets),
			wallet:    wealthBillions(r.walletBalance),
			contracts: wealthBillions(r.contractsEscrow),
			orders:    wealthBillions(r.ordersEscrow),
		})
		total += wealthBillions(r.total)
	}
	d = reduceCharacterWealthValues(d, wealthMaxCharacters)

	var maxValue float64
	for _, v := range d {
		maxValue = max(maxValue, v.assets+v.wallet+v.contracts+v.orders)
	}
	axisMax, tickCount := niceAxisBounds(maxValue, 5)

	fyne.Do(func() {
		a.characterBreakdownChart.SetValueAxis(fyneline.NewNumericAxis().
			WithFormatter(wealthAxisValueFormatter).
			WithDomain(0, axisMax).
			WithTickCount(tickCount))
		a.characterBreakdownChart.SetData(d)
		a.characterBreakdownTitleLabel.SetText(fmt.Sprintf("Wealth Breakdown by Character - Total: %.1f B", total))
	})
}

func (a *CharacterWealth) updateCharacterSplit(rows []characterWealthRow) {
	var total float64
	d := make([]namedValue, 0, len(rows))
	for _, r := range rows {
		v := wealthBillions(r.total)
		d = append(d, namedValue{name: r.chartName(), value: v})
		total += v
	}
	d = reduceSliceValues(d, wealthMinSliceShare, wealthMinSliceCount)

	entries := make([]*legendEntry, len(d))
	for i, v := range d {
		entries[i] = newLegendEntry(v.name, wealthArcColor(i))
	}

	fyne.Do(func() {
		a.characterSplitChart.SetData(d)
		a.characterSplitCard.legend.SetEntries(entries...)
		a.characterSplitTitleLabel.SetText(fmt.Sprintf("Total Net Worth By Character - Total: %.1f B", total))
	})
}

func (a *CharacterWealth) updateTotalSplit(rows []characterWealthRow) {
	var assets, wallets, contracts, orders, total float64
	for _, r := range rows {
		assets += wealthBillions(r.combinedAssets)
		contracts += wealthBillions(r.contractsEscrow)
		orders += wealthBillions(r.ordersEscrow)
		total += wealthBillions(r.total)
		wallets += wealthBillions(r.walletBalance)
	}
	d := []namedValue{
		{name: "Assets", value: assets},
		{name: "Wallet", value: wallets},
		{name: "Contracts", value: contracts},
		{name: "Orders", value: orders},
	}

	fyne.Do(func() {
		a.totalSplitChart.SetData(d)
		title := fmt.Sprintf("Total Net Worth By Category - Total: %.1f B", total)
		a.totalSplitTitleLabel.SetText(title)
	})
}

// fetchRows returns the rows for all characters sorted by name and the grand total.
func (a *CharacterWealth) fetchRows(ctx context.Context) ([]characterWealthRow, optional.Optional[float64], error) {
	cc, err := a.u.Character().ListCharacters(ctx)
	if err != nil {
		return nil, optional.Optional[float64]{}, err
	}
	rows := make([]characterWealthRow, 0, len(cc))
	totals := make([]optional.Optional[float64], 0, len(cc))
	for _, c := range cc {
		tags, err := a.u.Character().ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return nil, optional.Optional[float64]{}, err
		}
		combinedAssets := c.CombinedAssetsValue()
		total := optional.Sum(c.WalletBalance, combinedAssets, c.ContractsEscrow, c.OrdersEscrow)
		totals = append(totals, total)
		rows = append(rows, characterWealthRow{
			alliance:        c.EveCharacter.Alliance,
			characterID:     c.ID,
			corporation:     c.EveCharacter.Corporation,
			characterName:   c.EveCharacter.Name,
			combinedAssets:  combinedAssets,
			contractsEscrow: c.ContractsEscrow,
			ordersEscrow:    c.OrdersEscrow,
			skillPoints:     c.SkillPointsValue,
			tags:            tags,
			total:           total,
			walletBalance:   c.WalletBalance,
		})
	}
	slices.SortFunc(rows, func(a, b characterWealthRow) int {
		return strings.Compare(a.characterName, b.characterName)
	})
	return rows, optional.Sum(totals...), nil
}

// characterWealthDetailsRow represents a row that is shown in the character wealth details table.
type characterWealthDetailsRow struct {
	characterID            int64
	characterName          string
	combinedAssetsDisplay  string
	combinedAssetsValue    optional.Optional[float64]
	contractsEscrow        optional.Optional[float64]
	contractsEscrowDisplay string
	isTotal                bool
	ordersEscrow           optional.Optional[float64]
	ordersEscrowDisplay    string
	searchTarget           string
	skillPoints            optional.Optional[float64]
	skillPointsDisplay     string
	tagsDisplay            string
	totalNetWorth          optional.Optional[float64]
	totalNetWorthDisplay   string
	walletBalance          optional.Optional[float64]
	walletDisplay          string
}

func newWealthDetailsRow(r characterWealthRow) characterWealthDetailsRow {
	return characterWealthDetailsRow{
		characterID:            r.characterID,
		characterName:          r.characterName,
		combinedAssetsDisplay:  formatISKValue(r.combinedAssets),
		combinedAssetsValue:    r.combinedAssets,
		contractsEscrow:        r.contractsEscrow,
		contractsEscrowDisplay: formatISKValue(r.contractsEscrow),
		ordersEscrow:           r.ordersEscrow,
		ordersEscrowDisplay:    formatISKValue(r.ordersEscrow),
		searchTarget:           strings.ToLower(r.characterName),
		skillPoints:            r.skillPoints,
		skillPointsDisplay:     formatISKValue(r.skillPoints),
		tagsDisplay:            strings.Join(slices.Sorted(r.tags.All()), ", "),
		totalNetWorth:          r.total,
		totalNetWorthDisplay:   formatISKValue(r.total),
		walletBalance:          r.walletBalance,
		walletDisplay:          formatISKValue(r.walletBalance),
	}
}

func (r characterWealthDetailsRow) eveEntity() *app.EveEntity {
	return &app.EveEntity{
		Category: app.EveEntityCharacter,
		ID:       r.characterID,
		Name:     r.characterName,
	}
}

// characterWealthDetails is the details tab of the wealth screen.
// Its data is provided by [CharacterWealth].
type characterWealthDetails struct {
	widget.BaseWidget

	columnSorter *xwidget.ColumnSorter[characterWealthDetailsRow]
	filterRun    latestRun
	main         fyne.CanvasObject
	rows         []characterWealthDetailsRow
	rowsFiltered []characterWealthDetailsRow
	searchEntry  *xwidget.SearchEntry
	sortChip     *kxwidget.SortChip
	u            baseUI
}

const characterWealthDetailsValueWidth = 125

const characterWealthHelpText = `Wallet Balance: The balance of the wallet.

Combined Assets: The estimated value of all personal assets, items in outstanding sell orders on the market and items in outstanding contracts.

Contract Escrow: The total sum of all escrows in a character's currently outstanding contracts.

Orders Escrow: The total sum of all escrows in a character's currently outstanding buy orders on the market.

Total Net Worth: Sum of Wallet Balance, Combined Assets, Contract Escrow and Orders Escrow.

Skill Points: Value of extracted skill points (trained + unallocated) calculated with: (MarketPriceOfLargeSkillInjector − MarketPriceOfSkillExtractor) x ((TotalSP − 5,000,000) / 500,000)

NOTE: Blueprints, PLEX in the account wallet are not included.`

func newCharacterWealthDetails(u baseUI) *characterWealthDetails {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[characterWealthDetailsRow]{
		ui.MakeEveEntityColumn(ui.MakeEveEntityColumnParams[characterWealthDetailsRow]{
			EIS: u.EVEImage(),
			GetEntity: func(r characterWealthDetailsRow) *app.EveEntity {
				return &app.EveEntity{
					ID:       r.characterID,
					Name:     r.characterName,
					Category: app.EveEntityCharacter,
				}
			},
			IsAvatar: true,
			Label:    "Character",
		}), {
			Label: "Tags",
			Width: 150,
			Update: func(r characterWealthDetailsRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.tagsDisplay)
			},
		}, {
			Label: "Combined Assets",
			Width: characterWealthDetailsValueWidth,
			Update: func(r characterWealthDetailsRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.combinedAssetsDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b characterWealthDetailsRow) int {
				return optional.Compare(a.combinedAssetsValue, b.combinedAssetsValue)
			},
		}, {
			Label: "Wallet Balance",
			Width: characterWealthDetailsValueWidth,
			Update: func(r characterWealthDetailsRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.walletDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b characterWealthDetailsRow) int {
				return optional.Compare(a.walletBalance, b.walletBalance)
			},
		}, {
			Label: "Contracts Escrow",
			Width: characterWealthDetailsValueWidth,
			Update: func(r characterWealthDetailsRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.contractsEscrowDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b characterWealthDetailsRow) int {
				return optional.Compare(a.contractsEscrow, b.contractsEscrow)
			},
		},
		{
			Label: "Orders Escrow",
			Width: characterWealthDetailsValueWidth,
			Update: func(r characterWealthDetailsRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.ordersEscrowDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b characterWealthDetailsRow) int {
				return optional.Compare(a.ordersEscrow, b.ordersEscrow)
			},
		}, {
			Label: "Total Net Worth",
			Width: characterWealthDetailsValueWidth,
			Update: func(r characterWealthDetailsRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.totalNetWorthDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b characterWealthDetailsRow) int {
				return optional.Compare(a.totalNetWorth, b.totalNetWorth)
			},
		}, {
			Label: "Skill Points",
			Width: characterWealthDetailsValueWidth,
			Update: func(r characterWealthDetailsRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.skillPointsDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b characterWealthDetailsRow) int {
				return optional.Compare(a.skillPoints, b.skillPoints)
			},
		},
	})
	a := &characterWealthDetails{
		columnSorter: xwidget.NewColumnSorter(columns, "Character", xwidget.SortAsc),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	a.searchEntry = xwidget.NewSearchEntry("Search characters", func(_ string) {
		a.filterRowsAsync("")
	})

	showRow := func(r characterWealthDetailsRow) {
		o := r.eveEntity()
		if o.ID == 0 {
			return
		}
		u.InfoViewer().Show(o)
	}

	if a.u.IsMobile() {
		a.main = xwidget.MakeDataList(
			columns,
			&a.rowsFiltered,
			func(col string, r characterWealthDetailsRow) []widget.RichTextSegment {
				var s []widget.RichTextSegment
				switch col {
				case "Character":
					s = xwidget.RichTextSegmentsFromText(r.characterName)
				case "Tags":
					s = xwidget.RichTextSegmentsFromText(r.tagsDisplay)
				case "Wallet Balance":
					s = xwidget.RichTextSegmentsFromText(r.walletDisplay)
				case "Combined Assets":
					s = xwidget.RichTextSegmentsFromText(r.combinedAssetsDisplay)
				case "Contracts Escrow":
					s = xwidget.RichTextSegmentsFromText(r.contractsEscrowDisplay)
				case "Orders Escrow":
					s = xwidget.RichTextSegmentsFromText(r.ordersEscrowDisplay)
				case "Total Net Worth":
					s = xwidget.RichTextSegmentsFromText(r.totalNetWorthDisplay)
				case "Skill Points":
					s = xwidget.RichTextSegmentsFromText(r.skillPointsDisplay)
				}
				return s
			},
			showRow,
		)
	} else {
		a.main = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync,
			func(_ int, r characterWealthDetailsRow) {
				showRow(r)
			},
		)
	}
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	})
	return a
}

func (a *characterWealthDetails) CreateRenderer() fyne.WidgetRenderer {
	var topBox fyne.CanvasObject
	if a.u.IsMobile() {
		topBox = container.NewVBox(
			a.searchEntry,
			container.NewHScroll(container.NewHBox(a.sortChip)),
		)
	} else {
		topBox = a.searchEntry
	}
	c := container.NewBorder(topBox, nil, nil, nil, a.main)
	return widget.NewSimpleRenderer(c)
}

// setRows replaces the data. Must be called on the main thread.
func (a *characterWealthDetails) setRows(rows []characterWealthRow) {
	a.rows = xslices.Map(rows, newWealthDetailsRow)
	a.filterRowsAsync("")
}

func (a *characterWealthDetails) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.start()
	rows := slices.Clone(a.rows)
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r characterWealthDetailsRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		// add totals
		var assets, wallets, totals, contracts, orders, skillpoints []optional.Optional[float64]
		for _, r := range rows {
			assets = append(assets, r.combinedAssetsValue)
			contracts = append(contracts, r.contractsEscrow)
			orders = append(orders, r.ordersEscrow)
			skillpoints = append(skillpoints, r.skillPoints)
			totals = append(totals, r.totalNetWorth)
			wallets = append(wallets, r.walletBalance)
		}
		assetsTotal := optional.Sum(assets...)
		contractsTotal := optional.Sum(contracts...)
		grandTotal1 := optional.Sum(totals...)
		ordersTotal := optional.Sum(orders...)
		walletsTotal := optional.Sum(wallets...)
		skillpointsTotal := optional.Sum(skillpoints...)
		rows = append(rows, characterWealthDetailsRow{
			characterID:            0,
			characterName:          "TOTAL",
			combinedAssetsDisplay:  formatISKValue(assetsTotal),
			combinedAssetsValue:    assetsTotal,
			contractsEscrow:        contractsTotal,
			contractsEscrowDisplay: formatISKValue(contractsTotal),
			isTotal:                true,
			ordersEscrow:           ordersTotal,
			ordersEscrowDisplay:    formatISKValue(ordersTotal),
			searchTarget:           "",
			skillPoints:            skillpointsTotal,
			skillPointsDisplay:     formatISKValue(skillpointsTotal),
			totalNetWorth:          grandTotal1,
			totalNetWorthDisplay:   formatISKValue(grandTotal1),
			walletBalance:          walletsTotal,
			walletDisplay:          formatISKValue(walletsTotal),
		})

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.rowsFiltered = rows
			a.main.Refresh()
		})
	})
}

// reduceCharacterWealthValues keeps the top m rows by combined value, bucketing the rest into "Others".
func reduceCharacterWealthValues(rows []characterWealthValue, m int) []characterWealthValue {
	if len(rows) <= m {
		return rows
	}
	combined := func(v characterWealthValue) float64 { return v.assets + v.wallet + v.contracts + v.orders }
	slices.SortFunc(rows, func(a, b characterWealthValue) int {
		return cmp.Compare(combined(b), combined(a))
	})
	others := rows[m]
	others.name = "Others"
	for _, x := range rows[m+1:] {
		others.assets += x.assets
		others.wallet += x.wallet
		others.contracts += x.contracts
		others.orders += x.orders
	}
	rows = rows[:m]
	slices.SortFunc(rows, func(a, b characterWealthValue) int {
		return strings.Compare(a.name, b.name)
	})
	rows = append(rows, others)
	return rows
}

// reduceSliceValues buckets entries below minShare of the total into
// "Others", always keeping the top minCount entries by value regardless of
// share, so the result never collapses to a single 100% "Others" slice.
func reduceSliceValues(rows []namedValue, minShare float64, minCount int) []namedValue {
	var total float64
	for _, r := range rows {
		total += r.value
	}
	if total <= 0 {
		return rows
	}

	sortedByValue := slices.Clone(rows)
	slices.SortFunc(sortedByValue, func(a, b namedValue) int {
		return cmp.Compare(b.value, a.value)
	})
	floor := min(minCount, len(sortedByValue))
	guaranteed := make(map[string]bool, floor)
	for _, r := range sortedByValue[:floor] {
		guaranteed[r.name] = true
	}

	kept := make([]namedValue, 0, len(rows))
	var others float64
	for _, r := range rows {
		if !guaranteed[r.name] && r.value/total < minShare {
			others += r.value
			continue
		}
		kept = append(kept, r)
	}
	if others <= 0 {
		return kept
	}
	slices.SortFunc(kept, func(a, b namedValue) int {
		return strings.Compare(a.name, b.name)
	})
	kept = append(kept, namedValue{name: "Others", value: others})
	return kept
}

func formatISKValue(v optional.Optional[float64]) string {
	return v.StringFunc("?", func(v float64) string {
		return humanize.FormatFloat("#,###.", v)
	})
}

// showHelpPopUp shows a popUp with text as content
// and it's position aligned to widget obj.
func showHelpPopUp(text string, isMobile bool, obj fyne.CanvasObject) {
	var pu *widget.PopUp
	closePopUp := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		pu.Hide()
	})
	title := widget.NewLabel("Help")
	title.TextStyle.Bold = true
	body := widget.NewLabel(text)
	body.Wrapping = fyne.TextWrapWord

	p := theme.Padding()
	canvas := fyne.CurrentApp().Driver().CanvasForObject(obj)
	var spacerSize fyne.Size
	if isMobile {
		_, s := canvas.InteractiveArea()
		spacerSize = fyne.NewSize(s.Width-2*p, s.Height/2)
	} else {
		spacerSize = fyne.NewSize(300, 400)
	}
	spacer := xwidget.NewSpacer(spacerSize)
	c := container.NewStack(spacer, container.NewBorder(
		container.NewHBox(title, layout.NewSpacer(), closePopUp),
		nil,
		nil,
		nil,
		container.NewVScroll(container.NewPadded(body)),
	))
	pu = widget.NewPopUp(c, canvas)

	if isMobile {
		pos, s := canvas.InteractiveArea()
		x := pos.X
		y := pos.Y + s.Height/2
		pu.ShowAtPosition(fyne.NewPos(x, y))
	} else {
		x := obj.MinSize().Width - pu.MinSize().Width
		y := obj.MinSize().Height - pu.MinSize().Height + 2*p
		pu.ShowAtRelativePosition(fyne.NewPos(x, y), obj)
	}
}

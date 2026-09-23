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
	"fyne.io/fyne/v2/widget"
	"github.com/nathabonfim59/fyneline"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

const (
	wealthMaxCharacters        = 10
	wealthMinSliceCount        = 1 // not included others
	wealthMinSliceShare        = 0.05
	wealthMultiplier           = 1_000_000_000
	wealthNameTruncationLimit  = 20
	wealthNameTruncationSuffix = 0
)

type characterWealthRow struct {
	characterID     int64
	characterName   string
	walletBalance   float64
	combinedAssets  float64
	contractsEscrow float64
	ordersEscrow    float64
	total           float64
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

	characterBreakdownCard       *chartCard
	characterBreakdownChart      *fyneline.BarChart[characterWealthValue]
	characterBreakdownTitleLabel *widget.Label
	characterSplitCard           *chartCard
	characterSplitChart          *fyneline.ArcChart[namedValue]
	characterSplitTitleLabel     *widget.Label
	details                      *WealthDetails
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
		topLabel:                 ui.NewLabelWithWrapping(""),
		totalSplitChart: fyneline.NewArcChart([]namedValue(nil),
			func(v namedValue) float64 { return v.value },
			sliceLabel,
		),
		totalSplitTitleLabel: newChartTitleLabel(),
		u:                    u,
		details:              newWealthDetails(u),
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
	return a
}

func (a *CharacterWealth) CreateRenderer() fyne.WidgetRenderer {
	tabs := container.NewAppTabs(
		container.NewTabItem(
			"Overview",
			container.NewAdaptiveGrid(2, a.totalSplitCard, a.characterSplitCard),
		),
		container.NewTabItem("Characters", a.characterBreakdownCard),
		container.NewTabItem("Details", a.details),
	)
	var c fyne.CanvasObject
	if !a.u.IsMobile() {
		c = container.NewBorder(
			a.topLabel,
			nil,
			nil,
			nil,
			tabs,
		)
	} else {
		c = tabs
	}
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterWealth) update(ctx context.Context) {
	rows, total, err := a.fetchData(ctx)
	if err != nil {
		slog.Error("Failed to fetch data for charts", "err", err)
		fyne.Do(func() {
			a.topLabel.Text = fmt.Sprintf("Failed to fetch data for charts: %s", a.u.ErrorDisplay(err))
			a.topLabel.Importance = widget.DangerImportance
			a.topLabel.Refresh()
			a.topLabel.Show()
		})
		return
	}
	if len(rows) == 0 {
		fyne.Do(func() {
			a.topLabel.Text = "No characters"
			a.topLabel.Importance = widget.LowImportance
			a.topLabel.Refresh()
			a.topLabel.Show()
		})
		return
	}

	fyne.Do(func() {
		a.topLabel.Hide()
	})

	a.updateAssetWalletDetail(ctx, rows)
	a.updateCharacterSplit(ctx, rows)
	a.updateTotalSplit(ctx, rows)

	fyne.Do(func() {
		if a.OnUpdate != nil {
			a.OnUpdate(total)
		}
	})
}

func (a *CharacterWealth) updateAssetWalletDetail(_ context.Context, rows []characterWealthRow) {
	var total float64
	d := make([]characterWealthValue, 0, len(rows))
	for _, r := range rows {
		d = append(d, characterWealthValue{
			name:      r.characterName,
			assets:    r.combinedAssets,
			wallet:    r.walletBalance,
			contracts: r.contractsEscrow,
			orders:    r.ordersEscrow,
		})
		total += r.total
	}
	d = reduceAssetWalletValues(d, wealthMaxCharacters)

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

func (a *CharacterWealth) updateCharacterSplit(_ context.Context, rows []characterWealthRow) {
	var total float64
	d := make([]namedValue, 0, len(rows))
	for _, r := range rows {
		d = append(d, namedValue{name: r.characterName, value: r.total})
		total += r.total
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

func (a *CharacterWealth) updateTotalSplit(_ context.Context, rows []characterWealthRow) {
	var assets, wallets, contracts, orders, total float64
	for _, r := range rows {
		assets += r.combinedAssets
		contracts += r.contractsEscrow
		orders += r.ordersEscrow
		total += r.total
		wallets += r.walletBalance
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

func (a *CharacterWealth) fetchData(ctx context.Context) ([]characterWealthRow, optional.Optional[float64], error) {
	cc, err := a.u.Character().ListCharacters(ctx)
	if err != nil {
		return nil, optional.Optional[float64]{}, err
	}
	var rows []characterWealthRow
	var totals []optional.Optional[float64]
	for _, c := range cc {
		combinedAssets := c.CombinedAssetsValue()
		total := optional.Sum(c.WalletBalance, combinedAssets, c.ContractsEscrow, c.OrdersEscrow)
		totals = append(totals, total)
		if total.IsEmpty() {
			continue
		}
		name := xstrings.TruncateWithSuffix(c.EveCharacter.Name, wealthNameTruncationLimit, wealthNameTruncationSuffix)
		r := characterWealthRow{
			characterID:     c.ID,
			characterName:   name,
			combinedAssets:  combinedAssets.ValueOrZero() / wealthMultiplier,
			contractsEscrow: c.ContractsEscrow.ValueOrZero() / wealthMultiplier,
			ordersEscrow:    c.OrdersEscrow.ValueOrZero() / wealthMultiplier,
			total:           total.ValueOrZero() / wealthMultiplier,
			walletBalance:   c.WalletBalance.ValueOrZero() / wealthMultiplier,
		}
		rows = append(rows, r)
	}
	slices.SortFunc(rows, func(a, b characterWealthRow) int {
		return strings.Compare(a.characterName, b.characterName)
	})
	grantTotal := optional.Sum(totals...)
	return rows, grantTotal, nil
}

// reduceAssetWalletValues keeps the top m rows by combined value, bucketing the rest into "Others".
func reduceAssetWalletValues(rows []characterWealthValue, m int) []characterWealthValue {
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

package screens

import (
	"cmp"
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"math"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/nathabonfim59/fyneline"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

const (
	wealthArcCornerRadius      = 8
	wealthArcInnerRadius       = 0.6
	wealthArcPadAngle          = 1.5
	wealthMaxCharacters        = 10
	wealthMinSliceCount        = 1 // not included others
	wealthMinSliceShare        = 0.05
	wealthMultiplier           = 1_000_000_000
	wealthNameTruncationLimit  = 20
	wealthNameTruncationSuffix = 0
)

// wealthWalletSeriesColor, wealthContractsSeriesColor, and
// wealthOrdersSeriesColor match fyneline's default series colors.
var (
	wealthWalletSeriesColor    = color.NRGBA{R: 240, G: 135, B: 48, A: 255}
	wealthContractsSeriesColor = color.NRGBA{R: 47, G: 176, B: 117, A: 255}
	wealthOrdersSeriesColor    = color.NRGBA{R: 220, G: 72, B: 103, A: 255}
)

type wealthRow struct {
	characterID     int64
	characterName   string
	walletBalance   float64
	combinedAssets  float64
	contractsEscrow float64
	ordersEscrow    float64
	total           float64
}

// namedValue is a single category/value pair used by the charts.
type namedValue struct {
	name  string
	value float64
}

// assetWalletValue holds one character's wealth breakdown.
type assetWalletValue struct {
	name      string
	assets    float64
	wallet    float64
	contracts float64
	orders    float64
}

type Wealth struct {
	widget.BaseWidget

	OnUpdate func(totalNetWorth optional.Optional[float64])

	characters             *fyneline.BarChart[assetWalletValue]
	charactersCard         *chartCard
	assetWalletDetailTitle *widget.Label
	assetsSwatch           *legendSwatch
	walletSwatch           *legendSwatch
	contractsSwatch        *legendSwatch
	ordersSwatch           *legendSwatch
	characterSplit         *fyneline.ArcChart[namedValue]
	characterSplitCard     *chartCard
	characterSplitTitle    *widget.Label
	top                    *widget.Label
	totalSplit             *fyneline.ArcChart[namedValue]
	totalSplitCard         *chartCard
	totalSplitTitle        *widget.Label
	u                      baseUI
	details                *WealthDetails
}

func NewWealth(u baseUI) *Wealth {
	sliceLabel := func(v namedValue) string { return fmt.Sprintf("%s: %.1f", v.name, v.value) }
	a := &Wealth{
		characters: fyneline.NewBarChart([]assetWalletValue(nil),
			func(v assetWalletValue) string { return v.name },
			fyneline.NewBarSeries("Assets", func(v assetWalletValue) float64 { return v.assets }),
			fyneline.NewBarSeries("Wallet", func(v assetWalletValue) float64 { return v.wallet }),
			fyneline.NewBarSeries("Contracts", func(v assetWalletValue) float64 { return v.contracts }),
			fyneline.NewBarSeries("Orders", func(v assetWalletValue) float64 { return v.orders }),
		),
		assetWalletDetailTitle: newChartTitleLabel(),
		characterSplit: fyneline.NewArcChart([]namedValue(nil),
			func(v namedValue) float64 { return v.value },
			sliceLabel,
		),
		characterSplitTitle: newChartTitleLabel(),
		top:                 ui.NewLabelWithWrapping(""),
		totalSplit: fyneline.NewArcChart([]namedValue(nil),
			func(v namedValue) float64 { return v.value },
			sliceLabel,
		),
		totalSplitTitle: newChartTitleLabel(),
		u:               u,
		details:         newWealthDetails(u),
	}
	a.ExtendBaseWidget(a)
	a.top.Hide()

	a.characters.SetValueAxis(fyneline.NewNumericAxis().WithFormatter(wealthAxisValueFormatter))
	a.characters.SetOrientation(fyneline.BarHorizontal)
	a.characters.SetSeriesLayout(fyneline.SeriesStack)
	configureArcChart(a.characterSplit)
	configureArcChart(a.totalSplit)

	a.assetsSwatch = newLegendSwatch(func() color.Color { return theme.ColorForWidget(theme.ColorNamePrimary, a) })
	a.walletSwatch = newLegendSwatch(func() color.Color { return wealthWalletSeriesColor })
	a.contractsSwatch = newLegendSwatch(func() color.Color { return wealthContractsSeriesColor })
	a.ordersSwatch = newLegendSwatch(func() color.Color { return wealthOrdersSeriesColor })
	legend := newSeriesLegend(
		newLegendEntry("Assets", a.assetsSwatch),
		newLegendEntry("Wallet", a.walletSwatch),
		newLegendEntry("Contracts", a.contractsSwatch),
		newLegendEntry("Orders", a.ordersSwatch),
	)

	a.charactersCard = newChartCard(a.assetWalletDetailTitle, legend, a.characters)
	a.characterSplitCard = newChartCard(a.characterSplitTitle, nil, a.characterSplit)
	a.totalSplitCard = newChartCard(a.totalSplitTitle, nil, a.totalSplit)

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

func (a *Wealth) CreateRenderer() fyne.WidgetRenderer {
	tabs := container.NewAppTabs(
		container.NewTabItem(
			"Overview",
			container.NewAdaptiveGrid(2, a.totalSplitCard, a.characterSplitCard),
		),
		container.NewTabItem("Characters", a.charactersCard),
		container.NewTabItem("Details", a.details),
	)
	var c fyne.CanvasObject
	if !a.u.IsMobile() {
		c = container.NewBorder(
			a.top,
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

func (a *Wealth) update(ctx context.Context) {
	rows, total, err := a.fetchData(ctx)
	if err != nil {
		slog.Error("Failed to fetch data for charts", "err", err)
		fyne.Do(func() {
			a.top.Text = fmt.Sprintf("Failed to fetch data for charts: %s", a.u.ErrorDisplay(err))
			a.top.Importance = widget.DangerImportance
			a.top.Refresh()
			a.top.Show()
		})
		return
	}
	if len(rows) == 0 {
		fyne.Do(func() {
			a.top.Text = "No characters"
			a.top.Importance = widget.LowImportance
			a.top.Refresh()
			a.top.Show()
		})
		return
	}

	fyne.Do(func() {
		a.top.Hide()
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

func (a *Wealth) updateAssetWalletDetail(_ context.Context, rows []wealthRow) {
	var total float64
	d := make([]assetWalletValue, 0, len(rows))
	for _, r := range rows {
		d = append(d, assetWalletValue{
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
		a.characters.SetValueAxis(fyneline.NewNumericAxis().
			WithFormatter(wealthAxisValueFormatter).
			WithDomain(0, axisMax).
			WithTickCount(tickCount))
		a.characters.SetData(d)
		a.assetWalletDetailTitle.SetText(fmt.Sprintf("Wealth Breakdown by Character - Total: %.1f B", total))
	})
}

func (a *Wealth) updateCharacterSplit(_ context.Context, rows []wealthRow) {
	var total float64
	d := make([]namedValue, 0, len(rows))
	for _, r := range rows {
		d = append(d, namedValue{name: r.characterName, value: r.total})
		total += r.total
	}
	d = reduceSliceValues(d, wealthMinSliceShare, wealthMinSliceCount)

	fyne.Do(func() {
		a.characterSplit.SetData(d)
		a.characterSplitTitle.SetText(fmt.Sprintf("Total Net Worth By Character - Total: %.1f B", total))
	})
}

func (a *Wealth) updateTotalSplit(_ context.Context, rows []wealthRow) {
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
		a.totalSplit.SetData(d)
		title := fmt.Sprintf("Total Net Worth By Category - Total: %.1f B", total)
		a.totalSplitTitle.SetText(title)
	})
}

func (a *Wealth) fetchData(ctx context.Context) ([]wealthRow, optional.Optional[float64], error) {
	cc, err := a.u.Character().ListCharacters(ctx)
	if err != nil {
		return nil, optional.Optional[float64]{}, err
	}
	var rows []wealthRow
	var totals []optional.Optional[float64]
	for _, c := range cc {
		combinedAssets := c.CombinedAssetsValue()
		total := optional.Sum(c.WalletBalance, combinedAssets, c.ContractsEscrow, c.OrdersEscrow)
		totals = append(totals, total)
		if total.IsEmpty() {
			continue
		}
		name := xstrings.TruncateWithSuffix(c.EveCharacter.Name, wealthNameTruncationLimit, wealthNameTruncationSuffix)
		r := wealthRow{
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
	slices.SortFunc(rows, func(a, b wealthRow) int {
		return strings.Compare(a.characterName, b.characterName)
	})
	grantTotal := optional.Sum(totals...)
	return rows, grantTotal, nil
}

// newChartTitleLabel returns a bold label for a chart card's title.
func newChartTitleLabel() *widget.Label {
	return widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

// configureArcChart applies this screen's shared pie/doughnut styling.
func configureArcChart(chart *fyneline.ArcChart[namedValue]) {
	chart.SetLabels(true)
	chart.SetInnerRadius(wealthArcInnerRadius)
	chart.SetPadAngle(wealthArcPadAngle)
	chart.SetCornerRadius(wealthArcCornerRadius)
}

// wealthAxisValueFormatter formats a value-axis tick to 1 decimal.
func wealthAxisValueFormatter(v float64) string { return fmt.Sprintf("%.1f", v) }

// niceAxisBounds picks an axis max and tick count so ticks fall on round
// step boundaries and the last tick sits close to value, instead of value
// possibly landing well short of a coarsely-rounded max (e.g. 74 -> 100).
func niceAxisBounds(value float64, targetIntervals int) (axisMax float64, tickCount int) {
	if value <= 0 {
		return 1, 2
	}
	step := niceStep(value / float64(max(targetIntervals, 1)))
	axisMax = step * math.Ceil(value/step)
	return axisMax, int(math.Round(axisMax/step)) + 1
}

// niceStep rounds value up to the nearest "nice" 1-2-5-10 number at its
// order of magnitude.
func niceStep(value float64) float64 {
	magnitude := math.Pow(10, math.Floor(math.Log10(value)))
	normalized := value / magnitude
	var niceFraction float64
	switch {
	case normalized <= 1:
		niceFraction = 1
	case normalized <= 2:
		niceFraction = 2
	case normalized <= 5:
		niceFraction = 5
	default:
		niceFraction = 10
	}
	return niceFraction * magnitude
}

// reduceAssetWalletValues keeps the top m rows by combined value, bucketing the rest into "Others".
func reduceAssetWalletValues(rows []assetWalletValue, m int) []assetWalletValue {
	if len(rows) <= m {
		return rows
	}
	combined := func(v assetWalletValue) float64 { return v.assets + v.wallet + v.contracts + v.orders }
	slices.SortFunc(rows, func(a, b assetWalletValue) int {
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
	slices.SortFunc(rows, func(a, b assetWalletValue) int {
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

// chartCard wraps a chart with a title, an optional legend, and a themed
// grey backdrop, and adds theming support.
type chartCard struct {
	widget.BaseWidget

	title   *widget.Label
	legend  fyne.CanvasObject
	chart   fyne.CanvasObject
	bg      *canvas.Rectangle
	variant fyne.ThemeVariant
}

// newChartCard creates a chart card; applyTheme reapplies colors fyneline
// itself won't re-derive, e.g. WithFill fills.
func newChartCard(title *widget.Label, legend, chart fyne.CanvasObject) *chartCard {
	w := &chartCard{
		title:  title,
		legend: legend,
		chart:  chart,
		bg:     canvas.NewRectangle(color.Transparent),
	}
	w.ExtendBaseWidget(w)
	w.bg.CornerRadius = theme.Size(theme.SizeNameCardRadius)
	w.bg.FillColor = theme.Color(theme.ColorNameInputBackground)
	return w
}

func (w *chartCard) CreateRenderer() fyne.WidgetRenderer {
	content := container.NewBorder(w.title, w.legend, nil, nil, w.chart)
	return widget.NewSimpleRenderer(container.NewStack(w.bg, container.NewPadded(content)))
}

func (w *chartCard) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(theme.ColorNameInputBackground, v)
	w.bg.Refresh()
	w.BaseWidget.Refresh()
}

// legendSwatch is a color swatch that tracks a theme-derived color.
type legendSwatch struct {
	rect    *canvas.Rectangle
	colorFn func() color.Color
}

func newLegendSwatch(colorFn func() color.Color) *legendSwatch {
	return &legendSwatch{rect: canvas.NewRectangle(colorFn()), colorFn: colorFn}
}

func (s *legendSwatch) object() fyne.CanvasObject {
	const swatchSize = 12
	return container.NewGridWrap(fyne.NewSize(swatchSize, swatchSize), s.rect)
}

func newLegendEntry(label string, swatch *legendSwatch) fyne.CanvasObject {
	// Match the text size fyneline uses for its axis labels.
	l := widget.NewLabel(label)
	l.SizeName = theme.SizeNameCaptionText
	return container.NewHBox(container.NewCenter(swatch.object()), container.NewCenter(l))
}

// newSeriesLegend centers a row of legend entries (as built by [newLegendEntry]),
// for use in a chartCard's legend slot.
func newSeriesLegend(entries ...fyne.CanvasObject) fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 0, len(entries)+2)
	objects = append(objects, layout.NewSpacer())
	objects = append(objects, entries...)
	objects = append(objects, layout.NewSpacer())
	return container.NewHBox(objects...)
}

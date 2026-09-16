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
	wealthArcCornerRadius      = 4
	wealthArcInnerRadius       = 0.6
	wealthArcPadAngle          = 1.5
	wealthMaxCharacters        = 10
	wealthMinSliceShare        = 0.05
	wealthMultiplier           = 1_000_000_000
	wealthNameTruncationLimit  = 20
	wealthNameTruncationSuffix = 0
)

// wealthWalletSeriesColor matches fyneline's default second-series color.
var wealthWalletSeriesColor = color.NRGBA{R: 240, G: 135, B: 48, A: 255}

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

// assetWalletValue holds assets and wallet balance for one character.
type assetWalletValue struct {
	name   string
	assets float64
	wallet float64
}

type Wealth struct {
	widget.BaseWidget

	OnUpdate func(totalNetWorth optional.Optional[float64])

	assetWalletDetail      *fyneline.BarChart[assetWalletValue]
	characters             *chartCard
	assetWalletDetailTitle *widget.Label
	assetsSwatch           *legendSwatch
	walletSwatch           *legendSwatch
	characterSplit         *fyneline.ArcChart[namedValue]
	characterSplitCard     *chartCard
	characterSplitTitle    *widget.Label
	top                    *widget.Label
	totalSplit             *fyneline.ArcChart[namedValue]
	totalSplitCard         *chartCard
	totalSplitTitle        *widget.Label
	u                      baseUI
	details                *WealthOverview
}

func NewWealth(u baseUI) *Wealth {
	sliceLabel := func(v namedValue) string { return fmt.Sprintf("%s: %.1f", v.name, v.value) }
	a := &Wealth{
		assetWalletDetail: fyneline.NewBarChart([]assetWalletValue(nil),
			func(v assetWalletValue) string { return v.name },
			fyneline.NewBarSeries("Assets", func(v assetWalletValue) float64 { return v.assets }),
			fyneline.NewBarSeries("Wallet", func(v assetWalletValue) float64 { return v.wallet }),
		),
		assetWalletDetailTitle: widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		characterSplit: fyneline.NewArcChart([]namedValue(nil),
			func(v namedValue) float64 { return v.value },
			sliceLabel,
		),
		characterSplitTitle: widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		top:                 ui.NewLabelWithWrapping(""),
		totalSplit: fyneline.NewArcChart([]namedValue(nil),
			func(v namedValue) float64 { return v.value },
			sliceLabel,
		),
		totalSplitTitle: widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		u:               u,
		details:         NewWealthOverview(u),
	}
	a.ExtendBaseWidget(a)
	a.top.Hide()

	a.assetWalletDetail.SetValueAxis(fyneline.NewNumericAxis().WithFormatter(wealthAxisValueFormatter))
	a.assetWalletDetail.SetOrientation(fyneline.BarHorizontal)
	a.assetWalletDetail.SetSeriesLayout(fyneline.SeriesGroup)
	a.characterSplit.SetLabels(true)
	a.characterSplit.SetInnerRadius(wealthArcInnerRadius)
	a.characterSplit.SetPadAngle(wealthArcPadAngle)
	a.characterSplit.SetCornerRadius(wealthArcCornerRadius)
	a.totalSplit.SetLabels(true)
	a.totalSplit.SetInnerRadius(wealthArcInnerRadius)
	a.totalSplit.SetPadAngle(wealthArcPadAngle)
	a.totalSplit.SetCornerRadius(wealthArcCornerRadius)

	a.assetsSwatch = newLegendSwatch(func() color.Color { return theme.ColorForWidget(theme.ColorNamePrimary, a) })
	a.walletSwatch = newLegendSwatch(func() color.Color { return wealthWalletSeriesColor })
	legend := container.NewHBox(
		layout.NewSpacer(),
		newLegendEntry("Assets", a.assetsSwatch),
		newLegendEntry("Wallet", a.walletSwatch),
		layout.NewSpacer(),
	)
	applyAssetWalletColors := func() {
		a.assetWalletDetail.SetSeries(
			fyneline.NewBarSeries("Assets", func(v assetWalletValue) float64 { return v.assets }).
				WithFill(theme.ColorForWidget(theme.ColorNamePrimary, a)),
			fyneline.NewBarSeries("Wallet", func(v assetWalletValue) float64 { return v.wallet }).
				WithFill(wealthWalletSeriesColor),
		)
		a.assetsSwatch.refresh()
		a.walletSwatch.refresh()
	}
	a.characters = newChartCard(a.assetWalletDetailTitle, legend, a.assetWalletDetail, applyAssetWalletColors)
	a.characterSplitCard = newChartCard(a.characterSplitTitle, nil, a.characterSplit, nil)
	a.totalSplitCard = newChartCard(a.totalSplitTitle, nil, a.totalSplit, nil)

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
		container.NewTabItem("Characters", a.characters),
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

// chartCard wraps a chart with a title, an optional legend, and a themed
// grey backdrop, and adds theming support.
type chartCard struct {
	widget.BaseWidget

	title      *widget.Label
	legend     fyne.CanvasObject
	chart      fyne.CanvasObject
	bg         *canvas.Rectangle
	applyTheme func()
	variant    fyne.ThemeVariant
}

// newChartCard creates a chart card; applyTheme reapplies colors fyneline
// itself won't re-derive, e.g. WithFill fills.
func newChartCard(title *widget.Label, legend, chart fyne.CanvasObject, applyTheme func()) *chartCard {
	c := &chartCard{
		title:      title,
		legend:     legend,
		chart:      chart,
		bg:         canvas.NewRectangle(color.Transparent),
		applyTheme: applyTheme,
	}
	c.ExtendBaseWidget(c)
	c.applyThemeColors()
	return c
}

func (c *chartCard) CreateRenderer() fyne.WidgetRenderer {
	content := container.NewBorder(c.title, c.legend, nil, nil, c.chart)
	return widget.NewSimpleRenderer(container.NewStack(c.bg, container.NewPadded(content)))
}

// applyThemeColors recomputes theme-derived colors and records the variant.
func (c *chartCard) applyThemeColors() {
	c.variant = fyne.CurrentApp().Settings().ThemeVariant()
	c.bg.FillColor = theme.ColorForWidget(theme.ColorNameInputBackground, c)
	c.bg.CornerRadius = theme.CurrentForWidget(c).Size(theme.SizeNameCardRadius)
	if c.applyTheme != nil {
		c.applyTheme()
	}
}

func (c *chartCard) Refresh() {
	if fyne.CurrentApp().Settings().ThemeVariant() != c.variant {
		c.applyThemeColors()
	}
	c.BaseWidget.Refresh()
}

// legendSwatch is a color swatch that tracks a theme-derived color.
type legendSwatch struct {
	rect    *canvas.Rectangle
	colorFn func() color.Color
}

func newLegendSwatch(colorFn func() color.Color) *legendSwatch {
	return &legendSwatch{rect: canvas.NewRectangle(colorFn()), colorFn: colorFn}
}

func (s *legendSwatch) refresh() {
	s.rect.FillColor = s.colorFn()
	s.rect.Refresh()
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
	var totalAssets, totalWallet float64
	d := make([]assetWalletValue, 0, len(rows))
	for _, r := range rows {
		d = append(d, assetWalletValue{name: r.characterName, assets: r.combinedAssets, wallet: r.walletBalance})
		totalAssets += r.combinedAssets
		totalWallet += r.walletBalance
	}
	d = reduceAssetWalletValues(d, wealthMaxCharacters)

	var maxValue float64
	for _, v := range d {
		maxValue = max(maxValue, v.assets, v.wallet)
	}
	niceMax := niceCeil(maxValue)

	fyne.Do(func() {
		a.assetWalletDetail.SetValueAxis(fyneline.NewNumericAxis().
			WithFormatter(wealthAxisValueFormatter).
			WithDomain(0, niceMax).
			WithTickCount(6))
		a.assetWalletDetail.SetData(d)
		a.assetWalletDetailTitle.SetText(fmt.Sprintf(
			"Characters - Total: %.1f B ISK", totalAssets+totalWallet))
	})
}

// wealthAxisValueFormatter formats a value-axis tick to 1 decimal.
func wealthAxisValueFormatter(v float64) string { return fmt.Sprintf("%.1f", v) }

// niceCeil rounds value up to a "nice" 1-2-5-10 number for round axis ticks.
func niceCeil(value float64) float64 {
	if value <= 0 {
		return 1
	}
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

func (a *Wealth) updateCharacterSplit(_ context.Context, rows []wealthRow) {
	var total float64
	d := make([]namedValue, 0, len(rows))
	for _, r := range rows {
		d = append(d, namedValue{name: r.characterName, value: r.total})
		total += r.total
	}
	d = reduceSliceValues(d, wealthMinSliceShare)

	fyne.Do(func() {
		a.characterSplit.SetData(d)
		a.characterSplitTitle.SetText(fmt.Sprintf("Total Net Worth By Character - Total: %.1f B ISK", total))
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
		{name: "Wallet Balances", value: wallets},
		{name: "Combined Assets", value: assets},
		{name: "Contracts Escrow", value: contracts},
		{name: "Orders Escrow", value: orders},
	}

	fyne.Do(func() {
		a.totalSplit.SetData(d)
		title := fmt.Sprintf("Total Net Worth By Category - Total: %.1f B ISK", total)
		a.totalSplitTitle.SetText(title)
	})
}

// reduceAssetWalletValues keeps the top m rows by combined value, bucketing the rest into "Others".
func reduceAssetWalletValues(rows []assetWalletValue, m int) []assetWalletValue {
	if len(rows) <= m {
		return rows
	}
	slices.SortFunc(rows, func(a, b assetWalletValue) int {
		return cmp.Compare(b.assets+b.wallet, a.assets+a.wallet)
	})
	othersAssets, othersWallet := rows[m].assets, rows[m].wallet
	for _, x := range rows[m+1:] {
		othersAssets += x.assets
		othersWallet += x.wallet
	}
	rows = rows[:m]
	slices.SortFunc(rows, func(a, b assetWalletValue) int {
		return strings.Compare(a.name, b.name)
	})
	rows = append(rows, assetWalletValue{name: "Others", assets: othersAssets, wallet: othersWallet})
	return rows
}

// reduceSliceValues buckets entries below minShare of the total into "Others".
func reduceSliceValues(rows []namedValue, minShare float64) []namedValue {
	var total float64
	for _, r := range rows {
		total += r.value
	}
	if total <= 0 {
		return rows
	}
	kept := make([]namedValue, 0, len(rows))
	var others float64
	for _, r := range rows {
		if r.value/total < minShare {
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

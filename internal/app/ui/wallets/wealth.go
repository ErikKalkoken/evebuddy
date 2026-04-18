package wallets

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-daehling/fyne-charts/pkg/coord"
	"github.com/s-daehling/fyne-charts/pkg/data"
	"github.com/s-daehling/fyne-charts/pkg/prop"
	"github.com/s-daehling/fyne-charts/pkg/style"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

const (
	wealthMultiplier           = 1_000_000_000
	wealthMaxCharacters        = 7
	wealthNameTruncationLimit  = 16
	wealthNameTruncationSuffix = 2
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

type Wealth struct {
	widget.BaseWidget

	OnUpdate func(wallet, assets float64)

	assetDetail          *coord.CartesianCategoricalChart
	characterSplit       *prop.PieChart
	top                  *widget.Label
	totalSplit           *prop.PieChart
	u                    baseUI
	walletDetail         *coord.CartesianCategoricalChart
	defaultPieLabelStyle style.ValueLabelStyle
	defaultBarLabelStyle style.ValueLabelStyle
	overview             *Overview
}

func NewWealth(u baseUI) *Wealth {
	a := &Wealth{
		assetDetail:    coord.NewCartesianCategoricalChart(""),
		characterSplit: prop.NewPieChart(""),
		top:            ui.NewLabelWithWrapping(""),
		totalSplit:     prop.NewPieChart(""),
		u:              u,
		walletDetail:   coord.NewCartesianCategoricalChart(""),
		overview:       NewOverview(u),
	}
	a.ExtendBaseWidget(a)
	a.top.Hide()

	ts := style.DefaultTitleStyle()
	ts.SizeName = theme.SizeNameText
	ts.TextStyle.Bold = true
	yls := style.DefaultAxisLabelStyle()
	yls.SizeName = theme.SizeNameText

	a.assetDetail.SetTitleStyle(ts)
	a.assetDetail.HideLegend()
	a.assetDetail.SetYAxisStyle(yls, style.DefaultAxisStyle())
	a.assetDetail.SetYAxisLabel("B ISK")
	a.walletDetail.SetTitleStyle(ts)
	a.walletDetail.HideLegend()
	a.walletDetail.SetYAxisStyle(yls, style.DefaultAxisStyle())
	a.walletDetail.SetYAxisLabel("B ISK")
	a.totalSplit.SetTitleStyle(ts)
	a.characterSplit.SetTitleStyle(ts)

	a.defaultPieLabelStyle = style.DefaultValueLabelStyle()
	ls := style.DefaultValueLabelStyle()
	ls.StrokeWidth = 0
	a.defaultBarLabelStyle = ls

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
	a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		if arg.Section == app.SectionEveMarketPrices {
			a.update(ctx)
		}
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
		container.NewTabItem("Characters", a.overview),
		container.NewTabItem(
			"Total",
			container.NewAdaptiveGrid(2, a.totalSplit, a.characterSplit),
		),
		container.NewTabItem("Assets", a.assetDetail),
		container.NewTabItem("Wallets", a.walletDetail),
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
	rows, err := a.fetchData(ctx)
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

	a.updateAssetDetail(ctx, rows)
	a.updateCharacterSplit(ctx, rows)
	a.updateTotalSplit(ctx, rows)
	a.updateWalletDetail(ctx, rows)

	fyne.Do(func() {
		if a.OnUpdate != nil {
			a.OnUpdate(0, 0)
		}
	})
}

func (a *Wealth) updateAssetDetail(_ context.Context, rows []wealthRow) {
	colors := newColorWheel()
	var total float64
	var d []data.CategoricalPoint
	for _, r := range rows {
		d = append(d, data.CategoricalPoint{
			C:   r.characterName,
			Val: r.combinedAssets,
		})
		total += r.combinedAssets
	}
	d = reduceCategoricalPoints(d, wealthMaxCharacters)

	fyne.Do(func() {
		a.assetDetail.RemoveSeries("Characters")
		s, err := coord.NewCategoricalPointSeries("Characters", colors.next(), d)
		if err != nil {
			slog.Error("wealth: asset details", "error", err)
			return
		}
		s.SetValueLabelStyle(true, a.defaultBarLabelStyle)
		err = a.assetDetail.AddBarSeries(s)
		if err != nil {
			slog.Error("wealth: asset details", "error", err)
			return
		}
		a.assetDetail.SetTitle(fmt.Sprintf("Assets By Character - Total: %.1f B ISK", total))
	})
}

func (a *Wealth) updateCharacterSplit(_ context.Context, rows []wealthRow) {
	colors := newColorWheel()
	var total float64
	var d []data.ProportionalPoint
	for _, r := range rows {
		d = append(d, data.ProportionalPoint{
			C:       r.characterName,
			Val:     r.total,
			ColName: colors.next(),
		})
		total += r.total
	}
	d = reduceProportionalPoints(d, wealthMaxCharacters)
	fyne.Do(func() {
		a.characterSplit.RemoveSeries("Characters")
		s, err := prop.NewSeries("Characters", d)
		if err != nil {
			slog.Error("wealth: character split", "error", err)
			return
		}
		s.SetValueLabelStyle(true, a.defaultPieLabelStyle)
		err = a.characterSplit.AddSeries(s)
		if err != nil {
			slog.Error("wealth: character split", "error", err)
			return
		}
		a.characterSplit.SetTitle(fmt.Sprintf("Total Net Worth By Character - Total: %.1f B ISK", total))
	})
}

func (a *Wealth) updateTotalSplit(_ context.Context, rows []wealthRow) {
	colors := newColorWheel()
	var assets, wallets, contracts, orders, total float64
	for _, r := range rows {
		assets += r.combinedAssets
		contracts += r.contractsEscrow
		orders += r.ordersEscrow
		total += r.total
		wallets += r.walletBalance
	}
	fyne.Do(func() {
		a.totalSplit.RemoveSeries("")
		s, err := prop.NewSeries("", []data.ProportionalPoint{{
			C:       "Wallet Balances",
			Val:     wallets,
			ColName: colors.next(),
		}, {
			C:       "Combined Assets",
			Val:     assets,
			ColName: colors.next(),
		}, {
			C:       "Contracts Escrow",
			Val:     contracts,
			ColName: colors.next(),
		}, {
			C:       "Orders Escrow",
			Val:     orders,
			ColName: colors.next(),
		}})
		if err != nil {
			slog.Error("wealth: total split", "error", err)
			return
		}
		s.SetValueLabelStyle(true, a.defaultPieLabelStyle)
		err = a.totalSplit.AddSeries(s)
		if err != nil {
			slog.Error("wealth: total split", "error", err)
			return
		}
		title := fmt.Sprintf("Total Net Worth By Category - Total: %.1f B ISK", total)
		a.totalSplit.SetTitle(title)
	})
}

func (a *Wealth) updateWalletDetail(_ context.Context, rows []wealthRow) {
	colors := newColorWheel()
	var total float64
	var d []data.CategoricalPoint
	for _, r := range rows {
		d = append(d, data.CategoricalPoint{
			C:   r.characterName,
			Val: r.walletBalance,
		})
		total += r.walletBalance
	}
	d = reduceCategoricalPoints(d, wealthMaxCharacters)
	fyne.Do(func() {
		a.walletDetail.RemoveSeries("Characters")
		s, err := coord.NewCategoricalPointSeries("Characters", colors.next(), d)
		if err != nil {
			slog.Error("wealth: wallet details", "error", err)
			return
		}
		s.SetValueLabelStyle(true, a.defaultBarLabelStyle)
		err = a.walletDetail.AddBarSeries(s)
		if err != nil {
			slog.Error("wealth: wallet details", "error", err)
			return
		}
		a.walletDetail.SetTitle(fmt.Sprintf("Wallets By Character - Total: %.1f B ISK", total))
	})
}

func reduceProportionalPoints(rows []data.ProportionalPoint, m int) []data.ProportionalPoint {
	if len(rows) <= m {
		return rows
	}
	slices.SortFunc(rows, func(a, b data.ProportionalPoint) int {
		return cmp.Compare(b.Val, a.Val)
	})
	others := rows[m].Val
	if len(rows) > m {
		for _, x := range rows[m+1:] {
			others += x.Val
		}
	}
	rows = rows[:m]
	slices.SortFunc(rows, func(a, b data.ProportionalPoint) int {
		return strings.Compare(a.C, b.C)
	})
	rows = append(rows,
		data.ProportionalPoint{
			C:       "Others",
			Val:     others,
			ColName: theme.ColorNameDisabled,
		})
	return rows
}

func reduceCategoricalPoints(rows []data.CategoricalPoint, m int) []data.CategoricalPoint {
	if len(rows) <= m {
		return rows
	}
	slices.SortFunc(rows, func(a, b data.CategoricalPoint) int {
		return cmp.Compare(b.Val, a.Val)
	})
	others := rows[m].Val
	if len(rows) > m {
		for _, x := range rows[m+1:] {
			others += x.Val
		}
	}
	rows = rows[:m]
	slices.SortFunc(rows, func(a, b data.CategoricalPoint) int {
		return strings.Compare(a.C, b.C)
	})
	rows = append(rows,
		data.CategoricalPoint{
			C:   "Others",
			Val: others,
		})
	return rows
}

func (a *Wealth) fetchData(ctx context.Context) ([]wealthRow, error) {
	cc, err := a.u.Character().ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	var rows []wealthRow
	for _, c := range cc {
		combinedAssets := c.CombinedAssetsValue()
		total := optional.Sum(c.WalletBalance, combinedAssets, c.ContractsEscrow, c.OrdersEscrow)
		if total.IsEmpty() {
			continue
		}
		name := TruncateWithSuffix(c.EveCharacter.Name, wealthNameTruncationLimit, wealthNameTruncationSuffix)
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
	return rows, nil
}

type colorWheel struct {
	n      int
	colors []fyne.ThemeColorName
}

func newColorWheel() colorWheel {
	w := colorWheel{
		colors: []fyne.ThemeColorName{
			theme.ColorNamePrimary,
			theme.ColorNameWarning,
			theme.ColorNameSuccess,
			theme.ColorNameError,
			ui.ColorNameInfo,
			ui.ColorNameAttention,
			ui.ColorNameCreative,
			ui.ColorNameSystem,
			theme.ColorNamePlaceHolder,
		},
	}
	return w
}

func (w *colorWheel) next() fyne.ThemeColorName {
	c := w.colors[w.n]
	if w.n < len(w.colors)-1 {
		w.n++
	} else {
		w.n = 0
	}
	return c
}

// func (w *colorWheel) reset() {
// 	w.n = 0
// }

// TruncateWithSuffix shortens a string to length limit.
// It adds "..." and keeps 'suffixLen' characters at the end.
func TruncateWithSuffix(s string, limit int, suffixLen int) string {
	runes := []rune(strings.TrimRight(s, " "))
	if len(runes) <= limit {
		return string(runes)
	}
	prefixLen := max(limit-1-suffixLen, 0) // ellipsis counts as 1
	prefix := runes[:prefixLen]
	suffix := runes[len(runes)-suffixLen:]
	strSuffix := strings.TrimRight(string(suffix), " ")
	return string(prefix) + "..." + strSuffix
}

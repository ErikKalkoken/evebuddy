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
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const (
	wealthMultiplier           = 1_000_000_000
	wealthMaxCharacters        = 7
	wealthNameTruncationLimit  = 16
	wealthNameTruncationSuffix = 2
)

type Wealth struct {
	widget.BaseWidget

	OnUpdate func(wallet, assets float64)

	assetDetail    *coord.CartesianCategoricalChart
	assetSplit     *prop.PieChart
	characterSplit *prop.PieChart
	top            *widget.Label
	totalSplit     *prop.PieChart
	u              baseUI
	walletDetail   *coord.CartesianCategoricalChart
	walletSplit    *prop.PieChart
}

func NewWealth(u baseUI) *Wealth {
	a := &Wealth{
		assetDetail:    coord.NewCartesianCategoricalChart(""),
		assetSplit:     prop.NewPieChart(""),
		characterSplit: prop.NewPieChart(""),
		top:            ui.NewLabelWithWrapping(""),
		totalSplit:     prop.NewPieChart(""),
		u:              u,
		walletDetail:   coord.NewCartesianCategoricalChart(""),
		walletSplit:    prop.NewPieChart(""),
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
	a.assetSplit.SetTitleStyle(ts)
	a.walletSplit.SetTitleStyle(ts)
	a.walletDetail.SetTitleStyle(ts)
	a.walletDetail.HideLegend()
	a.walletDetail.SetYAxisStyle(yls, style.DefaultAxisStyle())
	a.walletDetail.SetYAxisLabel("B ISK")
	a.totalSplit.SetTitleStyle(ts)
	a.characterSplit.SetTitleStyle(ts)

	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		switch arg.Section {
		case app.SectionCharacterAssets, app.SectionCharacterWalletBalance:
			a.Update(ctx)
		}
	})
	a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		if arg.Section == app.SectionEveMarketPrices {
			a.Update(ctx)
		}
	})
	return a
}

func (a *Wealth) CreateRenderer() fyne.WidgetRenderer {
	tabs := container.NewAppTabs(
		container.NewTabItem("Total", container.NewBorder(
			container.NewPadded(),
			nil,
			nil,
			nil,
			container.NewAdaptiveGrid(2, a.totalSplit, a.characterSplit),
		)),
		container.NewTabItem("Breakdown", container.NewBorder(
			container.NewPadded(),
			nil,
			nil,
			nil,
			container.NewAdaptiveGrid(2, a.assetSplit, a.walletSplit),
		)),
		container.NewTabItem("Assets", container.NewBorder(
			container.NewPadded(),
			nil,
			nil,
			nil,
			a.assetDetail,
		)),
		container.NewTabItem("Wallets", container.NewBorder(
			container.NewPadded(),
			nil,
			nil,
			nil,
			a.walletDetail,
		)),
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

func (a *Wealth) Update(ctx context.Context) {
	rows, characters, err := a.fetchData(ctx)
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
	if characters == 0 {
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

	var totalWallet, totalAssets float64
	for _, r := range rows {
		totalAssets += r.assets
		totalWallet += r.wallet
	}
	a.updateAssetDetail(ctx, rows, totalAssets)
	a.updateAssetSplit(ctx, rows, totalAssets)
	a.updateCharacterSplit(ctx, rows, totalAssets, totalWallet)
	a.updateTotalSplit(ctx, totalAssets, totalWallet)
	a.updateWalletDetail(ctx, rows, totalWallet)
	a.updateWalletSplit(ctx, rows, totalWallet)

	fyne.Do(func() {
		if a.OnUpdate != nil {
			a.OnUpdate(totalWallet*wealthMultiplier, totalAssets*wealthMultiplier)
		}
	})
}

func (a *Wealth) updateAssetDetail(_ context.Context, rows []wealthRow, totalAssets float64) {
	colors := newColorWheel()
	d := xslices.Map(rows, func(r wealthRow) data.CategoricalPoint {
		return data.CategoricalPoint{
			C:   r.character,
			Val: r.assets,
		}
	})
	d = reduceCategoricalPoints(d, wealthMaxCharacters)

	fyne.Do(func() {
		a.assetDetail.RemoveSeries("Characters")
		s, err := coord.NewCategoricalPointSeries("Characters", colors.next(), d)
		if err != nil {
			slog.Error("wealth: asset details", "error", err)
			return
		}
		err = a.assetDetail.AddBarSeries(s)
		if err != nil {
			slog.Error("wealth: asset details", "error", err)
			return
		}
		a.assetDetail.SetTitle(fmt.Sprintf("Assets By Character - Total: %.1f B ISK", totalAssets))
	})
}

func (a *Wealth) updateAssetSplit(_ context.Context, rows []wealthRow, totalAssets float64) {
	colors := newColorWheel()
	d := xslices.Map(rows, func(r wealthRow) data.ProportionalPoint {
		return data.ProportionalPoint{
			C:       r.character,
			Val:     r.assets,
			ColName: colors.next(),
		}
	})
	d = reduceProportionalPoints(d, wealthMaxCharacters)
	fyne.Do(func() {
		a.assetSplit.RemoveSeries("Characters")
		s, err := prop.NewSeries("Characters", d)
		if err != nil {
			slog.Error("wealth: asset split", "error", err)
			return
		}
		err = a.assetSplit.AddSeries(s)
		if err != nil {
			slog.Error("wealth: asset split", "error", err)
			return
		}
		a.assetSplit.SetTitle(fmt.Sprintf("Assets By Character - Total: %.1f B ISK", totalAssets))
	})
}

func (a *Wealth) updateCharacterSplit(_ context.Context, rows []wealthRow, totalAssets float64, totalWallet float64) {
	colors := newColorWheel()
	d := xslices.Map(rows, func(r wealthRow) data.ProportionalPoint {
		return data.ProportionalPoint{
			C:       r.character,
			Val:     r.assets,
			ColName: colors.next(),
		}
	})
	d = reduceProportionalPoints(d, wealthMaxCharacters)
	fyne.Do(func() {
		a.characterSplit.RemoveSeries("Characters")
		s, err := prop.NewSeries("Characters", d)
		if err != nil {
			slog.Error("wealth: character split", "error", err)
			return
		}
		err = a.characterSplit.AddSeries(s)
		if err != nil {
			slog.Error("wealth: character split", "error", err)
			return
		}
		a.characterSplit.SetTitle(fmt.Sprintf("Wealth By Character - Total: %.1f B ISK", totalAssets+totalWallet))
	})
}

func (a *Wealth) updateTotalSplit(_ context.Context, totalAssets float64, totalWallet float64) {
	colors := newColorWheel()
	fyne.Do(func() {
		a.totalSplit.RemoveSeries("")
		s, err := prop.NewSeries("", []data.ProportionalPoint{{
			C:       "Assets combined",
			Val:     totalAssets,
			ColName: colors.next(),
		}, {
			C:       "Wallets combined",
			Val:     totalWallet,
			ColName: colors.next(),
		}})
		if err != nil {
			slog.Error("wealth: total split", "error", err)
			return
		}
		err = a.totalSplit.AddSeries(s)
		if err != nil {
			slog.Error("wealth: total split", "error", err)
			return
		}
		a.totalSplit.SetTitle(fmt.Sprintf("Wealth By Source - Total: %.1f B ISK", totalWallet+totalAssets))
	})
}

func (a *Wealth) updateWalletDetail(_ context.Context, rows []wealthRow, totalWallet float64) {
	colors := newColorWheel()
	d := xslices.Map(rows, func(r wealthRow) data.CategoricalPoint {
		return data.CategoricalPoint{
			C:   r.character,
			Val: r.wallet,
		}
	})
	d = reduceCategoricalPoints(d, wealthMaxCharacters)
	fyne.Do(func() {
		a.walletDetail.RemoveSeries("Characters")
		s, err := coord.NewCategoricalPointSeries("Characters", colors.next(), d)
		if err != nil {
			slog.Error("wealth: wallet details", "error", err)
			return
		}
		err = a.walletDetail.AddBarSeries(s)
		if err != nil {
			slog.Error("wealth: wallet details", "error", err)
			return
		}
		a.walletDetail.SetTitle(fmt.Sprintf("Wallets By Character - Total: %.1f B ISK", totalWallet))
	})
}

func (a *Wealth) updateWalletSplit(_ context.Context, rows []wealthRow, totalWallet float64) {
	colors := newColorWheel()
	d := xslices.Map(rows, func(r wealthRow) data.ProportionalPoint {
		return data.ProportionalPoint{
			C:       r.character,
			Val:     r.wallet,
			ColName: colors.next(),
		}
	})
	d = reduceProportionalPoints(d, wealthMaxCharacters)
	fyne.Do(func() {
		a.walletSplit.RemoveSeries("Characters")
		s, err := prop.NewSeries("Characters", d)
		if err != nil {
			slog.Error("wealth: wallet split", "error", err)
			return
		}
		err = a.walletSplit.AddSeries(s)
		if err != nil {
			slog.Error("wealth: wallet split", "error", err)
			return
		}
		a.walletSplit.SetTitle(fmt.Sprintf("Wallets By Character - Total: %.1f B ISK", totalWallet))
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

type wealthRow struct {
	character string
	wallet    float64
	assets    float64
	total     float64
}

func (a *Wealth) fetchData(ctx context.Context) ([]wealthRow, int, error) {
	cc, err := a.u.Character().ListCharacters(ctx)
	if err != nil {
		return nil, 0, err
	}
	var selected []*app.Character
	for _, c := range cc {
		hasAssets := a.u.StatusCache().HasCharacterSection(c.ID, app.SectionCharacterAssets)
		hasWallet := a.u.StatusCache().HasCharacterSection(c.ID, app.SectionCharacterWalletBalance)
		if hasAssets && hasWallet {
			selected = append(selected, c)
		}
	}
	var rows []wealthRow
	for _, c := range selected {
		assetTotal, err := a.u.Character().AssetTotalValue(ctx, c.ID)
		if err != nil {
			return nil, 0, err
		}
		if c.WalletBalance.IsEmpty() && assetTotal.IsEmpty() {
			continue
		}
		character := TruncateWithSuffix(c.EveCharacter.Name, wealthNameTruncationLimit, wealthNameTruncationSuffix)
		wallet := c.WalletBalance.ValueOrZero() / wealthMultiplier
		assets := assetTotal.ValueOrZero() / wealthMultiplier
		r := wealthRow{
			character: character,
			assets:    assets,
			wallet:    wallet,
			total:     assets + wallet,
		}
		rows = append(rows, r)
	}
	slices.SortFunc(rows, func(a, b wealthRow) int {
		return strings.Compare(a.character, b.character)
	})
	return rows, len(selected), nil
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

package ui

import (
	"cmp"
	"context"
	"fmt"
	"image/color"
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

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const (
	wealthMultiplier           = 1_000_000_000
	wealthMaxCharacters        = 7
	wealthNameTruncationLimit  = 16
	wealthNameTruncationSuffix = 2
)

type wealth struct {
	widget.BaseWidget

	onUpdate func(wallet, assets float64)

	assetDetail    *coord.CartesianCategoricalChart
	assetSplit     *prop.PieChart
	characterSplit *prop.PieChart
	top            *widget.Label
	totalSplit     *prop.PieChart
	u              *baseUI
	walletDetail   *coord.CartesianCategoricalChart
	walletSplit    *prop.PieChart
}

func newWealth(u *baseUI) *wealth {
	a := &wealth{
		assetDetail:    coord.NewCartesianCategoricalChart(""),
		assetSplit:     prop.NewPieChart(""),
		characterSplit: prop.NewPieChart(""),
		top:            makeTopLabel(),
		totalSplit:     prop.NewPieChart(""),
		u:              u,
		walletDetail:   coord.NewCartesianCategoricalChart(""),
		walletSplit:    prop.NewPieChart(""),
	}
	a.ExtendBaseWidget(a)
	a.top.Hide()
	size := theme.SizeNameSubHeadingText
	color := theme.ColorNameForeground
	a.assetDetail.SetTitleStyle(size, color)
	a.assetDetail.HideLegend()
	a.assetDetail.SetYAxisLabel("B ISK")
	a.assetSplit.SetTitleStyle(size, color)
	a.walletSplit.SetTitleStyle(size, color)
	a.walletDetail.SetTitleStyle(size, color)
	a.walletDetail.HideLegend()
	a.walletDetail.SetYAxisLabel("B ISK")
	a.totalSplit.SetTitleStyle(size, color)
	a.characterSplit.SetTitleStyle(size, color)

	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		switch arg.section {
		case app.SectionCharacterAssets, app.SectionCharacterWalletBalance:
			a.update()
		}
	})
	a.u.generalSectionChanged.AddListener(func(_ context.Context, arg generalSectionUpdated) {
		if arg.section == app.SectionEveMarketPrices {
			a.update()
		}
	})
	return a
}

func (a *wealth) CreateRenderer() fyne.WidgetRenderer {
	tabs := container.NewAppTabs(
		container.NewTabItem("Total", container.NewAdaptiveGrid(2, a.totalSplit, a.characterSplit)),
		container.NewTabItem("Breakdown", container.NewAdaptiveGrid(2, a.assetSplit, a.walletSplit)),
		container.NewTabItem("Assets", a.assetDetail),
		container.NewTabItem("Wallets", a.walletDetail),
	)
	var c fyne.CanvasObject
	if !a.u.isMobile {
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

func (a *wealth) update() {
	rows, characters, err := a.compileData(a.u.services())
	if err != nil {
		slog.Error("Failed to fetch data for charts", "err", err)
		fyne.Do(func() {
			a.top.Text = fmt.Sprintf("Failed to fetch data for charts: %s", a.u.humanizeError(err))
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
	// totals
	if a.onUpdate != nil {
		a.onUpdate(totalWallet*wealthMultiplier, totalAssets*wealthMultiplier)
	}

	a.updateAssetDetail(rows, totalAssets)
	a.updateAssetSplit(rows, totalAssets)
	a.updateCharacterSplit(rows, totalAssets, totalWallet)
	a.updateTotalSplit(totalAssets, totalWallet)
	a.updateWalletDetail(rows, totalWallet)
	a.updateWalletSplit(rows, totalWallet)
}

func (a *wealth) updateAssetDetail(rows []wealthRow, totalAssets float64) {
	colors := newColorWheel()
	d := xslices.Map(rows, func(r wealthRow) data.CategoricalPoint {
		return data.CategoricalPoint{
			C:   r.character,
			Val: r.assets,
		}
	})
	d = reduceCategoricalPoints(d, wealthMaxCharacters)
	s, err := coord.NewCategoricalPointSeries("Characters", colors.next(), d)
	if err != nil {
		slog.Error("wealth: asset details", "error", err)
		return
	}
	fyne.Do(func() {
		a.assetDetail.RemoveSeries("Characters")
		err = a.assetDetail.AddBarSeries(s)
		if err != nil {
			slog.Error("wealth: asset details", "error", err)
			return
		}
		a.assetDetail.SetTitle(fmt.Sprintf("Assets By Character - Total: %.1f B ISK", totalAssets))
	})
}

func (a *wealth) updateAssetSplit(rows []wealthRow, totalAssets float64) {
	colors := newColorWheel()
	d := xslices.Map(rows, func(r wealthRow) data.ProportionalPoint {
		return data.ProportionalPoint{
			C:   r.character,
			Val: r.assets,
			Col: colors.next(),
		}
	})
	d = reduceProportionalPoints(d, wealthMaxCharacters)
	s, err := prop.NewSeries("Characters", d)
	if err != nil {
		slog.Error("wealth: asset split", "error", err)
		return
	}
	fyne.Do(func() {
		a.assetSplit.RemoveSeries("Characters")
		err = a.assetSplit.AddSeries(s)
		if err != nil {
			slog.Error("wealth: asset split", "error", err)
			return
		}
		a.assetSplit.SetTitle(fmt.Sprintf("Assets By Character - Total: %.1f B ISK", totalAssets))
	})
}

func (a *wealth) updateCharacterSplit(rows []wealthRow, totalAssets float64, totalWallet float64) {
	colors := newColorWheel()
	d := xslices.Map(rows, func(r wealthRow) data.ProportionalPoint {
		return data.ProportionalPoint{
			C:   r.character,
			Val: r.assets,
			Col: colors.next(),
		}
	})
	d = reduceProportionalPoints(d, wealthMaxCharacters)
	s, err := prop.NewSeries("Characters", d)
	if err != nil {
		slog.Error("wealth: character split", "error", err)
		return
	}
	fyne.Do(func() {
		a.characterSplit.RemoveSeries("Characters")
		err = a.characterSplit.AddSeries(s)
		if err != nil {
			slog.Error("wealth: character split", "error", err)
			return
		}
		a.characterSplit.SetTitle(fmt.Sprintf("Wealth By Character - Total: %.1f B ISK", totalAssets+totalWallet))
	})
}

func (a *wealth) updateTotalSplit(totalAssets float64, totalWallet float64) {
	colors := newColorWheel()
	s, err := prop.NewSeries("", []data.ProportionalPoint{
		{
			C:   "Assets combined",
			Val: totalAssets,
			Col: colors.next(),
		},
		{
			C:   "Wallets combined",
			Val: totalWallet,
			Col: colors.next(),
		},
	})
	fyne.Do(func() {
		a.totalSplit.RemoveSeries("")
		err = a.totalSplit.AddSeries(s)
		if err != nil {
			slog.Error("wealth: total split", "error", err)
			return
		}
		a.totalSplit.SetTitle(fmt.Sprintf("Wealth By Source - Total: %.1f B ISK", totalWallet+totalAssets))
	})
}

func (a *wealth) updateWalletDetail(rows []wealthRow, totalWallet float64) {
	colors := newColorWheel()
	d := xslices.Map(rows, func(r wealthRow) data.CategoricalPoint {
		return data.CategoricalPoint{
			C:   r.character,
			Val: r.wallet,
		}
	})
	d = reduceCategoricalPoints(d, wealthMaxCharacters)
	s, err := coord.NewCategoricalPointSeries("Characters", colors.next(), d)
	if err != nil {
		slog.Error("wealth: wallet details", "error", err)
		return
	}
	fyne.Do(func() {
		a.walletDetail.RemoveSeries("Characters")
		err = a.walletDetail.AddBarSeries(s)
		if err != nil {
			slog.Error("wealth: wallet details", "error", err)
			return
		}
		a.walletDetail.SetTitle(fmt.Sprintf("Wallets By Character - Total: %.1f B ISK", totalWallet))
	})
}

func (a *wealth) updateWalletSplit(rows []wealthRow, totalWallet float64) {
	colors := newColorWheel()
	d := xslices.Map(rows, func(r wealthRow) data.ProportionalPoint {
		return data.ProportionalPoint{
			C:   r.character,
			Val: r.wallet,
			Col: colors.next(),
		}
	})
	d = reduceProportionalPoints(d, wealthMaxCharacters)
	s, err := prop.NewSeries("Characters", d)
	if err != nil {
		slog.Error("wealth: wallet split", "error", err)
		return
	}
	fyne.Do(func() {
		a.walletSplit.RemoveSeries("Characters")
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
			C:   "Others",
			Val: others,
			Col: theme.Color(theme.ColorNameDisabled),
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

func (*wealth) compileData(s services) ([]wealthRow, int, error) {
	ctx := context.Background()
	cc, err := s.cs.ListCharacters(ctx)
	if err != nil {
		return nil, 0, err
	}
	selected := make([]*app.Character, 0)
	for _, c := range cc {
		hasAssets := s.scs.HasCharacterSection(c.ID, app.SectionCharacterAssets)
		hasWallet := s.scs.HasCharacterSection(c.ID, app.SectionCharacterWalletBalance)
		if hasAssets && hasWallet {
			selected = append(selected, c)
		}
	}
	rows := make([]wealthRow, 0)
	for _, c := range selected {
		assetTotal, err := s.cs.AssetTotalValue(ctx, c.ID)
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
	colors []color.Color
}

func newColorWheel() colorWheel {
	w := colorWheel{
		colors: make([]color.Color, 0),
	}
	w.colors = []color.Color{
		theme.Color(theme.ColorNamePrimary),
		theme.Color(theme.ColorNameWarning),
		theme.Color(theme.ColorNameSuccess),
		theme.Color(theme.ColorNameError),
		theme.Color(colorNameInfo),
		theme.Color(colorNameAttention),
		theme.Color(colorNameCreative),
		theme.Color(colorNameSystem),
		theme.Color(theme.ColorNamePlaceHolder),
	}
	return w
}

func (w *colorWheel) next() color.Color {
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

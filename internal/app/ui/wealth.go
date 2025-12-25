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
	chartData "github.com/s-daehling/fyne-charts/pkg/data"
	"github.com/s-daehling/fyne-charts/pkg/prop"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const (
	wealthMultiplier    = 1_000_000_000
	wealthMaxCharacters = 7
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
	a.assetSplit.SetTitleStyle(size, color)
	a.walletSplit.SetTitleStyle(size, color)
	a.walletDetail.SetTitleStyle(size, color)
	a.walletDetail.HideLegend()
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
	if a.u.isDesktop {
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
	data, characters, err := a.compileData(a.u.services())
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
	for _, r := range data {
		totalAssets += r.assets
		totalWallet += r.wallet
	}
	colors := newColorWheel()

	// totals
	if a.onUpdate != nil {
		a.onUpdate(totalWallet*wealthMultiplier, totalAssets*wealthMultiplier)
	}

	// total
	s5, err := prop.NewSeries("", []chartData.ProportionalPoint{
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
		err = a.totalSplit.AddSeries(s5)
		if err != nil {
			panic(err)
		}
		a.totalSplit.SetTitle(fmt.Sprintf("Wealth By Source - Total: %.1f B ISK", totalWallet+totalAssets))
	})

	// Total split
	d6 := xslices.Map(data, func(r dataRow) chartData.ProportionalPoint {
		return chartData.ProportionalPoint{
			C:   r.label,
			Val: r.assets,
			Col: colors.next(),
		}
	})
	d6 = reduceProportionalPoints(d6, wealthMaxCharacters)
	s6, err := prop.NewSeries("Characters", d6)
	if err != nil {
		panic(err)
	}
	fyne.Do(func() {
		a.characterSplit.RemoveSeries("Characters")
		err = a.characterSplit.AddSeries(s6)
		if err != nil {
			panic(err)
		}
		a.characterSplit.SetTitle(fmt.Sprintf("Wealth By Character - Total: %.1f B ISK", totalAssets+totalWallet))
	})

	// Asset split
	d1 := xslices.Map(data, func(r dataRow) chartData.ProportionalPoint {
		return chartData.ProportionalPoint{
			C:   r.label,
			Val: r.assets,
			Col: colors.next(),
		}
	})
	d1 = reduceProportionalPoints(d1, wealthMaxCharacters)
	s1, err := prop.NewSeries("Characters", d1)
	if err != nil {
		panic(err)
	}
	fyne.Do(func() {
		a.assetSplit.RemoveSeries("Characters")
		err = a.assetSplit.AddSeries(s1)
		if err != nil {
			panic(err)
		}
		a.assetSplit.SetTitle(fmt.Sprintf("Assets By Character - Total: %.1f B ISK", totalAssets))
	})

	// Wallet split
	colors.reset()
	d2 := xslices.Map(data, func(r dataRow) chartData.ProportionalPoint {
		return chartData.ProportionalPoint{
			C:   r.label,
			Val: r.wallet,
			Col: colors.next(),
		}
	})
	d2 = reduceProportionalPoints(d2, wealthMaxCharacters)
	s2, err := prop.NewSeries("Characters", d2)
	if err != nil {
		panic(err)
	}
	fyne.Do(func() {
		a.walletSplit.RemoveSeries("Characters")
		err = a.walletSplit.AddSeries(s2)
		if err != nil {
			panic(err)
		}
		a.walletSplit.SetTitle(fmt.Sprintf("Wallets By Character - Total: %.1f B ISK", totalWallet))
	})

	// Assets detail
	colors.reset()
	d4 := xslices.Map(data, func(r dataRow) chartData.CategoricalPoint {
		return chartData.CategoricalPoint{
			C:   r.label,
			Val: r.assets,
		}
	})
	d4 = reduceCategoricalPoints(d4, wealthMaxCharacters)
	s4, err := coord.NewCategoricalPointSeries("Characters", colors.next(), d4)
	if err != nil {
		panic(err)
	}
	fyne.Do(func() {
		a.assetDetail.RemoveSeries("Characters")
		err = a.assetDetail.AddBarSeries(s4)
		if err != nil {
			panic(err)
		}
		a.assetDetail.SetTitle(fmt.Sprintf("Assets By Character - Total: %.1f B ISK", totalAssets))
	})

	// Wallet details
	colors.reset()
	d3 := xslices.Map(data, func(r dataRow) chartData.CategoricalPoint {
		return chartData.CategoricalPoint{
			C:   r.label,
			Val: r.wallet,
		}
	})
	d3 = reduceCategoricalPoints(d3, wealthMaxCharacters)
	s3, err := coord.NewCategoricalPointSeries("Characters", colors.next(), d3)
	if err != nil {
		panic(err)
	}
	fyne.Do(func() {
		a.walletDetail.RemoveSeries("Characters")
		err = a.walletDetail.AddBarSeries(s3)
		if err != nil {
			panic(err)
		}
		a.walletDetail.SetTitle(fmt.Sprintf("Wallets By Character - Total: %.1f B ISK", totalWallet))
	})

}

func reduceProportionalPoints(data []chartData.ProportionalPoint, m int) []chartData.ProportionalPoint {
	if len(data) <= m {
		return data
	}
	slices.SortFunc(data, func(a, b chartData.ProportionalPoint) int {
		return cmp.Compare(b.Val, a.Val)
	})
	others := data[m].Val
	if len(data) > m {
		for _, x := range data[m+1:] {
			others += x.Val
		}
	}
	data = data[:m]
	slices.SortFunc(data, func(a, b chartData.ProportionalPoint) int {
		return strings.Compare(a.C, b.C)
	})
	data = append(data,
		chartData.ProportionalPoint{
			C:   "Others",
			Val: others,
			Col: theme.Color(theme.ColorNameDisabled),
		})
	return data
}

func reduceCategoricalPoints(data []chartData.CategoricalPoint, m int) []chartData.CategoricalPoint {
	if len(data) <= m {
		return data
	}
	slices.SortFunc(data, func(a, b chartData.CategoricalPoint) int {
		return cmp.Compare(b.Val, a.Val)
	})
	others := data[m].Val
	if len(data) > m {
		for _, x := range data[m+1:] {
			others += x.Val
		}
	}
	data = data[:m]
	slices.SortFunc(data, func(a, b chartData.CategoricalPoint) int {
		return strings.Compare(a.C, b.C)
	})
	data = append(data,
		chartData.CategoricalPoint{
			C:   "Others",
			Val: others,
		})
	return data
}

type dataRow struct {
	label  string
	wallet float64
	assets float64
	total  float64
}

func (*wealth) compileData(s services) ([]dataRow, int, error) {
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
	data := make([]dataRow, 0)
	for _, c := range selected {
		assetTotal, err := s.cs.AssetTotalValue(ctx, c.ID)
		if err != nil {
			return nil, 0, err
		}
		if c.WalletBalance.IsEmpty() && assetTotal.IsEmpty() {
			continue
		}
		wallet := c.WalletBalance.ValueOrZero() / wealthMultiplier
		assets := assetTotal.ValueOrZero() / wealthMultiplier
		label := c.EveCharacter.Name
		r := dataRow{
			label:  label,
			assets: assets,
			wallet: wallet,
			total:  assets + wallet,
		}
		data = append(data, r)
	}
	slices.SortFunc(data, func(a, b dataRow) int {
		return strings.Compare(a.label, b.label)
	})
	return data, len(selected), nil
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
		theme.Color(theme.ColorNameSuccess),
		theme.Color(theme.ColorNameError),
		theme.Color(theme.ColorNamePrimary),
		theme.Color(theme.ColorNameWarning),
		theme.Color(colorNameInfo),
		theme.Color(colorNameCreative),
		theme.Color(colorNameSystem),
		theme.Color(colorNameAttention),
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

func (w *colorWheel) reset() {
	w.n = 0
}

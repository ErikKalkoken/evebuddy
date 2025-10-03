package ui

import (
	"cmp"
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-daehling/fyne-charts/pkg/chart"
	chartData "github.com/s-daehling/fyne-charts/pkg/data"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

const (
	chartBaseSize = 440
	chartWidth    = chartBaseSize
	chartHeight   = chartBaseSize / 1.618
)

type wealth struct {
	widget.BaseWidget

	onUpdate func(wallet, assets float64)

	assets     *chart.CartesianCategoricalChart
	characters *chart.PolarProportionalChart
	top        *widget.Label
	types      *chart.PolarProportionalChart
	u          *baseUI
	wallets    *chart.CartesianCategoricalChart
}

func newWealth(u *baseUI) *wealth {
	a := &wealth{
		assets:     chart.NewCartesianCategoricalChart(),
		characters: chart.NewPolarProportionalChart(),
		top:        makeTopLabel(),
		types:      chart.NewPolarProportionalChart(),
		u:          u,
		wallets:    chart.NewCartesianCategoricalChart(),
	}
	a.ExtendBaseWidget(a)

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
	var charts *fyne.Container
	if a.u.isDesktop {
		charts = container.NewGridWithColumns(2,
			a.types,
			a.characters,
			a.wallets,
			a.assets,
		)
	} else {
		charts = container.NewVBox(
			a.types,
			a.characters,
			a.wallets,
			a.assets,
		)
	}
	c := container.NewBorder(
		a.top,
		nil,
		nil,
		nil,
		container.NewScroll(charts),
	)
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
		})
		return
	}
	if characters == 0 {
		fyne.Do(func() {
			a.top.Text = "No characters"
			a.top.Importance = widget.LowImportance
			a.top.Refresh()
		})
		return
	}

	colors := newColorWheel()
	// type
	var totalWallet, totalAssets float64
	for _, r := range data {
		totalAssets += r.assets
		totalWallet += r.wallet
	}
	d1 := []chartData.ProportionalDataPoint{{
		C:   "Assets",
		Val: totalAssets,
		Col: colors.next(),
	}, {
		C:   "Wallets",
		Val: totalWallet,
		Col: colors.next(),
	}}
	fyne.Do(func() {
		a.types.DeleteSeries("Type")
		a.types.SetTitle("Total Wealth By Type")
		_, err = a.types.AddProportionalSeries("Type", d1)
		if err != nil {
			panic(err)
		}
	})

	// characters
	colors.reset()
	d2 := make([]chartData.ProportionalDataPoint, 0)
	for _, r := range data {
		d2 = append(d2, chartData.ProportionalDataPoint{
			C:   r.label,
			Val: r.total,
			Col: colors.next(),
		})
	}
	fyne.Do(func() {
		a.characters.DeleteSeries("Characters")
		a.characters.SetTitle("Total Wealth By Character in B ISK")
		_, err = a.characters.AddProportionalSeries("Characters", d2)
		if err != nil {
			panic(err)
		}
	})

	// Assets
	var m3 float64
	d3 := make([]chartData.CategoricalDataPoint, 0)
	for _, r := range data {
		d3 = append(d3, chartData.CategoricalDataPoint{
			C:   r.label,
			Val: r.assets,
		})
		m3 = max(m3, r.assets)
	}
	fyne.Do(func() {
		a.assets.DeleteSeries("Characters")
		_, err = a.assets.AddBarSeries("Characters", d3, theme.Color(theme.ColorNameSuccess))
		if err != nil {
			panic(err)
		}
		a.assets.SetTitle("Assets Value By Character")
		a.assets.SetYRange(0, m3)
		a.assets.SetYAxisLabel("B ISK")
		a.assets.HideLegend()
	})

	// Wallets
	var m4 float64
	d4 := make([]chartData.CategoricalDataPoint, 0)
	for _, r := range data {
		d4 = append(d4, chartData.CategoricalDataPoint{
			C:   r.label,
			Val: r.wallet,
		})
		m4 = max(m4, r.wallet)
	}
	fyne.Do(func() {
		a.wallets.DeleteSeries("Characters")
		_, err = a.wallets.AddBarSeries("Characters", d4, theme.Color(theme.ColorNameSuccess))
		if err != nil {
			panic(err)
		}
		a.wallets.SetTitle("Wallet Balance By Character")
		a.wallets.SetYRange(0, m4)
		a.wallets.SetYAxisLabel("B ISK")
		a.wallets.HideLegend()
	})

	var total float64
	for _, r := range data {
		total += r.assets + r.wallet
	}

	totalText := ihumanize.Number(total, 1)

	fyne.Do(func() {
		a.top.Text = fmt.Sprintf("%s ISK total wealth • %d characters", totalText, characters)
		a.top.Importance = widget.MediumImportance
		a.top.Refresh()
	})

	if a.onUpdate != nil {
		a.onUpdate(totalWallet, totalAssets)
	}
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
	const b = 1_000_000_000
	data := make([]dataRow, 0)
	for _, c := range selected {
		assetTotal, err := s.cs.AssetTotalValue(ctx, c.ID)
		if err != nil {
			return nil, 0, err
		}
		if c.WalletBalance.IsEmpty() && assetTotal.IsEmpty() {
			continue
		}
		wallet := c.WalletBalance.ValueOrZero() / b
		assets := assetTotal.ValueOrZero() / b
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
		return cmp.Compare(a.total, b.total) * -1
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
	colorNames := []fyne.ThemeColorName{
		theme.ColorNameSuccess,
		theme.ColorNameError,
		theme.ColorNameWarning,
		theme.ColorNamePrimary,
		theme.ColorNameDisabled,
	}
	for _, n := range colorNames {
		w.colors = append(w.colors, theme.Color(n))
	}
	return w
}

func (w *colorWheel) next() color.Color {
	c := w.colors[w.n]
	if w.n < len(w.colors) {
		w.n++
	} else {
		w.n = 0
	}
	return c
}

func (w *colorWheel) reset() {
	w.n = 0
}

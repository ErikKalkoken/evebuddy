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
	"github.com/s-daehling/fyne-charts/pkg/coord"
	chartData "github.com/s-daehling/fyne-charts/pkg/data"
	"github.com/s-daehling/fyne-charts/pkg/prop"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

type wealth struct {
	widget.BaseWidget

	onUpdate func(wallet, assets float64)

	breakdown  *coord.CartesianCategoricalChart
	characters *prop.PieChart
	top        *widget.Label
	types      *prop.PieChart
	u          *baseUI
}

func newWealth(u *baseUI) *wealth {
	a := &wealth{
		breakdown:  coord.NewCartesianCategoricalChart("Wealth Breakdown By Character in B ISK"),
		characters: prop.NewPieChart("Total Wealth By Character in B ISK"),
		top:        makeTopLabel(),
		types:      prop.NewPieChart("Total Wealth By Type"),
		u:          u,
	}
	a.ExtendBaseWidget(a)
	a.breakdown.SetTitleStyle(theme.SizeNameSubHeadingText, theme.ColorNameForeground)
	a.characters.SetTitleStyle(theme.SizeNameSubHeadingText, theme.ColorNameForeground)
	a.types.SetTitleStyle(theme.SizeNameSubHeadingText, theme.ColorNameForeground)

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
	var total *fyne.Container
	if a.u.isDesktop {
		total = container.NewGridWithColumns(2, a.types, a.characters)
	} else {
		total = container.NewAdaptiveGrid(2, a.types, a.characters)
	}
	charts := container.NewAppTabs(
		container.NewTabItem("Total", total),
		container.NewTabItem("Breakdown", a.breakdown),
	)
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
	s1, err := prop.NewSeries("Type", []chartData.ProportionalPoint{{
		C:   "Assets",
		Val: totalAssets,
		Col: colors.next(),
	}, {
		C:   "Wallets",
		Val: totalWallet,
		Col: colors.next(),
	}})
	if err != nil {
		panic(err)
	}
	fyne.Do(func() {
		a.types.RemoveSeries("Type")
		err = a.types.AddSeries(s1)
		if err != nil {
			panic(err)
		}
	})

	// characters
	colors.reset()
	d1 := make([]chartData.ProportionalPoint, 0)
	for _, r := range data {
		d1 = append(d1, chartData.ProportionalPoint{
			C:   r.label,
			Val: r.total,
			Col: colors.next(),
		})
	}
	s2, err := prop.NewSeries("Characters", d1)
	fyne.Do(func() {
		a.characters.RemoveSeries("Characters")
		err = a.characters.AddSeries(s2)
		if err != nil {
			panic(err)
		}
	})

	// Breakdown
	d3 := make([]chartData.CategoricalPoint, 0)
	for _, r := range data {
		d3 = append(d3, chartData.CategoricalPoint{
			C:   r.label,
			Val: r.assets,
		})
	}
	s3, err := coord.NewCategoricalPointSeries("Assets", theme.Color(theme.ColorNameSuccess), d3)
	if err != nil {
		panic(err)
	}
	d4 := make([]chartData.CategoricalPoint, 0)
	for _, r := range data {
		d4 = append(d4, chartData.CategoricalPoint{
			C:   r.label,
			Val: r.wallet,
		})
	}
	s4, err := coord.NewCategoricalPointSeries("Wallets", theme.Color(theme.ColorNameError), d4)
	if err != nil {
		panic(err)
	}

	fyne.Do(func() {
		a.breakdown.RemoveSeries("Assets")
		a.breakdown.RemoveSeries("Wallets")
		err = a.breakdown.AddBarSeries(s3)
		if err != nil {
			panic(err)
		}
		err = a.breakdown.AddBarSeries(s4)
		if err != nil {
			panic(err)
		}
		a.breakdown.SetYAxisLabel("B ISK")
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

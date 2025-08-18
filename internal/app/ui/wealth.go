package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/golang/freetype/truetype"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/chartbuilder"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	chartBaseSize = 440
	chartWidth    = chartBaseSize
	chartHeight   = chartBaseSize / 1.618
)

type wealth struct {
	widget.BaseWidget

	onUpdate func(wallet, assets float64)

	charts *fyne.Container
	top    *widget.Label
	u      *baseUI
}

func newWealth(u *baseUI) *wealth {
	a := &wealth{
		top: makeTopLabel(),
		u:   u,
	}
	a.ExtendBaseWidget(a)
	a.charts = a.makeCharts()

	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		switch arg.Section {
		case app.SectionCharacterAssets, app.SectionCharacterWalletBalance:
			a.update()
		}
	})
	a.u.generalSectionChanged.AddListener(func(_ context.Context, arg generalSectionUpdated) {
		if arg.Section == app.SectionEveMarketPrices {
			a.update()
		}
	})

	return a
}

func (a *wealth) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		a.top,
		nil,
		nil,
		nil,
		container.NewScroll(a.charts),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *wealth) makeCharts() *fyne.Container {
	makePlaceholder := func() fyne.CanvasObject {
		x := iwidget.NewImageFromResource(theme.BrokenImageIcon(), fyne.NewSize(chartWidth, chartHeight))
		return container.NewPadded(x)
	}
	c := container.NewGridWrap(
		fyne.NewSize(chartWidth, chartHeight),
		makePlaceholder(),
		makePlaceholder(),
		makePlaceholder(),
		makePlaceholder(),
	)
	return c
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
	cb := chartbuilder.New(a.u.MainWindow())
	cb.ForegroundColor = theme.Color(theme.ColorNameForeground)
	cb.BackgroundColor = theme.Color(theme.ColorNameBackground)
	f := theme.DefaultTextFont().Content()
	font, err := truetype.Parse(f)
	if err != nil {
		slog.Error("Failed to initialize TTF", "error", err)
	} else {
		cb.Font = font
	}
	charactersData := make([]chartbuilder.Value, len(data))
	for i, r := range data {
		charactersData[i] = chartbuilder.Value{Label: r.label, Value: r.assets + r.wallet}
	}
	pieChartSize := fyne.NewSize(chartWidth, chartHeight)
	charactersChart := cb.Render(chartbuilder.Pie, pieChartSize, "Total Wealth By Character", charactersData)

	var totalWallet, totalAssets float64
	for _, r := range data {
		totalAssets += r.assets
		totalWallet += r.wallet
	}
	typesData := make([]chartbuilder.Value, 2)
	typesData[0] = chartbuilder.Value{Label: "Assets", Value: totalAssets}
	typesData[1] = chartbuilder.Value{Label: "Wallet", Value: totalWallet}
	typesChart := cb.Render(chartbuilder.Pie, pieChartSize, "Total Wealth By Type", typesData)

	barChartSize := fyne.NewSize(chartWidth, chartHeight)
	assetsData := make([]chartbuilder.Value, len(data))
	for i, r := range data {
		assetsData[i] = chartbuilder.Value{Label: r.label, Value: r.assets}
	}
	assetsChart := cb.Render(chartbuilder.Bar, barChartSize, "Assets Value By Character", assetsData)

	walletData := make([]chartbuilder.Value, len(data))
	for i, r := range data {
		walletData[i] = chartbuilder.Value{Label: r.label, Value: r.wallet}
	}
	walletChart := cb.Render(chartbuilder.Bar, barChartSize, "Wallet Balance By Character", walletData)

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
	fyne.Do(func() {
		charts := a.charts.Objects
		charts[0] = typesChart
		charts[1] = charactersChart
		charts[2] = assetsChart
		charts[3] = walletChart
		a.charts.Refresh()
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
	data := make([]dataRow, 0)
	for _, c := range selected {
		assetTotal, err := s.cs.AssetTotalValue(ctx, c.ID)
		if err != nil {
			return nil, 0, err
		}
		if c.WalletBalance.IsEmpty() && assetTotal.IsEmpty() {
			continue
		}
		wallet := c.WalletBalance.ValueOrZero()
		assets := assetTotal.ValueOrZero()
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

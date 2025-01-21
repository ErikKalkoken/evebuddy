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
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/chartbuilder"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/golang/freetype/truetype"
)

const (
	chartBaseSize  = 300
	pieChartWidth  = chartBaseSize
	pieChartHeight = chartBaseSize
	barChartWidth  = 2 * chartBaseSize
	barChartHeight = chartBaseSize
)

type WealthArea struct {
	Content fyne.CanvasObject
	charts  *fyne.Container
	top     *widget.Label
	u       *BaseUI
}

func (u *BaseUI) NewWealthArea() *WealthArea {
	a := &WealthArea{
		top: widget.NewLabel(""),
		u:   u,
	}
	a.charts = a.makeCharts()
	a.top.TextStyle.Bold = true
	cs := container.NewScroll(a.charts)
	cs.SetMinSize(fyne.NewSize(barChartWidth, barChartHeight))
	a.Content = container.NewBorder(
		container.NewVBox(a.top, widget.NewSeparator()), nil, nil, nil,
		cs,
	)
	return a
}

func (a *WealthArea) makeCharts() *fyne.Container {
	return container.NewVBox(
		container.NewHBox(widget.NewLabel(""), widget.NewLabel("")),
		widget.NewLabel(""),
		widget.NewLabel(""),
	)
}

func (a *WealthArea) Refresh() {
	data, characters, err := a.compileData()
	if err != nil {
		slog.Error("Failed to fetch data for charts", "err", err)
		a.top.Text = fmt.Sprintf("Failed to fetch data for charts: %s", humanize.Error(err))
		a.top.Importance = widget.DangerImportance
		a.top.Refresh()
		return
	}
	if characters == 0 {
		a.top.Text = "No characters"
		a.top.Importance = widget.LowImportance
		a.top.Refresh()
		return
	}
	cb := chartbuilder.New(a.u.Window)
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
	pieChartSize := fyne.NewSize(pieChartWidth, pieChartHeight)
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

	barChartSize := fyne.NewSize(barChartWidth, barChartHeight)
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

	pieCharts := a.charts.Objects[0].(*fyne.Container).Objects
	pieCharts[0] = typesChart
	pieCharts[1] = charactersChart
	a.charts.Objects[1] = assetsChart
	a.charts.Objects[2] = walletChart
	a.charts.Refresh()

	a.top.Text = fmt.Sprintf("%s ISK total wealth • %d characters", humanize.Number(total, 1), characters)
	a.top.Importance = widget.MediumImportance
	a.top.Refresh()
}

type dataRow struct {
	label  string
	wallet float64
	assets float64
	total  float64
}

func (a *WealthArea) compileData() ([]dataRow, int, error) {
	ctx := context.TODO()
	cc, err := a.u.CharacterService.ListCharacters(ctx)
	if err != nil {
		return nil, 0, err
	}
	selected := make([]*app.Character, 0)
	for _, c := range cc {
		hasAssets := a.u.StatusCacheService.CharacterSectionExists(c.ID, app.SectionAssets)
		hasWallet := a.u.StatusCacheService.CharacterSectionExists(c.ID, app.SectionWalletBalance)
		if hasAssets && hasWallet {
			selected = append(selected, c)
		}
	}
	data := make([]dataRow, 0)
	for _, c := range selected {
		assetTotal, err := a.u.CharacterService.CharacterAssetTotalValue(ctx, c.ID)
		if err != nil {
			return nil, 0, err
		}
		if assetTotal.IsEmpty() && a.u.StatusCacheService.CharacterSectionExists(c.ID, app.SectionAssets) {
			go func(characterID int32) {
				_, err := a.u.CharacterService.UpdateCharacterAssetTotalValue(ctx, characterID)
				if err != nil {
					slog.Error("update asset totals", "characterID", characterID, "err", err)
					return
				}
				a.u.WealthArea.Refresh()
				a.u.OverviewArea.Refresh()
			}(c.ID)
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

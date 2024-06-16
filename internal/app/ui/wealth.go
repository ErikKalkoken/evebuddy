package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/charts"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
)

type wealthArea struct {
	content fyne.CanvasObject
	charts  *fyne.Container
	top     *widget.Label
	ui      *ui

	mu sync.Mutex
}

func (u *ui) newWealthArea() *wealthArea {
	a := &wealthArea{
		ui:     u,
		charts: container.NewVBox(),
		top:    widget.NewLabel(""),
	}
	a.top.TextStyle.Bold = true
	a.content = container.NewBorder(
		container.NewVBox(a.top, widget.NewSeparator()), nil, nil, nil,
		container.NewScroll(a.charts))
	return a
}

type dataRow struct {
	label  string
	wallet float64
	assets float64
	total  float64
}

func (a *wealthArea) refresh() {
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
	cb := charts.NewChartBuilder()
	charactersData := make([]charts.Value, len(data))
	for i, r := range data {
		charactersData[i] = charts.Value{Label: r.label, Value: r.assets + r.wallet}
	}
	charactersChart := cb.Render(charts.Pie, "Total Wealth By Character", charactersData)

	var totalWallet, totalAssets float64
	for _, r := range data {
		totalAssets += r.assets
		totalWallet += r.wallet
	}
	typesData := make([]charts.Value, 2)
	typesData[0] = charts.Value{Label: "Assets", Value: totalAssets}
	typesData[1] = charts.Value{Label: "Wallet", Value: totalWallet}
	typesChart := cb.Render(charts.Pie, "Total Wealth By Type", typesData)

	assetsData := make([]charts.Value, len(data))
	for i, r := range data {
		assetsData[i] = charts.Value{Label: r.label, Value: r.assets}
	}
	assetsChart := cb.Render(charts.Bar, "Assets Value By Character", assetsData)

	walletData := make([]charts.Value, len(data))
	for i, r := range data {
		walletData[i] = charts.Value{Label: r.label, Value: r.wallet}
	}
	walletChart := cb.Render(charts.Bar, "Wallet Balance By Character", walletData)

	var total float64
	for _, r := range data {
		total += r.assets + r.wallet
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.charts.RemoveAll()
	a.charts.Add(typesChart)
	a.charts.Add(charactersChart)
	a.charts.Add(assetsChart)
	a.charts.Add(walletChart)

	a.top.Text = fmt.Sprintf("%s ISK total wealth • %d characters", humanize.Number(total, 1), characters)
	a.top.Importance = widget.MediumImportance
	a.top.Refresh()
}

func (a *wealthArea) compileData() ([]dataRow, int, error) {
	ctx := context.TODO()
	cc, err := a.ui.CharacterService.ListCharacters(ctx)
	if err != nil {
		return nil, 0, err
	}
	selected := make([]*app.Character, 0)
	for _, c := range cc {
		hasAssets := a.ui.StatusCacheService.CharacterSectionExists(c.ID, app.SectionAssets)
		hasWallet := a.ui.StatusCacheService.CharacterSectionExists(c.ID, app.SectionWalletBalance)
		if hasAssets && hasWallet {
			selected = append(selected, c)
		}
	}
	data := make([]dataRow, 0)
	for _, c := range selected {
		wallet := c.WalletBalance.Float64
		x, err := a.ui.CharacterService.CharacterAssetTotalValue(c.ID)
		if err != nil {
			return nil, 0, err
		}
		assets := x.Float64
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

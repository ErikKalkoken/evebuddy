package screens

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/nathabonfim59/fyneline"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

const (
	corporationWealthMultiplier     = 1_000_000_000
	corporationWealthContractsLabel = "Contract Escrow"
)

// corporationDivisionValue holds one division's wallet and asset value, in billions ISK.
type corporationDivisionValue struct {
	division      app.Division
	name          string // hangar/division name, used to label the assets chart
	walletName    string // wallet name, used to label the wallets chart
	walletBalance float64
	assetValue    float64
}

type CorporationWealth struct {
	widget.BaseWidget

	corporation atomic.Pointer[app.Corporation]

	assetsCard              *chartCard
	assetsChart             *fyneline.BarChart[corporationDivisionValue]
	assetsTitleLabel        *widget.Label
	categorySplitCard       *chartCard
	categorySplitChart      *fyneline.ArcChart[namedValue]
	categorySplitTitleLabel *widget.Label
	top                     *widget.Label
	u                       baseUI
	walletsCard             *chartCard
	walletsChart            *fyneline.BarChart[corporationDivisionValue]
	walletsTitleLabel       *widget.Label
}

func NewCorporationWealth(u baseUI) *CorporationWealth {
	sliceLabel := func(v namedValue) string { return fmt.Sprintf("%.1f", v.value) }
	a := &CorporationWealth{
		categorySplitChart: fyneline.NewArcChart([]namedValue(nil),
			func(v namedValue) float64 { return v.value },
			sliceLabel,
		),
		categorySplitTitleLabel: newChartTitleLabel(),
		walletsChart: fyneline.NewBarChart([]corporationDivisionValue(nil),
			func(v corporationDivisionValue) string { return v.walletName },
			fyneline.NewBarSeries("Wallet Balance", func(v corporationDivisionValue) float64 { return v.walletBalance }),
		),
		walletsTitleLabel: newChartTitleLabel(),
		assetsChart: fyneline.NewBarChart([]corporationDivisionValue(nil),
			func(v corporationDivisionValue) string { return v.name },
			fyneline.NewBarSeries("Asset Value", func(v corporationDivisionValue) float64 { return v.assetValue }),
		),
		assetsTitleLabel: newChartTitleLabel(),
		top:              ui.NewLabelWithWrapping(""),
		u:                u,
	}
	a.ExtendBaseWidget(a)
	a.top.Hide()

	configureArcChart(a.categorySplitChart)
	a.walletsChart.SetValueAxis(fyneline.NewNumericAxis().WithFormatter(wealthAxisValueFormatter))
	a.walletsChart.SetOrientation(fyneline.BarHorizontal)
	a.assetsChart.SetValueAxis(fyneline.NewNumericAxis().WithFormatter(wealthAxisValueFormatter))
	a.assetsChart.SetOrientation(fyneline.BarHorizontal)

	categorySplitLegend := newSeriesLegend(
		newLegendEntry("Assets", wealthArcColor(0)),
		newLegendEntry("Wallet", wealthArcColor(1)),
		newLegendEntry(corporationWealthContractsLabel, wealthArcColor(2)),
	)

	a.categorySplitCard = newChartCard(a.categorySplitTitleLabel, categorySplitLegend, a.categorySplitChart)
	a.walletsCard = newChartCard(a.walletsTitleLabel, nil, a.walletsChart)
	a.assetsCard = newChartCard(a.assetsTitleLabel, nil, a.assetsChart)

	// Signals
	a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		a.update(ctx)
	})
	a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		if a.corporation.Load().IDOrZero() != arg.CorporationID {
			return
		}
		switch arg.Section {
		case
			app.SectionCorporationAssets,
			app.SectionCorporationContracts,
			app.SectionCorporationWalletBalances:
			a.update(ctx)
		}
	})
	return a
}

func (a *CorporationWealth) CreateRenderer() fyne.WidgetRenderer {
	tabs := container.NewAppTabs(
		container.NewTabItem("Overview", a.categorySplitCard),
		container.NewTabItem("Wallets", a.walletsCard),
		container.NewTabItem("Assets", a.assetsCard),
	)
	var c fyne.CanvasObject
	if !a.u.IsMobile() {
		c = container.NewBorder(a.top, nil, nil, nil, tabs)
	} else {
		c = tabs
	}
	return widget.NewSimpleRenderer(c)
}

func (a *CorporationWealth) update(ctx context.Context) {
	corporationID := a.corporation.Load().IDOrZero()
	if corporationID == 0 {
		a.showTop("No corporation", widget.LowImportance)
		return
	}
	hasAssets, hasContracts, hasWallet, err := a.hasAnyData(ctx, corporationID)
	if err != nil {
		slog.Error("Failed to fetch data for corporation wealth charts", "corporationID", corporationID, "err", err)
		a.showTop(fmt.Sprintf("Failed to fetch data for charts: %s", a.u.ErrorDisplay(err)), widget.DangerImportance)
		return
	}
	if !hasAssets && !hasContracts && !hasWallet {
		a.showTop("Waiting for data to be loaded...", widget.WarningImportance)
		return
	}

	var totalAssetValue, contractsEscrow float64
	assetByDivision := make(map[app.Division]float64)
	if hasAssets {
		v, err := a.u.Corporation().CalculateAssetTotalValue(ctx, corporationID)
		if err != nil {
			slog.Error("Failed to fetch corporation asset value", "corporationID", corporationID, "err", err)
		} else {
			totalAssetValue = v
		}
		m, err := a.u.Corporation().CalculateAssetValueByDivision(ctx, corporationID)
		if err != nil {
			slog.Error("Failed to fetch corporation asset value by division", "corporationID", corporationID, "err", err)
		} else {
			assetByDivision = m
		}
	}
	if hasContracts {
		v, err := a.u.Corporation().CalculateContractsEscrow(ctx, corporationID)
		if err != nil {
			slog.Error("Failed to fetch corporation contracts escrow", "corporationID", corporationID, "err", err)
		} else {
			contractsEscrow = v
		}
	}
	var walletBalances []app.CorporationWalletBalanceWithName
	if hasWallet {
		wb, err := a.u.Corporation().ListWalletBalances(ctx, corporationID)
		if err != nil {
			slog.Error("Failed to fetch corporation wallet balances", "corporationID", corporationID, "err", err)
		} else {
			walletBalances = wb
		}
	}

	rows := make([]corporationDivisionValue, 0, len(app.Divisions))
	var totalWalletBalance float64
	walletByDivision := make(map[app.Division]corporationWalletEntry)
	for _, wb := range walletBalances {
		walletByDivision[app.Division(wb.DivisionID)] = corporationWalletEntry{name: wb.Name, balance: wb.Balance}
		totalWalletBalance += wb.Balance
	}
	hangarNames := a.u.Corporation().ListHangarNames(ctx, corporationID)
	for _, d := range app.Divisions {
		wallet := walletByDivision[d]
		name := hangarNames[d]
		if name == "" {
			name = d.DefaultHangarName()
		}
		walletName := wallet.name
		if walletName == "" {
			walletName = d.DefaultWalletName()
		}
		assetValue := assetByDivision[d]
		rows = append(rows, corporationDivisionValue{
			division:      d,
			name:          name,
			walletName:    walletName,
			walletBalance: wallet.balance / corporationWealthMultiplier,
			assetValue:    assetValue / corporationWealthMultiplier,
		})
	}

	a.showTop("", widget.LowImportance)

	a.updateWallets(rows)
	a.updateAssets(rows)
	a.updateCategorySplit(totalAssetValue, totalWalletBalance, contractsEscrow)
}

type corporationWalletEntry struct {
	name    string
	balance float64
}

// hasAnyData reports whether the corporation has synced data for any of the sections this screen uses.
func (a *CorporationWealth) hasAnyData(ctx context.Context, corporationID int64) (hasAssets, hasContracts, hasWallet bool, err error) {
	hasAssets, err = a.u.Corporation().HasSection(ctx, corporationID, app.SectionCorporationAssets)
	if err != nil {
		return false, false, false, err
	}
	hasContracts, err = a.u.Corporation().HasSection(ctx, corporationID, app.SectionCorporationContracts)
	if err != nil {
		return false, false, false, err
	}
	hasWallet, err = a.u.Corporation().HasSection(ctx, corporationID, app.SectionCorporationWalletBalances)
	if err != nil {
		return false, false, false, err
	}
	return hasAssets, hasContracts, hasWallet, nil
}

func (a *CorporationWealth) showTop(text string, importance widget.Importance) {
	fyne.Do(func() {
		if text == "" {
			a.top.Hide()
			return
		}
		a.top.Text = text
		a.top.Importance = importance
		a.top.Refresh()
		a.top.Show()
	})
}

func (a *CorporationWealth) updateWallets(rows []corporationDivisionValue) {
	var maxValue float64
	for _, v := range rows {
		maxValue = max(maxValue, v.walletBalance)
	}
	axisMax, tickCount := niceAxisBounds(maxValue, 5)
	fyne.Do(func() {
		a.walletsChart.SetValueAxis(fyneline.NewNumericAxis().
			WithFormatter(wealthAxisValueFormatter).
			WithDomain(0, axisMax).
			WithTickCount(tickCount))
		a.walletsChart.SetData(rows)
		a.walletsTitleLabel.SetText("Wallet Balance by Division")
	})
}

func (a *CorporationWealth) updateAssets(rows []corporationDivisionValue) {
	var maxValue float64
	for _, v := range rows {
		maxValue = max(maxValue, v.assetValue)
	}
	axisMax, tickCount := niceAxisBounds(maxValue, 5)
	fyne.Do(func() {
		a.assetsChart.SetValueAxis(fyneline.NewNumericAxis().
			WithFormatter(wealthAxisValueFormatter).
			WithDomain(0, axisMax).
			WithTickCount(tickCount))
		a.assetsChart.SetData(rows)
		a.assetsTitleLabel.SetText("Asset Value by Division")
	})
}

func (a *CorporationWealth) updateCategorySplit(totalAssetValue, totalWalletBalance, contractsEscrow float64) {
	assetsB := totalAssetValue / corporationWealthMultiplier
	walletB := totalWalletBalance / corporationWealthMultiplier
	contractsB := contractsEscrow / corporationWealthMultiplier
	total := assetsB + walletB + contractsB
	d := []namedValue{
		{name: "Assets", value: assetsB},
		{name: "Wallet", value: walletB},
		{name: corporationWealthContractsLabel, value: contractsB},
	}
	fyne.Do(func() {
		a.categorySplitChart.SetData(d)
		a.categorySplitTitleLabel.SetText(fmt.Sprintf("Total Net Worth By Category - Total: %.1f B", total))
	})
}

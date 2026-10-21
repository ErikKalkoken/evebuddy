package screens

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/nathabonfim59/fyneline"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	corporationWealthMultiplier         = 1_000_000_000
	corporationWealthContractsLabel     = "Contract Escrow"
	corporationWealthContractItemsLabel = "Contracts"
	corporationWealthOtherAssetsLabel   = "Other"
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

	OnUpdate func(totalNetWorth optional.Optional[float64])

	corporation atomic.Pointer[app.Corporation]

	assetsCard              *chartCard
	assetsChart             *fyneline.BarChart[corporationDivisionValue]
	assetsTitleLabel        *widget.Label
	categorySplitCard       *chartCard
	categorySplitChart      *fyneline.ArcChart[namedValue]
	categorySplitTitleLabel *widget.Label
	helpButton              *xwidget.Button
	statusLabel             *widget.Label
	u                       baseUI
	updateRun               xsync.LatestRun
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
		statusLabel:      ui.NewLabelWithWrapping(""),
		u:                u,
	}
	a.ExtendBaseWidget(a)
	a.statusLabel.Hide()

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

	a.helpButton = xwidget.NewIconButton(theme.QuestionIcon(), func() {
		showHelpPopUp(corporationWealthHelpText, a.u.IsMobile(), a.helpButton)
	})
	a.helpButton.SetToolTip("Show explanation for values")

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
			app.SectionCorporationDivisions,
			app.SectionCorporationWalletBalances:
			a.update(ctx)
		}
	})
	a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		if arg.Section == app.SectionEveMarketPrices {
			a.update(ctx)
		}
	})
	return a
}

func (a *CorporationWealth) CreateRenderer() fyne.WidgetRenderer {
	tabs := container.NewAppTabs(
		ui.NewTabItem("Overview", a.categorySplitCard),
		ui.NewTabItem("Wallets", a.walletsCard),
		ui.NewTabItem("Assets", a.assetsCard),
	)
	c := container.NewBorder(
		nil,
		container.NewVBox(a.statusLabel, container.NewHBox(layout.NewSpacer(), a.helpButton)),
		nil,
		nil,
		tabs,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *CorporationWealth) update(ctx context.Context) {
	isLatest := a.updateRun.Start()
	reset := func(text string, importance widget.Importance) {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.categorySplitChart.SetData(nil)
			a.categorySplitTitleLabel.SetText("")
			a.walletsChart.SetValueAxis(fyneline.NewNumericAxis().WithFormatter(wealthAxisValueFormatter))
			a.walletsChart.SetData(nil)
			a.walletsTitleLabel.SetText("")
			a.assetsChart.SetValueAxis(fyneline.NewNumericAxis().WithFormatter(wealthAxisValueFormatter))
			a.assetsChart.SetData(nil)
			a.assetsTitleLabel.SetText("")
			a.showStatus(text, importance)
			if a.OnUpdate != nil {
				a.OnUpdate(optional.Optional[float64]{})
			}
		})
	}
	corporationID := a.corporation.Load().IDOrZero()
	if corporationID == 0 {
		reset("No corporation", widget.LowImportance)
		return
	}
	hasAssets, hasContracts, hasWallet, err := a.hasAnyData(ctx, corporationID)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to fetch data for corporation wealth charts", "corporationID", corporationID, "err", err)
		reset(fmt.Sprintf("Failed to fetch data for charts: %s", a.u.ErrorDisplay(err)), widget.DangerImportance)
		return
	}
	if !hasAssets && !hasContracts && !hasWallet {
		reset("Waiting for data to be loaded...", widget.WarningImportance)
		return
	}

	v, err := a.fetchValues(ctx, corporationID, hasAssets, hasContracts, hasWallet)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to fetch values for corporation wealth charts", "corporationID", corporationID, "err", err)
		reset(fmt.Sprintf("Failed to fetch data for charts: %s", a.u.ErrorDisplay(err)), widget.DangerImportance)
		return
	}

	rows := make([]corporationDivisionValue, 0, len(app.Divisions))
	var totalWalletBalance float64
	var divisionAssetValue float64
	walletByDivision := make(map[app.Division]corporationWalletEntry)
	for _, wb := range v.walletBalances {
		walletByDivision[app.Division(wb.DivisionID)] = corporationWalletEntry{name: wb.Name, balance: wb.Balance}
		totalWalletBalance += wb.Balance
	}
	for _, d := range app.Divisions {
		wallet := walletByDivision[d]
		assetValue := v.assetByDivision[d]
		divisionAssetValue += assetValue
		rows = append(rows, corporationDivisionValue{
			division:      d,
			name:          corporationHangarLabel(d, v.hangarNames[d]),
			walletName:    corporationWalletLabel(d, wallet.name),
			walletBalance: wallet.balance / corporationWealthMultiplier,
			assetValue:    assetValue / corporationWealthMultiplier,
		})
	}

	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.showStatus("", widget.LowImportance)
		a.updateWallets(rows)
		a.updateAssets(corporationAssetRows(rows, v.totalAssetValue-divisionAssetValue, v.contractItemsValue))
		combinedAssets := v.totalAssetValue + v.contractItemsValue
		a.updateCategorySplit(combinedAssets, totalWalletBalance, v.contractsEscrow)
		if a.OnUpdate != nil {
			a.OnUpdate(optional.New(combinedAssets + totalWalletBalance + v.contractsEscrow))
		}
	})
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

type corporationWealthValues struct {
	assetByDivision    map[app.Division]float64
	contractItemsValue float64
	contractsEscrow    float64
	hangarNames        map[app.Division]string
	totalAssetValue    float64
	walletBalances     []app.CorporationWalletBalanceWithName
}

// fetchValues fetches the values for the sections the corporation has data for.
func (a *CorporationWealth) fetchValues(ctx context.Context, corporationID int64, hasAssets, hasContracts, hasWallet bool) (corporationWealthValues, error) {
	var v corporationWealthValues
	var err error
	if hasAssets {
		v.totalAssetValue, err = a.u.Corporation().CalculateAssetTotalValue(ctx, corporationID)
		if err != nil {
			return v, fmt.Errorf("asset total value: %w", err)
		}
		v.assetByDivision, err = a.u.Corporation().CalculateAssetValueByDivision(ctx, corporationID)
		if err != nil {
			return v, fmt.Errorf("asset value by division: %w", err)
		}
	}
	if hasContracts {
		v.contractsEscrow, err = a.u.Corporation().CalculateContractsEscrow(ctx, corporationID)
		if err != nil {
			return v, fmt.Errorf("contracts escrow: %w", err)
		}
		v.contractItemsValue, err = a.u.Corporation().CalculateContractItemsValue(ctx, corporationID)
		if err != nil {
			return v, fmt.Errorf("contract items value: %w", err)
		}
	}
	if hasWallet {
		v.walletBalances, err = a.u.Corporation().ListWalletBalances(ctx, corporationID)
		if err != nil {
			return v, fmt.Errorf("wallet balances: %w", err)
		}
	}
	v.hangarNames, err = a.u.Corporation().ListHangarNames(ctx, corporationID)
	if err != nil {
		return v, fmt.Errorf("hangar names: %w", err)
	}
	return v, nil
}

func (a *CorporationWealth) showStatus(text string, importance widget.Importance) {
	if text == "" {
		a.statusLabel.Hide()
		return
	}
	a.statusLabel.Text = text
	a.statusLabel.Importance = importance
	a.statusLabel.Refresh()
	a.statusLabel.Show()
}

func (a *CorporationWealth) updateWallets(rows []corporationDivisionValue) {
	var maxValue float64
	for _, v := range rows {
		maxValue = max(maxValue, v.walletBalance)
	}
	axisMax, tickCount := niceAxisBounds(maxValue, 5)
	a.walletsChart.SetValueAxis(fyneline.NewNumericAxis().
		WithFormatter(wealthAxisValueFormatter).
		WithDomain(0, axisMax).
		WithTickCount(tickCount))
	a.walletsChart.SetData(rows)
	a.walletsTitleLabel.SetText("Wallet Balance by Division")
}

// corporationHangarLabel returns the display label for a hangar division, e.g. "1st Division [1]".
func corporationHangarLabel(d app.Division, name string) string {
	if name == "" {
		name = d.DefaultHangarName()
	}
	return fmt.Sprintf("%s [%d]", name, d)
}

// corporationAssetRows returns the rows for the assets chart.
// Assets outside hangar divisions and contract items get their own bars,
// so the chart adds up to the Assets slice of the overview.
func corporationAssetRows(rows []corporationDivisionValue, otherValue, contractItemsValue float64) []corporationDivisionValue {
	rows = slices.Clone(rows)
	// Ignores float rounding differences between the asset total and the division sum.
	if otherValue >= 1 {
		rows = append(rows, corporationDivisionValue{
			name:       corporationWealthOtherAssetsLabel,
			assetValue: otherValue / corporationWealthMultiplier,
		})
	}
	if contractItemsValue > 0 {
		rows = append(rows, corporationDivisionValue{
			name:       corporationWealthContractItemsLabel,
			assetValue: contractItemsValue / corporationWealthMultiplier,
		})
	}
	return rows
}

func (a *CorporationWealth) updateAssets(rows []corporationDivisionValue) {
	var maxValue float64
	for _, v := range rows {
		maxValue = max(maxValue, v.assetValue)
	}
	axisMax, tickCount := niceAxisBounds(maxValue, 5)
	a.assetsChart.SetValueAxis(fyneline.NewNumericAxis().
		WithFormatter(wealthAxisValueFormatter).
		WithDomain(0, axisMax).
		WithTickCount(tickCount))
	a.assetsChart.SetData(rows)
	a.assetsTitleLabel.SetText("Asset Value by Division")
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
	a.categorySplitChart.SetData(d)
	a.categorySplitTitleLabel.SetText(fmt.Sprintf("Total Net Worth By Category - Total: %.1f B", total))
}

const corporationWealthHelpText = `Wallet: The total balance of all division wallets.

Assets: The estimated value of all corporation assets and of items in outstanding auction and item exchange contracts issued for the corporation. On the Assets tab, assets outside hangar divisions and contract items are shown as separate "Other" and "Contracts" bars.

Contract Escrow: Collateral of courier contracts the corporation accepted, plus ISK rewards of courier and item exchange contracts issued for the corporation, while they are open. Bids on auctions are not included.

Total Net Worth: Sum of Wallet, Assets and Contract Escrow.

Values in the charts are shown in billion ISK.`

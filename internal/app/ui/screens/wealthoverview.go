package screens

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type wealthOverviewRow struct {
	characterID            int64
	characterName          string
	combinedAssetsDisplay  string
	combinedAssetsValue    optional.Optional[float64]
	contractsEscrow        optional.Optional[float64]
	contractsEscrowDisplay string
	isTotal                bool
	ordersEscrow           optional.Optional[float64]
	ordersEscrowDisplay    string
	searchTarget           string
	skillPoints            optional.Optional[float64]
	skillPointsDisplay     string
	tags                   set.Set[string]
	tagsDisplay            string
	totalNetWorth          optional.Optional[float64]
	totalNetWorthDisplay   string
	walletBalance          optional.Optional[float64]
	walletDisplay          string
}

func (r wealthOverviewRow) eveEntity() *app.EveEntity {
	return &app.EveEntity{
		Category: app.EveEntityCharacter,
		ID:       r.characterID,
		Name:     r.characterName,
	}
}

type WealthOverview struct {
	widget.BaseWidget

	OnUpdate func(expired int)

	footer       *widget.Label
	columnSorter *xwidget.ColumnSorter[wealthOverviewRow]
	main         fyne.CanvasObject
	rows         []wealthOverviewRow
	rowsFiltered []wealthOverviewRow
	searchEntry  *xwidget.SearchEntry
	selectTag    *kxwidget.FilterChipSelect
	sortChip     *kxwidget.SortChip
	u            baseUI
	showHelp     *xwidget.IconButton
}

const valueWidth = 125

const overviewHelpText = `Wallet Balance: The balance of the wallet.

Combined Assets: The estimated value of all personal assets, items in outstanding sell orders on the market and items in outstanding contracts.

Contract Escrow: The total sum of all escrows in a character's currently outstanding contracts.

Orders Escrow: The total sum of all escrows in a character's currently outstanding buy orders on the market.

Total Net Worth: Sum of Wallet Balance, Combined Assets, Contract Escrow and Orders Escrow.

Skill Points: Value of extracted skill points (trained + unallocated) calculated with: (MarketPriceOfLargeSkillInjector − MarketPriceOfSkillExtractor) x ((TotalSP − 5,000,000) / 500,000)

NOTE: Blueprints, PLEX in the account wallet are not included.`

func NewWealthOverview(u baseUI) *WealthOverview {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[wealthOverviewRow]{
		ui.MakeEveEntityColumn(ui.MakeEveEntityColumnParams[wealthOverviewRow]{
			EIS: u.EVEImage(),
			GetEntity: func(r wealthOverviewRow) *app.EveEntity {
				return &app.EveEntity{
					ID:       r.characterID,
					Name:     r.characterName,
					Category: app.EveEntityCharacter,
				}
			},
			IsAvatar: true,
			Label:    "Character",
		}), {
			Label: "Tags",
			Width: 150,
			Update: func(r wealthOverviewRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.tagsDisplay)
			},
		}, {
			Label: "Wallet Balance",
			Width: valueWidth,
			Update: func(r wealthOverviewRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.walletDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b wealthOverviewRow) int {
				return optional.Compare(a.walletBalance, b.walletBalance)
			},
		}, {
			Label: "Combined Assets",
			Width: valueWidth,
			Update: func(r wealthOverviewRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.combinedAssetsDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b wealthOverviewRow) int {
				return optional.Compare(a.combinedAssetsValue, b.combinedAssetsValue)
			},
		},
		{
			Label: "Contracts Escrow",
			Width: valueWidth,
			Update: func(r wealthOverviewRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.contractsEscrowDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b wealthOverviewRow) int {
				return optional.Compare(a.contractsEscrow, b.contractsEscrow)
			},
		},
		{
			Label: "Orders Escrow",
			Width: valueWidth,
			Update: func(r wealthOverviewRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.ordersEscrowDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b wealthOverviewRow) int {
				return optional.Compare(a.ordersEscrow, b.ordersEscrow)
			},
		}, {
			Label: "Total Net Worth",
			Width: valueWidth,
			Update: func(r wealthOverviewRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.totalNetWorthDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b wealthOverviewRow) int {
				return optional.Compare(a.totalNetWorth, b.totalNetWorth)
			},
		}, {
			Label: "Skill Points",
			Width: valueWidth,
			Update: func(r wealthOverviewRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.skillPointsDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b wealthOverviewRow) int {
				return optional.Compare(a.skillPoints, b.skillPoints)
			},
		},
	})
	a := &WealthOverview{
		columnSorter: xwidget.NewColumnSorter(columns, "Character", xwidget.SortAsc),
		footer:       widget.NewLabel(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	a.searchEntry = xwidget.NewSearchEntry("Search characters", func(_ string) {
		a.filterRowsAsync(-1)
	})

	a.showHelp = xwidget.NewIconButton(theme.QuestionIcon(), func() {
		showHelpPopUp(overviewHelpText, a.u.IsMobile(), a.showHelp)
	})
	a.showHelp.SetToolTip("Show explanation for columns")

	showRow := func(r wealthOverviewRow) {
		o := r.eveEntity()
		if o.ID == 0 {
			return
		}
		u.InfoViewer().Show(o)
	}

	if a.u.IsMobile() {
		a.main = xwidget.MakeDataList(
			columns,
			&a.rowsFiltered,
			func(col string, r wealthOverviewRow) []widget.RichTextSegment {
				var s []widget.RichTextSegment
				switch col {
				case "Character":
					s = xwidget.RichTextSegmentsFromText(r.characterName)
				case "Tags":
					s = xwidget.RichTextSegmentsFromText(r.tagsDisplay)
				case "Wallet Balance":
					s = xwidget.RichTextSegmentsFromText(r.walletDisplay)
				case "Combined Assets":
					s = xwidget.RichTextSegmentsFromText(r.combinedAssetsDisplay)
				case "Contracts Escrow":
					s = xwidget.RichTextSegmentsFromText(r.contractsEscrowDisplay)
				case "Orders Escrow":
					s = xwidget.RichTextSegmentsFromText(r.ordersEscrowDisplay)
				case "Total Net Worth":
					s = xwidget.RichTextSegmentsFromText(r.totalNetWorthDisplay)
				case "Skill Points":
					s = xwidget.RichTextSegmentsFromText(r.skillPointsDisplay)
				}
				return s
			},
			showRow,
		)
	} else {
		a.main = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync,
			func(_ int, r wealthOverviewRow) {
				showRow(r)
			},
		)
	}
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync(-1)
	})

	// Signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		switch arg.Section {
		case
			app.SectionCharacterAssets,
			app.SectionCharacterContracts,
			app.SectionCharacterMarketOrders,
			app.SectionCharacterSkills,
			app.SectionCharacterWalletBalance:
			a.update(ctx)
		}
	})
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	a.u.Signals().CharacterChanged.AddListener(func(ctx context.Context, _ int64) {
		a.update(ctx)
	})
	return a
}

func (a *WealthOverview) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectTag)
	if a.u.IsMobile() {
		filter.Add(a.sortChip)
	}
	var topBox *fyne.Container
	if a.u.IsMobile() {
		topBox = container.NewVBox(
			a.searchEntry,
			container.NewHScroll(filter),
		)
	} else {
		topBox = container.NewBorder(nil, nil, filter, nil, a.searchEntry)
	}
	c := container.NewBorder(
		topBox,
		container.NewHBox(a.footer, layout.NewSpacer(), a.showHelp),
		nil,
		nil,
		a.main,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *WealthOverview) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	selectTag := a.selectTag.Selected
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		if selectTag != "" {
			rows = slices.DeleteFunc(rows, func(r wealthOverviewRow) bool {
				return !r.tags.Contains(selectTag)
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r wealthOverviewRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r wealthOverviewRow) set.Set[string] {
			return r.tags
		})...).All())

		footer := fmt.Sprintf("Showing %d / %d characters", len(rows), totalRows)

		// add totals
		var assets, wallets, totals, contracts, orders, skillpoints []optional.Optional[float64]
		for _, r := range rows {
			assets = append(assets, r.combinedAssetsValue)
			contracts = append(contracts, r.contractsEscrow)
			orders = append(orders, r.ordersEscrow)
			skillpoints = append(skillpoints, r.skillPoints)
			totals = append(totals, r.totalNetWorth)
			wallets = append(wallets, r.walletBalance)
		}
		assetsTotal := optional.Sum(assets...)
		contractsTotal := optional.Sum(contracts...)
		grandTotal1 := optional.Sum(totals...)
		ordersTotal := optional.Sum(orders...)
		walletsTotal := optional.Sum(wallets...)
		skillpointsTotal := optional.Sum(skillpoints...)
		rows = append(rows, wealthOverviewRow{
			characterID:            0,
			characterName:          "TOTAL",
			combinedAssetsDisplay:  formatISKValue(assetsTotal),
			combinedAssetsValue:    assetsTotal,
			contractsEscrow:        contractsTotal,
			contractsEscrowDisplay: formatISKValue(contractsTotal),
			isTotal:                true,
			ordersEscrow:           ordersTotal,
			ordersEscrowDisplay:    formatISKValue(ordersTotal),
			searchTarget:           "",
			skillPoints:            skillpointsTotal,
			skillPointsDisplay:     formatISKValue(skillpointsTotal),
			tags:                   set.Set[string]{},
			totalNetWorth:          grandTotal1,
			totalNetWorthDisplay:   formatISKValue(grandTotal1),
			walletBalance:          walletsTotal,
			walletDisplay:          formatISKValue(walletsTotal),
		})

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectTag.SetOptions(tagOptions)
			a.rowsFiltered = rows
			a.main.Refresh()
		})
	}()
}

func (a *WealthOverview) update(ctx context.Context) {
	rows, err := a.fetchRows(ctx)
	if err != nil {
		slog.Error("Failed to refresh wealth overview UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + a.u.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync(-1)
	})
}

func (a *WealthOverview) fetchRows(ctx context.Context) ([]wealthOverviewRow, error) {
	cc, err := a.u.Character().ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	var rows []wealthOverviewRow
	for _, c := range cc {
		tags, err := a.u.Character().ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return rows, err
		}

		combinedAssets := c.CombinedAssetsValue()
		total := optional.Sum(combinedAssets, c.WalletBalance, c.ContractsEscrow, c.OrdersEscrow)
		rows = append(rows, wealthOverviewRow{
			characterID:            c.ID,
			characterName:          c.EveCharacter.Name,
			combinedAssetsDisplay:  formatISKValue(combinedAssets),
			combinedAssetsValue:    combinedAssets,
			contractsEscrow:        c.ContractsEscrow,
			contractsEscrowDisplay: formatISKValue(c.ContractsEscrow),
			ordersEscrow:           c.OrdersEscrow,
			ordersEscrowDisplay:    formatISKValue(c.OrdersEscrow),
			searchTarget:           strings.ToLower(c.EveCharacter.Name),
			skillPoints:            c.SkillPointsValue,
			skillPointsDisplay:     formatISKValue(c.SkillPointsValue),
			tags:                   tags,
			tagsDisplay:            strings.Join(slices.Sorted(tags.All()), ", "),
			totalNetWorth:          total,
			totalNetWorthDisplay:   formatISKValue(total),
			walletBalance:          c.WalletBalance,
			walletDisplay:          formatISKValue(c.WalletBalance),
		})
	}
	return rows, nil
}

func formatISKValue(v optional.Optional[float64]) string {
	return v.StringFunc("?", func(v float64) string {
		return humanize.FormatFloat("#,###.", v)
	})
}

// showHelpPopUp shows a popUp with text as content
// and it's position aligned to widget obj.
func showHelpPopUp(text string, isMobile bool, obj fyne.CanvasObject) {
	var pu *widget.PopUp
	closePopUp := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		pu.Hide()
	})
	title := widget.NewLabel("Help")
	title.TextStyle.Bold = true
	body := widget.NewLabel(text)
	body.Wrapping = fyne.TextWrapWord

	p := theme.Padding()
	canvas := fyne.CurrentApp().Driver().CanvasForObject(obj)
	var spacerSize fyne.Size
	if isMobile {
		_, s := canvas.InteractiveArea()
		spacerSize = fyne.NewSize(s.Width-2*p, s.Height/2)
	} else {
		spacerSize = fyne.NewSize(300, 400)
	}
	spacer := xwidget.NewSpacer(spacerSize)
	c := container.NewStack(spacer, container.NewBorder(
		container.NewHBox(title, layout.NewSpacer(), closePopUp),
		nil,
		nil,
		nil,
		container.NewVScroll(container.NewPadded(body)),
	))
	pu = widget.NewPopUp(c, canvas)

	if isMobile {
		pos, s := canvas.InteractiveArea()
		x := pos.X
		y := pos.Y + s.Height/2
		pu.ShowAtPosition(fyne.NewPos(x, y))
	} else {
		x := obj.MinSize().Width - pu.MinSize().Width
		y := obj.MinSize().Height - pu.MinSize().Height + 2*p
		pu.ShowAtRelativePosition(fyne.NewPos(x, y), obj)
	}
}

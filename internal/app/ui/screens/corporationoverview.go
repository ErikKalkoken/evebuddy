package screens

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xlayout"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type corporationOverviewRow struct {
	activeContracts      optional.Optional[int64]
	activeIndustryJobs   optional.Optional[int64]
	alliance             optional.Optional[*app.EveEntity]
	corporationID        int64
	faction              optional.Optional[*app.EveEntity]
	memberCount          optional.Optional[int]
	name                 string
	reinforcedStructures optional.Optional[int64]
	searchTarget         string
	walletBalance        optional.Optional[float64]
}

func (r corporationOverviewRow) allianceName() string {
	return optional.Map(r.alliance, "", func(v *app.EveEntity) string {
		return v.Name
	})
}

func (r corporationOverviewRow) factionName() string {
	return optional.Map(r.faction, "", func(v *app.EveEntity) string {
		return v.Name
	})
}

// CorporationOverview shows a card per tracked corporation with key operational stats.
type CorporationOverview struct {
	widget.BaseWidget

	OnUpdate func(corporations int)

	filterRun      latestRun
	footer         *widget.Label
	columnSorter   *xwidget.ColumnSorter[corporationOverviewRow]
	loadInfo       *widget.Label
	main           fyne.CanvasObject
	rows           []corporationOverviewRow
	rowsFiltered   []corporationOverviewRow
	searchEntry    *xwidget.SearchEntry
	selectAlliance *kxwidget.FilterChipSelect
	selectFaction  *kxwidget.FilterChipSelect
	sortChip       *kxwidget.SortChip
	u              baseUI
}

func NewCorporationOverview(u baseUI) *CorporationOverview {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[corporationOverviewRow]{{
		Label: "Name",
		Sort: func(a, b corporationOverviewRow) int {
			return xstrings.CompareIgnoreCase(a.name, b.name)
		},
	}, {
		Label: "Alliance",
		Sort: func(a, b corporationOverviewRow) int {
			return xstrings.CompareIgnoreCase(a.allianceName(), b.allianceName())
		},
	}, {
		Label: "Faction",
		Sort: func(a, b corporationOverviewRow) int {
			return xstrings.CompareIgnoreCase(a.factionName(), b.factionName())
		},
	}, {
		Label: "Members",
		Sort: func(a, b corporationOverviewRow) int {
			return optional.Compare(a.memberCount, b.memberCount)
		},
	}, {
		Label: "Wallet",
		Sort: func(a, b corporationOverviewRow) int {
			return optional.Compare(a.walletBalance, b.walletBalance)
		},
	}, {
		Label: "Industry Jobs",
		Sort: func(a, b corporationOverviewRow) int {
			return optional.Compare(a.activeIndustryJobs, b.activeIndustryJobs)
		},
	}, {
		Label: "Contracts",
		Sort: func(a, b corporationOverviewRow) int {
			return optional.Compare(a.activeContracts, b.activeContracts)
		},
	}, {
		Label: "Reinforced",
		Sort: func(a, b corporationOverviewRow) int {
			return optional.Compare(a.reinforcedStructures, b.reinforcedStructures)
		},
	}})

	info := widget.NewLabel("Loading...")
	info.Importance = widget.LowImportance

	a := &CorporationOverview{
		footer:       ui.NewLabelWithTruncation(""),
		columnSorter: xwidget.NewColumnSorter(columns, "Name", xwidget.SortAsc),
		loadInfo:     info,
		u:            u,
	}
	a.ExtendBaseWidget(a)

	a.searchEntry = xwidget.NewSearchEntry("Search corporations", func(_ string) {
		a.filterRowsAsync("")
	})

	if !a.u.IsMobile() {
		a.main = a.makeGrid()
	} else {
		a.main = a.makeList()
	}

	a.selectAlliance = kxwidget.NewFilterChipSelect("Alliance", []string{}, func(string) {
		a.filterRowsAsync("")
	})
	a.selectFaction = kxwidget.NewFilterChipSelect("Faction", []string{}, func(string) {
		a.filterRowsAsync("")
	})
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	})

	// Signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	a.u.Signals().CorporationsChanged.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		switch arg.Section {
		case
			app.SectionCorporationMembers,
			app.SectionCorporationWalletBalances,
			app.SectionCorporationIndustryJobs,
			app.SectionCorporationContracts,
			app.SectionCorporationStructures:
			a.updateItem(ctx, arg.CorporationID)
		}
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		logErr := func(err error) {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to process CharacterSectionChanged", "arg", arg, "error", err)
		}
		if arg.Section == app.SectionCharacterRoles {
			character, err := u.Character().GetCharacter(ctx, arg.CharacterID)
			if err != nil {
				logErr(err)
				return
			}
			corporationID := character.EveCharacter.Corporation.ID
			ok, err := u.Corporation().HasCorporation(ctx, corporationID)
			if err != nil {
				logErr(err)
				return
			}
			if !ok {
				return
			}
			a.updateItem(ctx, corporationID)
		}
	})
	return a
}

func (a *CorporationOverview) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(
		a.selectAlliance,
		a.selectFaction,
		a.sortChip,
	)
	var topBox *fyne.Container
	if a.u.IsMobile() {
		topBox = container.NewVBox(a.searchEntry, container.NewHScroll(filter))
	} else {
		topBox = container.NewBorder(nil, nil, filter, nil, a.searchEntry)
	}
	c := container.NewBorder(
		topBox,
		a.footer,
		nil,
		nil,
		container.NewStack(a.loadInfo, a.main),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *CorporationOverview) makeGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newCorporationCard(
				a.u.EVEImage().CorporationLogoAsync,
				a.u.EVEImage().AllianceLogoAsync,
				a.u.EVEImage().FactionLogoAsync,
				false,
				a.u.InfoViewer().Show,
			)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			co.(*corporationCard).set(r)
		},
	)
	g.OnSelected = func(id widget.GridWrapItemID) {
		defer g.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		go a.u.ShowCorporation(context.Background(), r.corporationID)
	}
	return g
}

func (a *CorporationOverview) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newCorporationCard(
				a.u.EVEImage().CorporationLogoAsync,
				a.u.EVEImage().AllianceLogoAsync,
				a.u.EVEImage().FactionLogoAsync,
				true,
				a.u.InfoViewer().Show,
			)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			co.(*corporationCard).set(r)
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.GridWrapItemID) {
		defer l.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		go a.u.ShowCorporation(context.Background(), r.corporationID)
	}

	return l
}

func (a *CorporationOverview) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.start()
	rows := slices.Clone(a.rows)
	total := len(rows)
	alliance := a.selectAlliance.Selected
	faction := a.selectFaction.Selected
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
		if alliance != "" {
			rows = slices.DeleteFunc(rows, func(r corporationOverviewRow) bool {
				return r.allianceName() != alliance
			})
		}
		if faction != "" {
			rows = slices.DeleteFunc(rows, func(r corporationOverviewRow) bool {
				return r.factionName() != faction
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r corporationOverviewRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		allianceOptions := xslices.Map(rows, func(r corporationOverviewRow) string {
			return r.allianceName()
		})
		factionOptions := xslices.Map(rows, func(r corporationOverviewRow) string {
			return r.factionName()
		})

		footer := fmt.Sprintf("Showing %d / %d corporations", len(rows), total)

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectAlliance.SetOptions(allianceOptions)
			a.selectFaction.SetOptions(factionOptions)
			a.rowsFiltered = rows
			a.main.Refresh()
		})
	})
}

func (a *CorporationOverview) update(ctx context.Context) {
	reset := func() {
		fyne.Do(func() {
			xslices.Clear(&a.rows)
			a.filterRowsAsync("")
		})
	}
	setFooter := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.footer.Text = s
			a.footer.Importance = i
			a.footer.Refresh()
		})
	}
	rows, err := a.fetchRows(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		reset()
		setFooter("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		slog.Error("Failed to refresh corporation overview UI", "err", err)
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.loadInfo.Hide()
		a.filterRowsAsync("")
		if a.OnUpdate != nil {
			a.OnUpdate(len(rows))
		}
	})
}

func (a *CorporationOverview) updateItem(ctx context.Context, corporationID int64) {
	c, err := a.u.Corporation().GetCorporation(ctx, corporationID)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("corporationOverview: Failed to update item", "corporationID", corporationID, "error", err)
		return
	}
	r, err := a.fetchRow(ctx, c)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("corporationOverview: Failed to update item", "corporationID", corporationID, "error", err)
		return
	}
	fyne.Do(func() {
		id := slices.IndexFunc(a.rows, func(x corporationOverviewRow) bool {
			return x.corporationID == corporationID
		})
		if id == -1 {
			return
		}
		a.rows[id] = r
		a.filterRowsAsync("")
	})
}

func (a *CorporationOverview) fetchRows(ctx context.Context) ([]corporationOverviewRow, error) {
	corporations, err := a.u.Corporation().ListCorporations(ctx)
	if err != nil {
		return nil, err
	}
	var rows []corporationOverviewRow
	for _, c := range corporations {
		r, err := a.fetchRow(ctx, c)
		if errors.Is(err, app.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *CorporationOverview) fetchRow(ctx context.Context, corp *app.Corporation) (corporationOverviewRow, error) {
	if corp == nil {
		return corporationOverviewRow{}, fmt.Errorf("no corporation: %w", app.ErrInvalid)
	}
	r := corporationOverviewRow{
		alliance:      corp.EveCorporation.Alliance,
		corporationID: corp.ID,
		faction:       corp.EveCorporation.Faction,
		memberCount:   optional.New(int(corp.EveCorporation.MemberCount)),
		name:          corp.EveCorporation.Name,
		searchTarget:  strings.ToLower(corp.EveCorporation.Name),
	}
	permittedSections, err := a.u.Corporation().PermittedSections(ctx, corp.ID)
	if err != nil {
		return r, err
	}

	if permittedSections.Contains(app.SectionCorporationWalletBalances) {
		balance, err := a.u.Corporation().GetWalletBalancesTotal(ctx, corp.ID)
		if err != nil {
			return r, err
		}
		r.walletBalance = balance
	}

	if permittedSections.Contains(app.SectionCorporationIndustryJobs) {
		v, err := a.u.Corporation().ListActiveCorporationIndustryJobs(ctx, corp.ID)
		if err != nil {
			return r, err
		}
		r.activeIndustryJobs = optional.New(v)
	}

	if permittedSections.Contains(app.SectionCorporationContracts) {
		v, err := a.u.Corporation().ListActiveCorporationContracts(ctx, corp.ID)
		if err != nil {
			return r, err
		}
		r.activeContracts = optional.New(v)
	}

	if permittedSections.Contains(app.SectionCorporationStructures) {
		v, err := a.u.Corporation().ListReinforcedCorporationStructures(ctx, corp.ID)
		if err != nil {
			return r, err
		}
		r.reinforcedStructures = optional.New(v)
	}

	return r, nil
}

// corporationCard shows a corporation card, in a large (desktop) and small (mobile) variant.
type corporationCard struct {
	widget.BaseWidget

	activeContracts    *widget.Label
	activeIndustryJobs *widget.Label
	allianceLogo       *xwidget.TappableImage
	background         *canvas.Rectangle
	border             *canvas.Rectangle
	logoBackground     *canvas.Rectangle
	logoBorder         *canvas.Rectangle
	factionLogo        *xwidget.TappableImage
	isSmall            bool
	loadAlliance       loadFuncAsync
	loadCorporation    loadFuncAsync
	loadFaction        loadFuncAsync
	logo               *canvas.Image
	memberCount        *widget.Label
	name               *widget.Label
	reinforcedCount    *widget.Label
	reinforcedIcon     *ttwidget.Icon
	resourceNormal     fyne.Resource
	resourceReinforced fyne.Resource
	showInfo           func(*app.EveEntity)
	wallet             *widget.Label
}

func newCorporationCard(loadCorporation, loadAlliance, loadFaction loadFuncAsync, isSmall bool, showInfo func(*app.EveEntity)) *corporationCard {
	const numberTemplate = "9.999.999.999"
	makeLabel := func(s string) *widget.Label {
		l := widget.NewLabel(s)
		l.Alignment = fyne.TextAlignTrailing
		l.Truncation = fyne.TextTruncateEllipsis
		return l
	}
	var logo *canvas.Image
	if isSmall {
		logo = xwidget.NewImageFromResource(
			icons.Characterplaceholder256Jpeg,
			fyne.NewSquareSize(88),
		)
	} else {
		logo = xwidget.NewImageFromResource(
			icons.Characterplaceholder512Jpeg,
			fyne.NewSquareSize(192),
		)
	}

	resNormal := theme.NewThemedResource(icons.SpaceStationSvg)
	resReinforced := theme.NewErrorThemedResource(icons.SpaceStationSvg)

	w := &corporationCard{
		activeContracts:    makeLabel(numberTemplate),
		activeIndustryJobs: makeLabel(numberTemplate),
		allianceLogo:       xwidget.NewTappableImage(icons.Corporationplaceholder64Png, nil),
		background:         canvas.NewRectangle(theme.Color(theme.ColorNameHover)),
		border:             canvas.NewRectangle(color.Transparent),
		factionLogo:        xwidget.NewTappableImage(icons.Corporationplaceholder64Png, nil),
		isSmall:            isSmall,
		loadAlliance:       loadAlliance,
		loadCorporation:    loadCorporation,
		loadFaction:        loadFaction,
		logo:               logo,
		logoBackground:     canvas.NewRectangle(colorDarkBackground),
		logoBorder:         canvas.NewRectangle(color.Transparent),
		memberCount:        makeLabel(numberTemplate),
		name:               widget.NewLabel("Wayne Enterprises"),
		reinforcedCount:    makeLabel(numberTemplate),
		reinforcedIcon:     ttwidget.NewIcon(resNormal),
		resourceNormal:     resNormal,
		resourceReinforced: resReinforced,
		showInfo:           showInfo,
		wallet:             makeLabel(numberTemplate + " ISK"),
	}
	w.ExtendBaseWidget(w)
	var badgeSize float32
	if isSmall {
		badgeSize = ui.IconUnitSize
	} else {
		badgeSize = 40
	}

	w.allianceLogo.SetFillMode(canvas.ImageFillContain)
	w.allianceLogo.SetMinSize(fyne.NewSquareSize(badgeSize))
	w.allianceLogo.CornerRadius = theme.InputRadiusSize()

	w.factionLogo.SetFillMode(canvas.ImageFillContain)
	w.factionLogo.SetMinSize(fyne.NewSquareSize(badgeSize))
	w.factionLogo.CornerRadius = theme.InputRadiusSize()

	w.background.CornerRadius = theme.InputRadiusSize()

	w.name.SizeName = theme.SizeNameSubHeadingText
	w.name.Truncation = fyne.TextTruncateEllipsis

	w.border.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	w.border.StrokeWidth = 1
	w.border.CornerRadius = theme.Size(theme.SizeNameInputRadius)

	w.logoBackground.CornerRadius = theme.Size(theme.SizeNameInputRadius)

	w.logoBorder.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	w.logoBorder.StrokeWidth = 1
	w.logoBorder.CornerRadius = theme.Size(theme.SizeNameInputRadius)

	return w
}

func (w *corporationCard) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	if w.isSmall {
		c := container.NewBorder(
			nil,
			nil,
			container.New(layout.NewCustomPaddedLayout(0, 0, 0, 2*p),
				container.NewStack(
					w.background,
					container.New(layout.NewCustomPaddedLayout(2*p, 2*p, 3*p, 3*p),
						container.NewVBox(
							container.NewStack(w.logoBackground, w.logo, w.logoBorder),
							container.New(layout.NewCustomPaddedLayout(0, -p, 0, 0),
								container.NewHBox(w.allianceLogo, layout.NewSpacer(), w.factionLogo),
							),
						),
					),
				),
			),
			nil,
			container.New(layout.NewCustomPaddedVBoxLayout(-3*p),
				container.New(layout.NewCustomPaddedLayout(0, p, -2*p, 0), w.name),
				container.NewBorder(
					nil, nil, widget.NewIcon(theme.NewThemedResource(icons.AccountMultipleSvg)), nil, w.memberCount,
				),
				container.NewBorder(
					nil, nil, widget.NewIcon(theme.NewThemedResource(icons.CashSvg)), nil, w.wallet,
				),
				container.NewBorder(
					nil, nil, widget.NewIcon(theme.NewThemedResource(icons.FactorySvg)), nil, w.activeIndustryJobs,
				),
				container.NewBorder(
					nil, nil, widget.NewIcon(theme.NewThemedResource(icons.FileSignSvg)), nil, w.activeContracts,
				),
			),
		)
		r := container.New(layout.NewCustomPaddedLayout(p, p, p, p), container.NewStack(c, w.border))
		return widget.NewSimpleRenderer(r)
	}

	members := ttwidget.NewIcon(theme.NewThemedResource(icons.AccountMultipleSvg))
	members.SetToolTip("Member count")
	wallet := ttwidget.NewIcon(theme.NewThemedResource(icons.CashSvg))
	wallet.SetToolTip("Wallet balance")
	industry := ttwidget.NewIcon(theme.NewThemedResource(icons.FactorySvg))
	industry.SetToolTip("Active industry jobs")
	contracts := ttwidget.NewIcon(theme.NewThemedResource(icons.FileSignSvg))
	contracts.SetToolTip("Active contracts")
	w.reinforcedIcon.SetToolTip("Reinforced structures")

	logoBorder := &layout.CustomPaddedLayout{
		TopPadding:    1 * p,
		BottomPadding: 1 * p,
		LeftPadding:   1 * p,
		RightPadding:  1 * p,
	}

	c := container.NewBorder(
		container.New(layout.NewCustomPaddedLayout(0, 0, -p, -p), w.name),
		container.New(
			layout.NewCustomPaddedVBoxLayout(-2*p),
			container.NewBorder(nil, nil, members, nil, w.memberCount),
			container.NewBorder(nil, nil, wallet, nil, w.wallet),
			container.NewBorder(nil, nil, industry, nil, w.activeIndustryJobs),
			container.NewBorder(nil, nil, contracts, nil, w.activeContracts),
			container.NewBorder(nil, nil, w.reinforcedIcon, nil, w.reinforcedCount),
		),
		nil,
		nil,
		container.NewStack(
			container.NewStack(w.logoBackground, w.logo, w.logoBorder),
			container.New(&xlayout.BottomLeftLayout{}, container.New(logoBorder, w.allianceLogo)),
			container.New(&xlayout.BottomRightLayout{}, container.New(logoBorder, w.factionLogo)),
		),
	)
	r := container.NewStack(
		container.New(layout.NewCustomPaddedLayout(p, p, 2*p, 2*p), c),
		w.border,
	)
	return widget.NewSimpleRenderer(r)
}

func (w *corporationCard) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.background.FillColor = th.Color(theme.ColorNameHover, v)
	w.border.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	w.BaseWidget.Refresh()
}

func (w *corporationCard) set(c corporationOverviewRow) {
	var logoSize int
	if w.isSmall {
		logoSize = 256
	} else {
		logoSize = 512
	}
	w.loadCorporation(c.corporationID, logoSize, func(r fyne.Resource) {
		w.logo.Resource = r
		w.logo.Refresh()
	})

	const badgeSize = 64

	if alliance, ok := c.alliance.Value(); ok {
		w.allianceLogo.OnTapped = func() {
			w.showInfo(alliance)
		}
		w.allianceLogo.SetToolTip(alliance.Name)
		w.allianceLogo.Show()
		w.loadAlliance(alliance.ID, badgeSize, func(r fyne.Resource) {
			w.allianceLogo.SetResource(r)
		})
	} else {
		w.allianceLogo.Hide()
	}

	if faction, ok := c.faction.Value(); ok {
		w.factionLogo.OnTapped = func() {
			w.showInfo(faction)
		}
		w.factionLogo.SetToolTip(faction.Name)
		w.factionLogo.Show()
		w.loadFaction(faction.ID, badgeSize, func(r fyne.Resource) {
			w.factionLogo.SetResource(r)
		})
	} else {
		w.factionLogo.Hide()
	}

	w.name.SetText(c.name)

	w.memberCount.SetText(c.memberCount.StringFunc("?", func(v int) string {
		return humanize.Comma(int64(v))
	}))

	w.wallet.SetText(c.walletBalance.StringFunc("?", func(v float64) string {
		return humanize.Comma(int64(v)) + " ISK"
	}))

	w.activeIndustryJobs.SetText(c.activeIndustryJobs.StringFunc("?", func(v int64) string {
		if v == 0 {
			return "-"
		}
		return humanize.Comma(v)
	}))

	w.activeContracts.SetText(c.activeContracts.StringFunc("?", func(v int64) string {
		if v == 0 {
			return "-"
		}
		return humanize.Comma(v)
	}))

	if v, ok := c.reinforcedStructures.Value(); ok && v > 0 {
		w.reinforcedCount.SetText(humanize.Comma(int64(v)))
		w.reinforcedIcon.SetResource(w.resourceReinforced)
	} else {
		w.reinforcedCount.SetText("-")
		w.reinforcedIcon.SetResource(w.resourceNormal)
	}
}

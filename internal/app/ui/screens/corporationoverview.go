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
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// Names of the corporation overview filters, used as labels on desktop and as option names on mobile.
const (
	corporationOverviewFilterAlliance = "Alliance"
	corporationOverviewFilterFaction  = "Faction"
)

// corporationOverviewFilter is the selected value of each corporation overview filter. Empty means not filtered.
type corporationOverviewFilter struct {
	alliance string
	faction  string
}

// match reports whether row r passes all filters.
func (f corporationOverviewFilter) match(r corporationOverviewRow) bool {
	switch {
	case f.alliance != "" && r.allianceName() != f.alliance,
		f.faction != "" && r.factionName() != f.faction:
		return false
	}
	return true
}

type corporationOverviewRow struct {
	activeIndustryJobs   optional.Optional[int64]
	alliance             optional.Optional[*app.EveEntity]
	corporationID        int64
	faction              optional.Optional[*app.EveEntity]
	memberCount          optional.Optional[int]
	name                 string
	outstandingContracts optional.Optional[int64]
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

	filterChip     *xwidget.FilterChipCompact // only on mobile
	filterRun      xsync.LatestRun
	updateRun      xsync.LatestRun
	footerLabel    *widget.Label
	errorLabel     *widget.Label
	columnSorter   *xwidget.ColumnSorter[corporationOverviewRow]
	loadInfoLabel  *widget.Label
	main           fyne.CanvasObject
	rows           []corporationOverviewRow
	rowsFiltered   []corporationOverviewRow
	searchEntry    *xwidget.SearchEntry
	selectAlliance *kxwidget.FilterChipSelect // select chips only on desktop
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
			return optional.Compare(a.outstandingContracts, b.outstandingContracts)
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
		footerLabel:   ui.NewLabelWithTruncation(""),
		errorLabel:    ui.NewLabelWithWrapping(""),
		columnSorter:  xwidget.NewColumnSorter(columns, "Name", xwidget.SortAsc),
		loadInfoLabel: info,
		u:             u,
	}
	a.ExtendBaseWidget(a)

	a.errorLabel.Importance = widget.DangerImportance
	a.errorLabel.Hide()

	placeholder := "Search corporations"
	if a.u.IsMobile() {
		placeholder = "Search" // shares the row with the chips
	}
	a.searchEntry = xwidget.NewSearchEntry(placeholder, func(_ string) {
		a.filterRowsAsync("")
	})

	if !a.u.IsMobile() {
		a.main = a.makeGrid()
	} else {
		a.main = a.makeList()
	}

	if a.u.IsMobile() {
		a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
			a.filterRowsAsync("")
		})
	} else {
		makeSelect := func(label string) *kxwidget.FilterChipSelect {
			return kxwidget.NewFilterChipSelect(label, []string{}, func(string) {
				a.filterRowsAsync("")
			})
		}
		a.selectAlliance = makeSelect(corporationOverviewFilterAlliance)
		a.selectFaction = makeSelect(corporationOverviewFilterFaction)
	}
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
	// characters can change which corporations are listed
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		if arg.Section != app.SectionEveCharacters {
			return
		}
		characterIDs, err := a.u.Character().ListCharacterIDs(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("corporationOverview: Failed to list characters", "error", err)
			return
		}
		if characterIDs.ContainsAny(arg.Changed.All()) {
			a.update(ctx) // a character may have changed corporation
		}
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
		if arg.Section == app.SectionCharacterRoles {
			a.update(ctx) // roles can change which corporations are listed
		}
	})
	return a
}

func (a *CorporationOverview) CreateRenderer() fyne.WidgetRenderer {
	var topBox *fyne.Container
	if a.u.IsMobile() {
		topBox = container.NewBorder(nil, nil, nil, container.NewHBox(a.filterChip, a.sortChip), a.searchEntry)
	} else {
		filter := container.NewHBox(
			a.selectAlliance,
			a.selectFaction,
			a.sortChip,
		)
		topBox = container.NewBorder(nil, nil, filter, nil, a.searchEntry)
	}
	c := container.NewBorder(
		topBox,
		container.NewVBox(a.errorLabel, a.footerLabel),
		nil,
		nil,
		container.NewStack(a.loadInfoLabel, a.main),
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

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chips on desktop.
func (a *CorporationOverview) currentFilter() corporationOverviewFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		return corporationOverviewFilter{
			alliance: s[corporationOverviewFilterAlliance],
			faction:  s[corporationOverviewFilterFaction],
		}
	}
	return corporationOverviewFilter{
		alliance: a.selectAlliance.Selected,
		faction:  a.selectFaction.Selected,
	}
}

func (a *CorporationOverview) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.Start()
	rows := slices.Clone(a.rows)
	total := len(rows)
	filter := a.currentFilter()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
		rows = slices.DeleteFunc(rows, func(r corporationOverviewRow) bool {
			return !filter.match(r)
		})
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
			a.footerLabel.Text = footer
			a.footerLabel.Importance = widget.MediumImportance
			a.footerLabel.Refresh()
			if a.filterChip != nil {
				a.filterChip.SetOptions(
					xwidget.NewFilterOptionMultiChoice(corporationOverviewFilterAlliance, allianceOptions),
					xwidget.NewFilterOptionMultiChoice(corporationOverviewFilterFaction, factionOptions),
				)
			} else {
				a.selectAlliance.SetOptions(allianceOptions)
				a.selectFaction.SetOptions(factionOptions)
			}
			a.rowsFiltered = rows
			a.main.Refresh()
		})
	})
}

func (a *CorporationOverview) update(ctx context.Context) {
	isLatest := a.updateRun.Start()
	reset := func() {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.errorLabel.Hide()
			xslices.Clear(&a.rows)
			a.filterRowsAsync("")
		})
	}
	setError := func(err error) {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.errorLabel.SetText("ERROR: " + a.u.ErrorDisplay(err))
			a.errorLabel.Show()
			a.Refresh()
		})
	}
	rows, err := a.fetchRows(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		reset()
		setError(err)
		slog.Error("Failed to refresh corporation overview UI", "err", err)
		return
	}
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.errorLabel.Hide()
		a.rows = rows
		a.loadInfoLabel.Hide()
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
	selectable, err := a.u.Corporation().ListCorporationsForSelection(ctx, a.u.Settings().HideLimitedCorporations())
	if err != nil {
		return nil, err
	}
	selectableIDs := set.Collect(xiter.MapSlice(selectable, func(x *app.EntityShort) int64 {
		return x.ID
	}))
	var rows []corporationOverviewRow
	for _, c := range corporations {
		if !selectableIDs.Contains(c.ID) {
			continue
		}
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
		v, err := a.u.Corporation().CountOutstandingCorporationContracts(ctx, corp.ID)
		if err != nil {
			return r, err
		}
		r.outstandingContracts = optional.New(v)
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

	activeIndustryJobsLabel   *widget.Label
	border                    *canvas.Rectangle
	isSmall                   bool
	logo                      *corporationLogoWidget
	memberCountLabel          *widget.Label
	nameLabel                 *widget.Label
	outstandingContractsLabel *widget.Label
	reinforcedCountLabel      *widget.Label
	reinforcedIcon            *ttwidget.Icon
	resourceNormal            fyne.Resource
	resourceReinforced        fyne.Resource
	walletLabel               *widget.Label
}

func newCorporationCard(loadCorporation, loadAlliance, loadFaction loadFuncAsync, isSmall bool, showInfo func(*app.EveEntity)) *corporationCard {
	const numberTemplate = "9.999.999.999"
	makeLabel := func(s string) *widget.Label {
		l := widget.NewLabel(s)
		l.Alignment = fyne.TextAlignTrailing
		l.Truncation = fyne.TextTruncateEllipsis
		return l
	}
	resNormal := theme.NewThemedResource(icons.SpaceStationSvg)
	resReinforced := theme.NewErrorThemedResource(icons.SpaceStationSvg)

	w := &corporationCard{
		activeIndustryJobsLabel:   makeLabel(numberTemplate),
		border:                    canvas.NewRectangle(color.Transparent),
		isSmall:                   isSmall,
		logo:                      newCorporationLogoWithLogos(loadCorporation, loadAlliance, loadFaction, isSmall, showInfo),
		memberCountLabel:          makeLabel(numberTemplate),
		nameLabel:                 widget.NewLabel("Wayne Enterprises"),
		outstandingContractsLabel: makeLabel(numberTemplate),
		reinforcedCountLabel:      makeLabel(numberTemplate),
		reinforcedIcon:            ttwidget.NewIcon(resNormal),
		resourceNormal:            resNormal,
		resourceReinforced:        resReinforced,
		walletLabel:               makeLabel(numberTemplate + " ISK"),
	}
	w.ExtendBaseWidget(w)

	w.nameLabel.SizeName = theme.SizeNameSubHeadingText
	w.nameLabel.Truncation = fyne.TextTruncateEllipsis

	w.border.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	w.border.StrokeWidth = 1
	w.border.CornerRadius = theme.Size(theme.SizeNameInputRadius)

	return w
}

func (w *corporationCard) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	if w.isSmall {
		c := container.NewBorder(
			nil,
			nil,
			container.New(layout.NewCustomPaddedLayout(0, 0, 0, 2*p), w.logo),
			nil,
			container.New(layout.NewCustomPaddedVBoxLayout(-3*p),
				container.New(layout.NewCustomPaddedLayout(0, p, -2*p, 0), w.nameLabel),
				container.NewBorder(
					nil, nil, widget.NewIcon(theme.NewThemedResource(icons.AccountMultipleSvg)), nil, w.memberCountLabel,
				),
				container.NewBorder(
					nil, nil, widget.NewIcon(theme.NewThemedResource(icons.CashSvg)), nil, w.walletLabel,
				),
				container.NewBorder(
					nil, nil, widget.NewIcon(theme.NewThemedResource(icons.FactorySvg)), nil, w.activeIndustryJobsLabel,
				),
				container.NewBorder(
					nil, nil, widget.NewIcon(theme.NewThemedResource(icons.FileSignSvg)), nil, w.outstandingContractsLabel,
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
	contracts.SetToolTip("Outstanding contracts")
	w.reinforcedIcon.SetToolTip("Reinforced structures")

	c := container.NewBorder(
		container.New(layout.NewCustomPaddedLayout(0, 0, -p, -p), w.nameLabel),
		container.New(
			layout.NewCustomPaddedVBoxLayout(-2*p),
			container.NewBorder(nil, nil, members, nil, w.memberCountLabel),
			container.NewBorder(nil, nil, wallet, nil, w.walletLabel),
			container.NewBorder(nil, nil, industry, nil, w.activeIndustryJobsLabel),
			container.NewBorder(nil, nil, contracts, nil, w.outstandingContractsLabel),
			container.NewBorder(nil, nil, w.reinforcedIcon, nil, w.reinforcedCountLabel),
		),
		nil,
		nil,
		w.logo,
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
	w.border.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	w.BaseWidget.Refresh()
}

func (w *corporationCard) set(c corporationOverviewRow) {
	w.logo.set(c.corporationID, c.alliance, c.faction)

	w.nameLabel.SetText(c.name)

	w.memberCountLabel.SetText(c.memberCount.StringFunc("?", func(v int) string {
		return humanize.Comma(int64(v))
	}))

	w.walletLabel.SetText(c.walletBalance.StringFunc("?", func(v float64) string {
		return humanize.Comma(int64(v)) + " ISK"
	}))

	w.activeIndustryJobsLabel.SetText(c.activeIndustryJobs.StringFunc("?", func(v int64) string {
		if v == 0 {
			return "-"
		}
		return humanize.Comma(v)
	}))

	w.outstandingContractsLabel.SetText(c.outstandingContracts.StringFunc("?", func(v int64) string {
		if v == 0 {
			return "-"
		}
		return humanize.Comma(v)
	}))

	if v, ok := c.reinforcedStructures.Value(); ok && v > 0 {
		w.reinforcedCountLabel.SetText(humanize.Comma(int64(v)))
		w.reinforcedIcon.SetResource(w.resourceReinforced)
	} else {
		w.reinforcedCountLabel.SetText("-")
		w.reinforcedIcon.SetResource(w.resourceNormal)
	}
}

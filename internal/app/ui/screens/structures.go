package screens

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	structuresPowerLow  = "Low Power"
	structuresPowerHigh = "High Power"
)

// Names of the structure filters, used as labels on desktop and as option names on mobile.
const (
	structuresFilterOwner   = "Owner"
	structuresFilterPower   = "Power"
	structuresFilterRegion  = "Region"
	structuresFilterService = "Service"
	structuresFilterState   = "State"
	structuresFilterSystem  = "System"
	structuresFilterType    = "Type"
)

// structuresFilter is the selected value of each structure filter. Empty means not filtered.
type structuresFilter struct {
	owner       string
	power       string
	region      string
	service     string
	solarSystem string
	state       string
	typeName    string
}

// match reports whether row r passes all filters.
func (f structuresFilter) match(r structureRow) bool {
	switch {
	case f.owner != "" && r.corporationName != f.owner,
		f.power == structuresPowerHigh && !r.isFullPower,
		f.power == structuresPowerLow && r.isFullPower,
		f.region != "" && r.regionName != f.region,
		f.service != "" && !r.services.Contains(f.service),
		f.solarSystem != "" && r.solarSystemName != f.solarSystem,
		f.state != "" && r.stateDisplay != f.state,
		f.typeName != "" && r.typeName != f.typeName:
		return false
	}
	return true
}

type structureRow struct {
	corporationID      int64
	corporationName    string
	fuelExpires        optional.Optional[time.Time]
	fuelSort           time.Time
	isFullPower        bool
	isReinforced       bool
	regionID           int64
	regionName         string
	searchTarget       string
	services           set.Set[string]
	servicesText       string
	solarSystemDisplay []widget.RichTextSegment
	solarSystemID      int64
	solarSystemName    string
	stateColor         fyne.ThemeColorName
	stateDisplay       string
	stateText          string
	structureID        int64
	structureName      string
	typeID             int64
	typeName           string
}

func (r structureRow) fuelExpiresDisplay() []widget.RichTextSegment {
	var text string
	var color fyne.ThemeColorName
	if v, ok := r.fuelExpires.Value(); ok {
		color = theme.ColorNameForeground
		text = ihumanize.Duration(time.Until(v))
	} else {
		color = theme.ColorNameWarning
		text = "Low Power"
	}
	return xwidget.RichTextSegmentsFromText(text, widget.RichTextStyle{
		ColorName: color,
	})
}

// setSearchTarget sets the text the search entry matches against.
func (r *structureRow) setSearchTarget() {
	r.searchTarget = strings.ToLower(r.structureName + "\n" + r.solarSystemName) // separator prevents matches across names
}

type Structures struct {
	widget.BaseWidget

	OnUpdate func(count int)

	columnSorter      *xwidget.ColumnSorter[structureRow]
	corporation       atomic.Pointer[app.Corporation]
	filterChip        *xwidget.FilterChipCompact // only on mobile
	filterRun         latestRun
	footer            *widget.Label
	forCorporation    bool
	main              fyne.CanvasObject
	rows              []structureRow
	rowsFiltered      []structureRow
	searchEntry       *xwidget.SearchEntry
	selectPower       *kxwidget.FilterChipSelect // select chips only on desktop
	selectRegion      *kxwidget.FilterChipSelect
	selectService     *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectState       *kxwidget.FilterChipSelect
	selectType        *kxwidget.FilterChipSelect
	selectOwner       *kxwidget.FilterChipSelect
	sortChip          *kxwidget.SortChip
	u                 baseUI
}

func NewUnifiedStructures(u baseUI) *Structures {
	return newStructuresForCorporation(u, false)
}

func NewStructuresForCorporation(u baseUI) *Structures {
	return newStructuresForCorporation(u, true)
}

func newStructuresForCorporation(u baseUI, forCorporation bool) *Structures {
	cols := []xwidget.DataColumn[structureRow]{{
		Label: "Name",
		Width: 250,
		Sort: func(a, b structureRow) int {
			return xstrings.CompareIgnoreCase(a.structureName, b.structureName)
		},
		Update: func(r structureRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.structureName)
		},
	}, ui.MakeEveEntityColumn(ui.MakeEveEntityColumnParams[structureRow]{
		EIS:   u.EVEImage(),
		Label: "Type",
		GetEntity: func(r structureRow) *app.EveEntity {
			return &app.EveEntity{
				Category: app.EveEntityInventoryType,
				ID:       r.typeID,
				Name:     r.typeName,
			}
		},
	}), {
		Label: "Fuel Expires",
		Width: 150,
		Sort: func(a, b structureRow) int {
			return a.fuelSort.Compare(b.fuelSort)
		},
		Update: func(r structureRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.fuelExpiresDisplay())
		},
	}, {
		Label: "State",
		Width: 150,
		Update: func(r structureRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.stateText, widget.RichTextStyle{
				ColorName: r.stateColor,
			})
		},
	}, {
		Label: "Services",
		Width: 200,
		Update: func(r structureRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.servicesText)
		},
	}}
	if !forCorporation {
		cols = slices.Insert(cols, 4, ui.MakeEveEntityColumn(ui.MakeEveEntityColumnParams[structureRow]{
			EIS:   u.EVEImage(),
			Label: "Owner",
			GetEntity: func(r structureRow) *app.EveEntity {
				return &app.EveEntity{
					Category: app.EveEntityCorporation,
					ID:       r.corporationID,
					Name:     r.corporationName,
				}
			},
		}))
	}
	columns := xwidget.NewDataColumns(cols)
	a := &Structures{
		columnSorter:   xwidget.NewColumnSorter(columns, "Name", xwidget.SortAsc),
		footer:         ui.NewLabelWithWrapping(""),
		forCorporation: forCorporation,
		u:              u,
	}
	a.ExtendBaseWidget(a)
	if !a.u.IsMobile() {
		a.main = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync, func(_ int, r structureRow) {
				showCorporationStructureWindowAsync(context.Background(), u, r.corporationID, r.structureID, r.solarSystemName)
			},
		)
	} else {
		a.main = xwidget.MakeDataList(
			columns,
			&a.rowsFiltered,
			func(col string, r structureRow) []widget.RichTextSegment {
				switch col {
				case "Type":
					return xwidget.RichTextSegmentsFromText(r.typeName)
				case "Name":
					return xwidget.RichTextSegmentsFromText(r.structureName)
				case "Fuel Expires":
					return r.fuelExpiresDisplay()
				case "Owner":
					return xwidget.RichTextSegmentsFromText(r.corporationName)
				case "State":
					return xwidget.RichTextSegmentsFromText(r.stateText, widget.RichTextStyle{
						ColorName: r.stateColor,
					})
				case "Services":
					return xwidget.RichTextSegmentsFromText(r.servicesText)
				}
				return xwidget.RichTextSegmentsFromText("?")
			},
			func(r structureRow) {
				showCorporationStructureWindowAsync(context.Background(), u, r.corporationID, r.structureID, r.solarSystemName)
			},
		)
	}

	// filter
	placeholder := "Search structures and systems"
	if a.u.IsMobile() {
		placeholder = "Search" // shares the row with the chips
	}
	a.searchEntry = xwidget.NewSearchEntry(placeholder, func(_ string) {
		a.filterRowsAsync("")
	})
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
		a.selectOwner = makeSelect(structuresFilterOwner)
		a.selectRegion = makeSelect(structuresFilterRegion)
		a.selectService = makeSelect(structuresFilterService)
		a.selectSolarSystem = makeSelect(structuresFilterSystem)
		a.selectState = makeSelect(structuresFilterState)
		a.selectType = makeSelect(structuresFilterType)
		a.selectPower = makeSelect(structuresFilterPower)
		a.selectPower.SetOptions([]string{structuresPowerHigh, structuresPowerLow})
	}
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	})

	// Signals
	if forCorporation {
		a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			fyne.Do(func() {
				a.searchEntry.ClearSilent()
				if a.filterChip != nil {
					a.filterChip.ResetSilent()
					return
				}
				clearSelectsSilent(
					a.selectOwner,
					a.selectPower,
					a.selectRegion,
					a.selectService,
					a.selectSolarSystem,
					a.selectState,
					a.selectType,
				)
			})
			a.update(ctx)
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if a.corporation.Load().IDOrZero() != arg.CorporationID {
				return
			}
			if arg.Section != app.SectionCorporationStructures {
				return
			}
			a.update(ctx)
		})
		a.u.Signals().RefreshTickerExpired.AddListener(func(ctx context.Context, _ struct{}) {
			a.update(ctx)
		})
	} else {
		a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
			a.update(ctx)
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if arg.Section == app.SectionCorporationStructures {
				a.update(ctx)
			}
		})
		a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			a.update(ctx)
		})
		a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			a.update(ctx)
		})
	}
	return a
}

func (a *Structures) CreateRenderer() fyne.WidgetRenderer {
	var top fyne.CanvasObject
	if a.u.IsMobile() {
		top = container.NewBorder(nil, nil, nil, container.NewHBox(a.filterChip, a.sortChip), a.searchEntry)
	} else {
		objs := []fyne.CanvasObject{a.selectType, a.selectState, a.selectSolarSystem, a.selectRegion, a.selectService, a.selectPower}
		if !a.forCorporation {
			objs = slices.Insert(objs, 4, fyne.CanvasObject(a.selectOwner))
		}
		top = container.NewBorder(nil, nil, container.NewHBox(objs...), nil, a.searchEntry)
	}
	c := container.NewBorder(top, a.footer, nil, nil, a.main)
	return widget.NewSimpleRenderer(c)
}

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chips on desktop.
func (a *Structures) currentFilter() structuresFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		return structuresFilter{
			owner:       s[structuresFilterOwner],
			power:       s[structuresFilterPower],
			region:      s[structuresFilterRegion],
			service:     s[structuresFilterService],
			solarSystem: s[structuresFilterSystem],
			state:       s[structuresFilterState],
			typeName:    s[structuresFilterType],
		}
	}
	return structuresFilter{
		owner:       a.selectOwner.Selected,
		power:       a.selectPower.Selected,
		region:      a.selectRegion.Selected,
		service:     a.selectService.Selected,
		solarSystem: a.selectSolarSystem.Selected,
		state:       a.selectState.Selected,
		typeName:    a.selectType.Selected,
	}
}

func (a *Structures) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.start()
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	filter := a.currentFilter()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
		rows = slices.DeleteFunc(rows, func(r structureRow) bool {
			return !filter.match(r)
		})
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r structureRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		// set data & refresh
		ownerOptions := xslices.Map(rows, func(r structureRow) string {
			return r.corporationName
		})
		regionOptions := xslices.Map(rows, func(r structureRow) string {
			return r.regionName
		})
		solarSystemOptions := xslices.Map(rows, func(r structureRow) string {
			return r.solarSystemName
		})
		stateOptions := xslices.Map(rows, func(r structureRow) string {
			return r.stateDisplay
		})
		servicesOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r structureRow) set.Set[string] {
			return r.services
		})...).All())
		typeOptions := xslices.Map(rows, func(r structureRow) string {
			return r.typeName
		})

		footer := fmt.Sprintf("Showing %s / %s structures", ihumanize.Comma(len(rows)), ihumanize.Comma(totalRows))

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			if a.filterChip != nil {
				options := []xwidget.FilterOption{
					xwidget.NewFilterOptionMultiChoice(structuresFilterType, typeOptions),
					xwidget.NewFilterOptionMultiChoice(structuresFilterState, stateOptions),
					xwidget.NewFilterOptionMultiChoice(structuresFilterSystem, solarSystemOptions),
					xwidget.NewFilterOptionMultiChoice(structuresFilterRegion, regionOptions),
				}
				if !a.forCorporation {
					options = append(options, xwidget.NewFilterOptionMultiChoice(structuresFilterOwner, ownerOptions))
				}
				options = append(options,
					xwidget.NewFilterOptionMultiChoice(structuresFilterService, servicesOptions),
					xwidget.NewFilterOptionMultiChoice(structuresFilterPower, []string{structuresPowerHigh, structuresPowerLow}),
				)
				a.filterChip.SetOptions(options...)
			} else {
				a.selectOwner.SetOptions(ownerOptions)
				a.selectRegion.SetOptions(regionOptions)
				a.selectSolarSystem.SetOptions(solarSystemOptions)
				a.selectState.SetOptions(stateOptions)
				a.selectService.SetOptions(servicesOptions)
				a.selectType.SetOptions(typeOptions)
			}
			a.rowsFiltered = rows
			a.main.Refresh()
		})
	})
}

func (a *Structures) update(ctx context.Context) {
	reset := func() {
		fyne.Do(func() {
			xslices.Clear(&a.rows)
			a.filterRowsAsync("")
		})
	}
	rows, err := a.fetchData(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh corporation structures UI", "err", err)
		reset()
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + a.u.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
		return
	}
	var reinforceCount int
	for _, r := range rows {
		if r.isReinforced {
			reinforceCount++
		}
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync("")
		if a.OnUpdate != nil {
			a.OnUpdate(reinforceCount)
		}
	})
}

func (a *Structures) fetchData(ctx context.Context) ([]structureRow, error) {
	var structures []*app.CorporationStructure
	if a.forCorporation {
		corporationID := a.corporation.Load().IDOrZero()
		if corporationID == 0 {
			return nil, nil
		}
		x, err := a.u.Corporation().ListStructures(ctx, corporationID)
		if err != nil {
			return nil, err
		}
		structures = x
	} else {
		x, err := a.u.Corporation().ListAllStructures(ctx)
		if err != nil {
			return nil, err
		}
		structures = x
	}
	corporationNames, err := a.u.Corporation().CorporationNames(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]structureRow, len(structures))
	for i, s := range structures {
		stateText := s.State.DisplayShort()
		if v, ok := s.StateTimerEnd.Value(); ok {
			var x string
			d := time.Until(v)
			if d >= 0 {
				x = ihumanize.Duration(d)
			} else {
				x = "EXPIRED"
			}
			stateText += ": " + x
		}
		services := set.Collect(xiter.Map(xiter.FilterSlice(s.Services, func(x *app.StructureService) bool {
			return x.State == app.StructureServiceStateOnline
		}), func(x *app.StructureService) string {
			return x.Name
		}))
		servicesText := xstrings.JoinsOrEmpty(slices.Sorted(services.All()), ", ", "-")
		region := s.System.Constellation.Region

		rows[i] = structureRow{
			corporationID:      s.CorporationID,
			corporationName:    corporationNames[s.CorporationID],
			fuelExpires:        s.FuelExpires,
			fuelSort:           s.FuelExpires.ValueOrZero(),
			isFullPower:        !s.FuelExpires.IsEmpty(),
			isReinforced:       s.State.IsReinforce(),
			regionID:           region.ID,
			regionName:         region.Name,
			services:           services,
			servicesText:       servicesText,
			solarSystemDisplay: s.System.DisplayRichText(),
			solarSystemID:      s.System.ID,
			solarSystemName:    s.System.Name,
			stateColor:         s.State.Color(),
			stateDisplay:       s.State.Display(),
			stateText:          stateText,
			structureID:        s.StructureID,
			structureName:      s.DisplayName(),
			typeID:             s.Type.ID,
			typeName:           s.Type.Name,
		}
		rows[i].setSearchTarget()
	}
	return rows, nil
}

func showCorporationStructureWindowAsync(ctx context.Context, u baseUI, corporationID int64, structureID int64, title string) {
	windowID := fmt.Sprintf("corporationstructure-%d-%d", corporationID, structureID)
	w, created := u.GetOrCreateWindow(
		windowID,
		title,
	)
	if !created {
		w.Show()
		return
	}

	go func() {
		reportError := func(err error) {
			fyne.Do(func() {
				u.DestroyWindow(windowID)
				ui.ShowErrorAndLog("Failed to show structure", err, u.IsDeveloperMode(), u.MainWindow())
			})
		}
		structure, err := u.Corporation().GetStructure(ctx, corporationID, structureID)
		if err != nil {
			reportError(err)
			return
		}
		corporationNames, err := u.Corporation().CorporationNames(ctx)
		if err != nil {
			reportError(err)
			return
		}
		corporationName := corporationNames[corporationID]
		fyne.Do(func() {
			var services []widget.RichTextSegment
			if len(structure.Services) == 0 {
				services = xwidget.RichTextSegmentsFromText("-")
			} else {
				slices.SortFunc(structure.Services, func(a, b *app.StructureService) int {
					return strings.Compare(a.Name, b.Name)
				})
				for _, x := range structure.Services {
					var color fyne.ThemeColorName
					name := x.Name
					if x.State == app.StructureServiceStateOnline {
						color = theme.ColorNameForeground
					} else {
						color = theme.ColorNameDisabled
						name += " [offline]"
					}
					services = slices.Concat(services, xwidget.RichTextSegmentsFromText(name, widget.RichTextStyle{
						ColorName: color,
					}))
				}
			}

			var fuelText, powerText string
			var powerColor fyne.ThemeColorName
			if v, ok := structure.FuelExpires.Value(); ok {
				powerText = "Full Power"
				powerColor = theme.ColorNameSuccess
				fuelText = v.Format(app.DateTimeFormat)
			} else {
				powerText = "Low Power / Abandoned"
				powerColor = theme.ColorNameWarning
				fuelText = "N/A"
			}

			fi := []*widget.FormItem{
				widget.NewFormItem("Owner", makeCorporationActionLabel(
					corporationID,
					corporationName,
					u.InfoViewer().Show,
				)),
				widget.NewFormItem("Name", widget.NewLabel(structure.NameShort())),
				widget.NewFormItem("Type", ui.MakeLinkLabelWithWrap(structure.Type.Name, func() {
					u.InfoViewer().ShowType(structure.Type.ID, 0)
				})),
				widget.NewFormItem("System", makeSolarSystemLabel(structure.System, u.InfoViewer().Show)),
				widget.NewFormItem("Region", ui.MakeLinkLabel(structure.System.Constellation.Region.Name, func() {
					u.InfoViewer().Show(structure.System.Constellation.Region.ToEveEntity())
				})),
				widget.NewFormItem("Services", widget.NewRichText(services...)),
				widget.NewFormItem("Fuel Expires", widget.NewRichText(xwidget.RichTextSegmentsFromText(fuelText, widget.RichTextStyle{
					ColorName: powerColor,
				})...)),
				widget.NewFormItem("State", widget.NewRichText(xwidget.RichTextSegmentsFromText(structure.State.Display(), widget.RichTextStyle{
					ColorName: structure.State.Color(),
				})...)),
				widget.NewFormItem("Power Mode", widget.NewRichText(xwidget.RichTextSegmentsFromText(powerText, widget.RichTextStyle{
					ColorName: powerColor,
				})...)),
				widget.NewFormItem("Timer Start", widget.NewLabel(structure.StateTimerStart.StringFunc("-", func(v time.Time) string {
					return v.Format(app.DateTimeFormat)
				}))),
				widget.NewFormItem("Timer End", widget.NewLabel(structure.StateTimerEnd.StringFunc("-", func(v time.Time) string {
					return v.Format(app.DateTimeFormat)
				}))),
				widget.NewFormItem("Unanchor At", widget.NewLabel(structure.UnanchorsAt.StringFunc("-", func(v time.Time) string {
					return v.Format(app.DateTimeFormat)
				}))),
				widget.NewFormItem("Reinforce Hour", widget.NewLabel(structure.ReinforceHour.StringFunc("-", func(v int64) string {
					return fmt.Sprintf("%d:00", v)
				}))),
				widget.NewFormItem("Next Reinforce Apply", widget.NewLabel(structure.NextReinforceApply.StringFunc("-", func(v time.Time) string {
					return v.Format(app.DateTimeFormat)
				}))),
				widget.NewFormItem("Next Reinforce Hour", widget.NewLabel(structure.NextReinforceHour.StringFunc("-", func(v int64) string {
					return fmt.Sprintf("%d:00", v)
				}))),
			}

			f := widget.NewForm(fi...)
			f.Orientation = widget.Adaptive
			ui.MakeDetailWindow(ui.MakeDetailWindowParams{
				Content: f,
				ImageAction: func() {
					u.InfoViewer().ShowType(structure.Type.ID, 0)
				},
				ImageLoader: func(setter func(r fyne.Resource)) {
					u.EVEImage().InventoryTypeIconAsync(structure.Type.ID, 512, setter)
				},
				Title:  structure.DisplayName(),
				Window: w,
			})
			w.Show()
		})
	}()
}

func makeCorporationActionLabel(id int64, name string, action func(o *app.EveEntity)) fyne.CanvasObject {
	o := &app.EveEntity{
		ID:       id,
		Name:     name,
		Category: app.EveEntityCorporation,
	}
	return ui.MakeEveEntityActionLabel(o, action)
}

func makeSolarSystemLabel(o *app.EveSolarSystem, show func(o *app.EveEntity)) fyne.CanvasObject {
	if o == nil {
		return widget.NewLabel("?")
	}
	segs := slices.Concat(
		o.SecurityStatusRichText(),
		xwidget.RichTextSegmentsFromText(" ", widget.RichTextStyleInline),
		xwidget.RichTextSegmentsFromText(o.Name, widget.RichTextStyle{
			ColorName: theme.ColorNamePrimary,
		}))
	x := xwidget.NewTappableRichText(segs, func() {
		o := &app.EveEntity{
			ID:       o.ID,
			Name:     o.Name,
			Category: app.EveEntitySolarSystem,
		}
		show(o)
	})
	x.Wrapping = fyne.TextWrapWord
	return x
}

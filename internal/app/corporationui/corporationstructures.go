package corporationui

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
	awidget "github.com/ErikKalkoken/evebuddy/internal/app/commonui"
	"github.com/ErikKalkoken/evebuddy/internal/app/uiservices"
	"github.com/ErikKalkoken/evebuddy/internal/app/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/app/xwindow"
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

type corporationStructureRow struct {
	corporationID      int64
	corporationName    string
	fuelExpires        optional.Optional[time.Time]
	fuelSort           time.Time
	isFullPower        bool
	isReinforced       bool
	regionID           int64
	regionName         string
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

func (r corporationStructureRow) fuelExpiresDisplay() []widget.RichTextSegment {
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

type CorporationStructures struct {
	widget.BaseWidget

	OnUpdate func(count int)

	footer            *widget.Label
	columnSorter      *xwidget.ColumnSorter[corporationStructureRow]
	corporation       atomic.Pointer[app.Corporation]
	main              fyne.CanvasObject
	rows              []corporationStructureRow
	rowsFiltered      []corporationStructureRow
	selectPower       *kxwidget.FilterChipSelect
	selectRegion      *kxwidget.FilterChipSelect
	selectService     *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectState       *kxwidget.FilterChipSelect
	selectType        *kxwidget.FilterChipSelect
	sortButton        *xwidget.SortButton
	s                 uiservices.UIServices
}

const (
	structuresColName = iota + 1
	structuresColType
	structuresColFuelExpires
	structuresColState
	structuresColServices
)

func NewCorporationStructures(s uiservices.UIServices) *CorporationStructures {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[corporationStructureRow]{{
		ID:    structuresColName,
		Label: "Name",
		Width: 250,
		Sort: func(a, b corporationStructureRow) int {
			return xstrings.CompareIgnoreCase(a.structureName, b.structureName)
		},
		Update: func(r corporationStructureRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.structureName)
		},
	}, awidget.MakeEveEntityColumn(awidget.MakeEveEntityColumnParams[corporationStructureRow]{
		ColumnID: structuresColType,
		EIS:      s.EVEImage(),
		Label:    "Type",
		GetEntity: func(r corporationStructureRow) *app.EveEntity {
			return &app.EveEntity{
				Category: app.EveEntityInventoryType,
				ID:       r.typeID,
				Name:     r.typeName,
			}
		},
	}), {
		ID:    structuresColFuelExpires,
		Label: "Fuel Expires",
		Width: 150,
		Sort: func(a, b corporationStructureRow) int {
			return a.fuelSort.Compare(b.fuelSort)
		},
		Update: func(r corporationStructureRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.fuelExpiresDisplay())
		},
	}, {
		ID:    structuresColState,
		Label: "State",
		Width: 150,
		Update: func(r corporationStructureRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.stateText, widget.RichTextStyle{
				ColorName: r.stateColor,
			})
		},
	}, {
		ID:    structuresColServices,
		Label: "Services",
		Width: 200,
		Update: func(r corporationStructureRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.servicesText)
		},
	}})
	a := &CorporationStructures{
		columnSorter: xwidget.NewColumnSorter(columns, structuresColName, xwidget.SortAsc),
		footer:       newLabelWithWrapping(),
		s:            s,
	}
	a.ExtendBaseWidget(a)
	if !app.IsMobile() {
		a.main = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync, func(_ int, r corporationStructureRow) {
				go showCorporationStructureWindowAsync(context.Background(), s, r.corporationID, r.structureID, r.solarSystemName)
			},
		)
	} else {
		a.main = xwidget.MakeDataList(
			columns,
			&a.rowsFiltered,
			func(col int, r corporationStructureRow) []widget.RichTextSegment {
				switch col {
				case structuresColType:
					return xwidget.RichTextSegmentsFromText(r.typeName)
				case structuresColName:
					return xwidget.RichTextSegmentsFromText(r.structureName)
				case structuresColFuelExpires:
					return r.fuelExpiresDisplay()
				case structuresColState:
					return xwidget.RichTextSegmentsFromText(r.stateText, widget.RichTextStyle{
						ColorName: r.stateColor,
					})
				case structuresColServices:
					return xwidget.RichTextSegmentsFromText(r.servicesText)
				}
				return xwidget.RichTextSegmentsFromText("?")
			},
			func(r corporationStructureRow) {
				go showCorporationStructureWindowAsync(context.Background(), s, r.corporationID, r.structureID, r.solarSystemName)
			},
		)
	}

	// filter
	a.selectRegion = kxwidget.NewFilterChipSelect("Region", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectSolarSystem = kxwidget.NewFilterChipSelect("System", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectType = kxwidget.NewFilterChipSelect("Type", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectState = kxwidget.NewFilterChipSelect("State", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectService = kxwidget.NewFilterChipSelect("Service", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.s.MainWindow())
	a.selectPower = kxwidget.NewFilterChipSelect("Power", []string{
		structuresPowerHigh,
		structuresPowerLow,
	}, func(_ string) {
		a.filterRowsAsync(-1)
	})

	// Signals
	a.s.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		a.update(ctx)
	})
	a.s.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		if corporationIDOrZero(a.corporation.Load()) != arg.CorporationID {
			return
		}
		if arg.Section != app.SectionCorporationStructures {
			return
		}
		a.update(ctx)
	})
	a.s.Signals().RefreshTickerExpired.AddListener(func(ctx context.Context, _ struct{}) {
		fyne.Do(func() {
			a.update(ctx)
		})
	})
	return a
}

func (a *CorporationStructures) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectType, a.selectState, a.selectSolarSystem, a.selectRegion, a.selectService, a.selectPower)
	if app.IsMobile() {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(container.NewHScroll(filter), a.footer, nil, nil, a.main)
	return widget.NewSimpleRenderer(c)
}

func (a *CorporationStructures) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	region := a.selectRegion.Selected
	solarSystem := a.selectSolarSystem.Selected
	state := a.selectState.Selected
	service := a.selectService.Selected
	type_ := a.selectType.Selected
	power := a.selectPower.Selected
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		// filter
		if region != "" {
			rows = slices.DeleteFunc(rows, func(r corporationStructureRow) bool {
				return r.regionName != region
			})
		}
		if solarSystem != "" {
			rows = slices.DeleteFunc(rows, func(r corporationStructureRow) bool {
				return r.solarSystemName != solarSystem
			})
		}
		if state != "" {
			rows = slices.DeleteFunc(rows, func(r corporationStructureRow) bool {
				return r.stateDisplay != state
			})
		}
		if service != "" {
			rows = slices.DeleteFunc(rows, func(r corporationStructureRow) bool {
				return !r.services.Contains(service)
			})
		}
		if type_ != "" {
			rows = slices.DeleteFunc(rows, func(r corporationStructureRow) bool {
				return r.typeName != type_
			})
		}
		if power != "" {
			rows = slices.DeleteFunc(rows, func(r corporationStructureRow) bool {
				switch power {
				case structuresPowerHigh:
					return !r.isFullPower
				case structuresPowerLow:
					return r.isFullPower
				}
				return true
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		// set data & refresh
		selectOptions := xslices.Map(rows, func(r corporationStructureRow) string {
			return r.regionName
		})
		solarSystemOptions := xslices.Map(rows, func(r corporationStructureRow) string {
			return r.solarSystemName
		})
		stateOptions := xslices.Map(rows, func(r corporationStructureRow) string {
			return r.stateDisplay
		})
		servicesOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r corporationStructureRow) set.Set[string] {
			return r.services
		})...).All())
		typeOptions := xslices.Map(rows, func(r corporationStructureRow) string {
			return r.typeName
		})

		footer := fmt.Sprintf("Showing %s / %s structures", ihumanize.Comma(len(rows)), ihumanize.Comma(totalRows))

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectRegion.SetOptions(selectOptions)
			a.selectSolarSystem.SetOptions(solarSystemOptions)
			a.selectState.SetOptions(stateOptions)
			a.selectService.SetOptions(servicesOptions)
			a.selectType.SetOptions(typeOptions)
			a.rowsFiltered = rows
			a.main.Refresh()
		})
	}()
}

func (a *CorporationStructures) update(ctx context.Context) {
	clear := func() {
		fyne.Do(func() {
			a.rows = xslices.Reset(a.rows)
			a.filterRowsAsync(-1)
		})
	}
	corporationID := corporationIDOrZero(a.corporation.Load())
	if corporationID == 0 {
		clear()
		return
	}
	rows, err := a.fetchData(ctx, corporationID)
	if err != nil {
		slog.Error("Failed to refresh corporation structures UI", "err", err)
		clear()
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + app.ErrorDisplay(err)
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
		a.filterRowsAsync(-1)
		if a.OnUpdate != nil {
			a.OnUpdate(reinforceCount)
		}
	})
}

func (a *CorporationStructures) fetchData(ctx context.Context, corporationID int64) ([]corporationStructureRow, error) {
	if corporationID == 0 {
		return []corporationStructureRow{}, nil
	}
	structures, err := a.s.Corporation().ListStructures(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	corporationNames, err := a.s.Corporation().CorporationNames(ctx)
	if err != nil {
		return nil, err
	}
	var rows []corporationStructureRow
	for _, s := range structures {
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

		rows = append(rows, corporationStructureRow{
			corporationID:      corporationID,
			corporationName:    corporationNames[corporationID],
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
		})
	}
	return rows, nil
}

func showCorporationStructureWindowAsync(ctx context.Context, s uiservices.UIServices, corporationID int64, structureID int64, title string) {
	w, created := s.GetOrCreateWindow(
		fmt.Sprintf("corporationstructure-%d-%d", corporationID, structureID),
		title,
	)
	if !created {
		w.Show()
		return
	}

	go func() {
		structure, err := s.Corporation().GetStructure(ctx, corporationID, structureID)
		if err != nil {
			xdialog.ShowErrorAndLog("Failed to show structure", err, s.MainWindow())
			return
		}
		corporationNames, err := s.Corporation().CorporationNames(ctx)
		if err != nil {
			xdialog.ShowErrorAndLog("Failed to show structure", err, s.MainWindow())
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
					s.InfoWindow().ShowEntity,
				)),
				widget.NewFormItem("Name", widget.NewLabel(structure.NameShort())),
				widget.NewFormItem("Type", makeLinkLabelWithWrap(structure.Type.Name, func() {
					s.InfoWindow().ShowType(structure.Type.ID)
				})),
				widget.NewFormItem("System", makeSolarSystemLabel(structure.System, s.InfoWindow().ShowEntity)),
				widget.NewFormItem("Region", makeLinkLabel(structure.System.Constellation.Region.Name, func() {
					s.InfoWindow().Show(app.EveEntityRegion, structure.System.Constellation.Region.ID)
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
			xwindow.Set(xwindow.Params{
				Content: f,
				ImageAction: func() {
					s.InfoWindow().ShowType(structure.Type.ID)
				},
				ImageLoader: func(setter func(r fyne.Resource)) {
					s.EVEImage().InventoryTypeIconAsync(structure.Type.ID, 512, setter)
				},
				Title:  structure.DisplayName(),
				Window: w,
			})
			w.Show()
		})
	}()
}

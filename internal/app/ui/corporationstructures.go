package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

const (
	structuresPowerLow  = "Low Power"
	structuresPowerHigh = "High Power"
)

type corporationStructureRow struct {
	corporationID      int32
	corporationName    string
	fuelExpires        optional.Optional[time.Time]
	fuelSort           time.Time
	isFullPower        bool
	isReinforced       bool
	regionID           int32
	regionName         string
	services           set.Set[string]
	servicesText       string
	solarSystemDisplay []widget.RichTextSegment
	solarSystemID      int32
	solarSystemName    string
	stateColor         fyne.ThemeColorName
	stateDisplay       string
	stateText          string
	structureID        int64
	structureName      string
	typeID             int32
	typeName           string
}

func (r corporationStructureRow) fuelExpiresDisplay() []widget.RichTextSegment {
	var text string
	var color fyne.ThemeColorName
	if r.fuelExpires.IsEmpty() {
		color = theme.ColorNameWarning
		text = "Low Power"
	} else {
		color = theme.ColorNameForeground
		text = ihumanize.Duration(time.Until(r.fuelExpires.ValueOrZero()))
	}
	return iwidget.RichTextSegmentsFromText(text, widget.RichTextStyle{
		ColorName: color,
	})
}

type corporationStructures struct {
	widget.BaseWidget

	OnUpdate func(count int)

	bottom            *widget.Label
	columnSorter      *iwidget.ColumnSorter[corporationStructureRow]
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
	sortButton        *iwidget.SortButton
	u                 *baseUI
}

const (
	structuresColName = iota + 1
	structuresColType
	structuresColFuelExpires
	structuresColState
	structuresColServices
)

func newCorporationStructures(u *baseUI) *corporationStructures {
	columns := iwidget.NewDataColumns([]iwidget.DataColumn[corporationStructureRow]{{
		ID:    structuresColName,
		Label: "Name",
		Width: 250,
		Sort: func(a, b corporationStructureRow) int {
			return xstrings.CompareIgnoreCase(a.structureName, b.structureName)
		},
		Update: func(r corporationStructureRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).SetWithText(r.structureName)
		},
	}, {
		ID:    structuresColType,
		Label: "Type",
		Width: 150,
		Sort: func(a, b corporationStructureRow) int {
			return strings.Compare(a.typeName, b.typeName)
		},
		Create: func() fyne.CanvasObject {
			icon := iwidget.NewImageFromResource(
				icons.BlankSvg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			name := widget.NewLabel("Template")
			return container.NewBorder(nil, nil, container.NewCenter(icon), nil, name)
		},
		Update: func(r corporationStructureRow, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(r.typeName)
			x := border[1].(*fyne.Container).Objects[0].(*canvas.Image)
			u.eis.InventoryTypeIconAsync(r.typeID, app.IconPixelSize, func(r fyne.Resource) {
				x.Resource = r
				x.Refresh()
			})
		},
	}, {
		ID:    structuresColFuelExpires,
		Label: "Fuel Expires",
		Width: 150,
		Sort: func(a, b corporationStructureRow) int {
			return a.fuelSort.Compare(b.fuelSort)
		},
		Update: func(r corporationStructureRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).Set(r.fuelExpiresDisplay())
		},
	}, {
		ID:    structuresColState,
		Label: "State",
		Width: 150,
		Update: func(r corporationStructureRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).SetWithText(r.stateText, widget.RichTextStyle{
				ColorName: r.stateColor,
			})
		},
	}, {
		ID:    structuresColServices,
		Label: "Services",
		Width: 200,
		Update: func(r corporationStructureRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).SetWithText(r.servicesText)
		},
	}})
	a := &corporationStructures{
		columnSorter: iwidget.NewColumnSorter(columns, structuresColName, iwidget.SortAsc),
		rows:         make([]corporationStructureRow, 0),
		rowsFiltered: make([]corporationStructureRow, 0),
		bottom:       makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	if !a.u.isMobile {
		a.main = iwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := iwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRows, func(_ int, r corporationStructureRow) {
				showCorporationStructureWindow(a.u, r.corporationID, r.structureID)
			},
		)
	} else {
		a.main = iwidget.MakeDataList(
			columns,
			&a.rowsFiltered,
			func(col int, r corporationStructureRow) []widget.RichTextSegment {
				switch col {
				case structuresColType:
					return iwidget.RichTextSegmentsFromText(r.typeName)
				case structuresColName:
					return iwidget.RichTextSegmentsFromText(r.structureName)
				case structuresColFuelExpires:
					return r.fuelExpiresDisplay()
				case structuresColState:
					return iwidget.RichTextSegmentsFromText(r.stateText, widget.RichTextStyle{
						ColorName: r.stateColor,
					})
				case structuresColServices:
					return iwidget.RichTextSegmentsFromText(r.servicesText)
				}
				return iwidget.RichTextSegmentsFromText("?")
			},
			func(r corporationStructureRow) {
				showCorporationStructureWindow(a.u, r.corporationID, r.structureID)
			},
		)
	}

	a.selectRegion = kxwidget.NewFilterChipSelect("Region", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectSolarSystem = kxwidget.NewFilterChipSelect("System", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectType = kxwidget.NewFilterChipSelect("Type", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectState = kxwidget.NewFilterChipSelect("State", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectService = kxwidget.NewFilterChipSelect("Service", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRows(-1)
	}, a.u.window)
	a.selectPower = kxwidget.NewFilterChipSelect("Power", []string{
		structuresPowerHigh,
		structuresPowerLow,
	}, func(_ string) {
		a.filterRows(-1)
	})

	a.u.currentCorporationExchanged.AddListener(func(_ context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		a.update()
	})
	a.u.corporationSectionChanged.AddListener(func(_ context.Context, arg corporationSectionUpdated) {
		if corporationIDOrZero(a.corporation.Load()) != arg.corporationID {
			return
		}
		if arg.section != app.SectionCorporationStructures {
			return
		}
		a.update()
	})
	a.u.refreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.update()
		})
	})
	return a
}

func (a *corporationStructures) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectType, a.selectState, a.selectSolarSystem, a.selectRegion, a.selectService, a.selectPower)
	if a.u.isMobile {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(container.NewHScroll(filter), a.bottom, nil, nil, a.main)
	return widget.NewSimpleRenderer(c)
}

func (a *corporationStructures) filterRows(sortCol int) {
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

		fyne.Do(func() {
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

func (a *corporationStructures) update() {
	rows := make([]corporationStructureRow, 0)
	t, i, err := func() (string, widget.Importance, error) {
		cc, err := a.fetchData(corporationIDOrZero(a.corporation.Load()))
		if err != nil {
			return "", 0, err
		}
		if len(cc) == 0 {
			return "No structures", widget.LowImportance, nil
		}
		rows = cc
		return "", widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh corporation structures UI", "err", err)
		t = "ERROR: " + a.u.humanizeError(err)
		i = widget.DangerImportance
	}
	fyne.Do(func() {
		if t != "" {
			a.bottom.Text = t
			a.bottom.Importance = i
			a.bottom.Refresh()
			a.bottom.Show()
		} else {
			a.bottom.Hide()
		}
	})
	if a.OnUpdate != nil {
		var reinforceCount int
		for _, r := range rows {
			if r.isReinforced {
				reinforceCount++
			}
		}
		a.OnUpdate(reinforceCount)
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRows(-1)
	})
}

func (a *corporationStructures) fetchData(corporationID int32) ([]corporationStructureRow, error) {
	if corporationID == 0 {
		return []corporationStructureRow{}, nil
	}
	structures, err := a.u.rs.ListStructures(context.Background(), corporationID)
	if err != nil {
		return nil, err
	}
	rows := make([]corporationStructureRow, 0)
	for _, s := range structures {
		stateText := s.State.DisplayShort()
		if !s.StateTimerEnd.IsEmpty() {
			var x string
			end := s.StateTimerEnd.ValueOrZero()
			d := time.Until(end)
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
			corporationName:    a.u.scs.CorporationName(corporationID),
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
			structureName:      s.Name,
			typeID:             s.Type.ID,
			typeName:           s.Type.Name,
		})
	}
	return rows, nil
}

func showCorporationStructureWindow(u *baseUI, corporationID int32, structureID int64) {
	s, err := u.rs.GetStructure(context.Background(), corporationID, structureID)
	if err != nil {
		u.showErrorDialog("Failed to fetch structure", err, u.MainWindow())
		return
	}
	corporationName := u.scs.CorporationName(corporationID)
	w, created := u.getOrCreateWindow(
		fmt.Sprintf("corporationstructure-%d-%d", corporationID, structureID),
		s.Name,
	)
	if !created {
		w.Show()
		return
	}
	var services []widget.RichTextSegment
	if len(s.Services) == 0 {
		services = iwidget.RichTextSegmentsFromText("-")
	} else {
		slices.SortFunc(s.Services, func(a, b *app.StructureService) int {
			return strings.Compare(a.Name, b.Name)
		})
		for _, x := range s.Services {
			var color fyne.ThemeColorName
			name := x.Name
			if x.State == app.StructureServiceStateOnline {
				color = theme.ColorNameForeground
			} else {
				color = theme.ColorNameDisabled
				name += " [offline]"
			}
			services = slices.Concat(services, iwidget.RichTextSegmentsFromText(name, widget.RichTextStyle{
				ColorName: color,
			}))
		}
	}

	var fuelText, powerText string
	var powerColor fyne.ThemeColorName
	if s.FuelExpires.IsEmpty() {
		powerText = "Low Power / Abandoned"
		powerColor = theme.ColorNameWarning
		fuelText = "N/A"
	} else {
		powerText = "Full Power"
		powerColor = theme.ColorNameSuccess
		fuelText = s.FuelExpires.ValueOrZero().Format(app.DateTimeFormat)
	}

	fi := []*widget.FormItem{
		widget.NewFormItem("Owner", makeCorporationActionLabel(
			corporationID,
			corporationName,
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Name", widget.NewLabel(s.NameShort())),
		widget.NewFormItem("Type", makeLinkLabelWithWrap(s.Type.Name, func() {
			u.ShowTypeInfoWindow(s.Type.ID)
		})),
		widget.NewFormItem("System", makeSolarSystemLabel(s.System, u.ShowEveEntityInfoWindow)),
		widget.NewFormItem("Region", makeLinkLabel(s.System.Constellation.Region.Name, func() {
			u.ShowInfoWindow(app.EveEntityRegion, s.System.Constellation.Region.ID)
		})),
		widget.NewFormItem("Services", widget.NewRichText(services...)),
		widget.NewFormItem("Fuel Expires", widget.NewRichText(iwidget.RichTextSegmentsFromText(fuelText, widget.RichTextStyle{
			ColorName: powerColor,
		})...)),
		widget.NewFormItem("State", widget.NewRichText(iwidget.RichTextSegmentsFromText(s.State.Display(), widget.RichTextStyle{
			ColorName: s.State.Color(),
		})...)),
		widget.NewFormItem("Power Mode", widget.NewRichText(iwidget.RichTextSegmentsFromText(powerText, widget.RichTextStyle{
			ColorName: powerColor,
		})...)),
		widget.NewFormItem("Timer Start", widget.NewLabel(s.StateTimerStart.StringFunc("-", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))),
		widget.NewFormItem("Timer End", widget.NewLabel(s.StateTimerEnd.StringFunc("-", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))),
		widget.NewFormItem("Unanchor At", widget.NewLabel(s.UnanchorsAt.StringFunc("-", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))),
		widget.NewFormItem("Reinforce Hour", widget.NewLabel(s.ReinforceHour.StringFunc("-", func(v int64) string {
			return fmt.Sprintf("%d:00", v)
		}))),
		widget.NewFormItem("Next Reinforce Apply", widget.NewLabel(s.NextReinforceApply.StringFunc("-", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))),
		widget.NewFormItem("Next Reinforce Hour", widget.NewLabel(s.NextReinforceHour.StringFunc("-", func(v int64) string {
			return fmt.Sprintf("%d:00", v)
		}))),
	}

	f := widget.NewForm(fi...)
	f.Orientation = widget.Adaptive
	setDetailWindow(detailWindowParams{
		content: f,
		imageAction: func() {
			u.ShowTypeInfoWindow(s.Type.ID)
		},
		imageLoader: func(setter func(r fyne.Resource)) {
			u.eis.InventoryTypeIconAsync(s.Type.ID, 512, setter)
		},
		title:  s.Name,
		window: w,
	})
	w.Show()
}

package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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
	fuelText           string
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

type corporationStructures struct {
	widget.BaseWidget

	OnUpdate func(count int)

	bottom            *widget.Label
	columnSorter      *iwidget.ColumnSorter
	corporation       *app.Corporation
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
	structuresColName     = 0
	structuresColType     = 1
	structuresColFuel     = 2
	structuresColState    = 3
	structuresColServices = 4
)

func newCorporationStructures(u *baseUI) *corporationStructures {
	headers := iwidget.NewDataTableDef([]iwidget.ColumnDef{{
		Col:   structuresColName,
		Label: "Name",
		Width: 250,
	}, {
		Col:   structuresColType,
		Label: "Type",
		Width: 150,
	}, {
		Col:   structuresColFuel,
		Label: "Fuel Expires",
		Width: 150,
	}, {
		Col:    structuresColState,
		Label:  "State",
		Width:  150,
		NoSort: true,
	}, {
		Col:    structuresColServices,
		Label:  "Services",
		Width:  200,
		NoSort: true,
	}})
	a := &corporationStructures{
		columnSorter: iwidget.NewColumnSorter(headers, structuresColName, iwidget.SortAsc),
		rows:         make([]corporationStructureRow, 0),
		rowsFiltered: make([]corporationStructureRow, 0),
		bottom:       makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r corporationStructureRow) []widget.RichTextSegment {
		switch col {
		case structuresColType:
			return iwidget.RichTextSegmentsFromText(r.typeName)
		case structuresColName:
			return iwidget.RichTextSegmentsFromText(r.structureName)
		case structuresColFuel:
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
		case structuresColState:
			return iwidget.RichTextSegmentsFromText(r.stateText, widget.RichTextStyle{
				ColorName: r.stateColor,
			})
		case structuresColServices:
			return iwidget.RichTextSegmentsFromText(r.servicesText)
		}
		return iwidget.RichTextSegmentsFromText("?")
	}
	if a.u.isDesktop {
		a.main = iwidget.MakeDataTable(
			headers,
			&a.rowsFiltered,
			makeCell,
			a.columnSorter,
			a.filterRows, func(_ int, r corporationStructureRow) {
				showCorporationStructureWindow(a.u, r.corporationID, r.structureID)
			})
	} else {
		a.main = iwidget.MakeDataList(headers, &a.rowsFiltered, makeCell, func(r corporationStructureRow) {
			showCorporationStructureWindow(a.u, r.corporationID, r.structureID)
		})
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
	a.sortButton = a.columnSorter.NewSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)
	a.selectPower = kxwidget.NewFilterChipSelect("Power", []string{
		structuresPowerHigh,
		structuresPowerLow,
	}, func(_ string) {
		a.filterRows(-1)
	})

	a.u.currentCorporationExchanged.AddListener(func(_ context.Context, c *app.Corporation) {
		a.corporation = c
	})
	a.u.corporationSectionChanged.AddListener(func(_ context.Context, arg corporationSectionUpdated) {
		if corporationIDOrZero(a.corporation) != arg.corporationID {
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
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(container.NewHScroll(filter), a.bottom, nil, nil, a.main)
	return widget.NewSimpleRenderer(c)
}

func (a *corporationStructures) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectRegion.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationStructureRow) bool {
			return r.regionName == x
		})
	}
	if x := a.selectSolarSystem.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationStructureRow) bool {
			return r.solarSystemName == x
		})
	}
	if x := a.selectState.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationStructureRow) bool {
			return r.stateDisplay == x
		})
	}
	if x := a.selectService.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationStructureRow) bool {
			return r.services.Contains(x)
		})
	}
	if x := a.selectType.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationStructureRow) bool {
			return r.typeName == x
		})
	}
	if x := a.selectPower.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationStructureRow) bool {
			switch x {
			case structuresPowerHigh:
				return r.isFullPower
			case structuresPowerLow:
				return !r.isFullPower
			}
			return false
		})
	}
	// sort
	a.columnSorter.Sort(sortCol, func(sortCol int, dir iwidget.SortDir) {
		slices.SortFunc(rows, func(a, b corporationStructureRow) int {
			var x int
			switch sortCol {
			case structuresColType:
				x = strings.Compare(a.typeName, b.typeName)
			case structuresColName:
				x = strings.Compare(a.structureName, b.structureName)
			case structuresColFuel:
				x = cmp.Compare(time.Until(a.fuelSort), time.Until(b.fuelSort))
			}
			if dir == iwidget.SortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	// set data & refresh
	a.selectRegion.SetOptions(xslices.Map(rows, func(r corporationStructureRow) string {
		return r.regionName
	}))
	a.selectSolarSystem.SetOptions(xslices.Map(rows, func(r corporationStructureRow) string {
		return r.solarSystemName
	}))
	a.selectState.SetOptions(xslices.Map(rows, func(r corporationStructureRow) string {
		return r.stateDisplay
	}))
	a.selectService.SetOptions(slices.Sorted(set.Union(xslices.Map(rows, func(r corporationStructureRow) set.Set[string] {
		return r.services
	})...).All()))
	a.selectType.SetOptions(xslices.Map(rows, func(r corporationStructureRow) string {
		return r.typeName
	}))
	a.rowsFiltered = rows
	a.main.Refresh()
}

func (a *corporationStructures) update() {
	rows := make([]corporationStructureRow, 0)
	t, i, err := func() (string, widget.Importance, error) {
		cc, err := a.fetchData(corporationIDOrZero(a.corporation))
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
		servicesText := stringsJoinsOrEmpty(slices.Sorted(services.All()), ", ", "-")
		region := s.System.Constellation.Region

		rows = append(rows, corporationStructureRow{
			corporationID:      corporationID,
			corporationName:    a.u.scs.CorporationName(corporationID),
			fuelExpires:        s.FuelExpires,
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
		"Corporation Structure",
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
	subTitle := fmt.Sprintf("Corporation Structure: %s", s.Name)
	setDetailWindow(detailWindowParams{
		content: f,
		imageAction: func() {
			u.ShowTypeInfoWindow(s.Type.ID)
		},
		imageLoader: func() (fyne.Resource, error) {
			return u.eis.InventoryTypeIcon(s.Type.ID, 512)
		},
		title:  subTitle,
		window: w,
	})
	w.Show()
}

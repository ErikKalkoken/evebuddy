package ui

import (
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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type corporationStructureRow struct {
	corporationID      int32
	corporationName    string
	regionID           int32
	regionName         string
	solarSystemDisplay []widget.RichTextSegment
	solarSystemID      int32
	solarSystemName    string
	stateDisplay       string
	stateColor         fyne.ThemeColorName
	stateText          string
	structureID        int64
	structureName      string
	structureNameShort string
	typeID             int32
	typeName           string
}

type corporationStructures struct {
	widget.BaseWidget

	corporation       *app.Corporation
	body              fyne.CanvasObject
	columnSorter      *columnSorter
	rows              []corporationStructureRow
	rowsFiltered      []corporationStructureRow
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectType        *kxwidget.FilterChipSelect
	selectState       *kxwidget.FilterChipSelect
	sortButton        *sortButton
	bottom            *widget.Label
	u                 *baseUI
}

func newCorporationStructures(u *baseUI) *corporationStructures {
	headers := []headerDef{
		{label: "System", width: 150},
		{label: "Region", width: columnWidthRegion},
		{label: "Type", width: 150},
		{label: "Name", width: 250},
		{label: "State", width: 150},
	}
	a := &corporationStructures{
		columnSorter: newColumnSorterWithInit(headers, 0, sortAsc),
		rows:         make([]corporationStructureRow, 0),
		rowsFiltered: make([]corporationStructureRow, 0),
		bottom:       makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r corporationStructureRow) []widget.RichTextSegment {
		switch col {
		case 0:
			return r.solarSystemDisplay
		case 1:
			return iwidget.RichTextSegmentsFromText(r.regionName)
		case 2:
			return iwidget.RichTextSegmentsFromText(r.typeName)
		case 3:
			return iwidget.RichTextSegmentsFromText(r.structureNameShort)
		case 4:
			return iwidget.RichTextSegmentsFromText(r.stateDisplay, widget.RichTextStyle{
				ColorName: r.stateColor,
			})
		}
		return iwidget.RichTextSegmentsFromText("?")
	}
	if a.u.isDesktop {
		a.body = makeDataTable(
			headers,
			&a.rowsFiltered,
			makeCell,
			a.columnSorter,
			a.filterRows, func(_ int, r corporationStructureRow) {
				showCorporationStructureWindow(a.u, r.corporationID, r.structureID)
			})
	} else {
		a.body = makeDataList(headers, &a.rowsFiltered, makeCell, func(r corporationStructureRow) {
			showCorporationStructureWindow(a.u, r.corporationID, r.structureID)
		})
	}

	a.selectRegion = kxwidget.NewFilterChipSelectWithSearch("Region", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectSolarSystem = kxwidget.NewFilterChipSelectWithSearch("System", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectType = kxwidget.NewFilterChipSelect("Type", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectState = kxwidget.NewFilterChipSelect("State", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)

	a.u.corporationExchanged.AddListener(
		func(_ context.Context, c *app.Corporation) {
			a.corporation = c
		},
	)
	a.u.corporationSectionChanged.AddListener(func(_ context.Context, arg corporationSectionUpdated) {
		if corporationIDOrZero(a.corporation) != arg.corporationID {
			return
		}
		if arg.section != app.SectionCorporationStructures {
			return
		}
		a.update()
	})
	return a
}

func (a *corporationStructures) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectSolarSystem, a.selectRegion, a.selectType, a.selectState)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(container.NewHScroll(filter), a.bottom, nil, nil, a.body)
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
	if x := a.selectType.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationStructureRow) bool {
			return r.typeName == x
		})
	}
	if x := a.selectState.Selected; x != "" {
		rows = xslices.Filter(rows, func(r corporationStructureRow) bool {
			return r.stateText == x
		})
	}
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b corporationStructureRow) int {
			var x int
			switch sortCol {
			case 0:
				x = strings.Compare(a.solarSystemName, b.solarSystemName)
			case 1:
				x = strings.Compare(a.regionName, b.regionName)
			case 2:
				x = strings.Compare(a.typeName, b.typeName)
			case 3:
				x = strings.Compare(a.structureNameShort, b.structureNameShort)
			case 4:
				x = strings.Compare(a.stateText, b.stateText)
			}
			if dir == sortAsc {
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
	a.selectType.SetOptions(xslices.Map(rows, func(r corporationStructureRow) string {
		return r.typeName
	}))
	a.selectState.SetOptions(xslices.Map(rows, func(r corporationStructureRow) string {
		return r.stateText
	}))
	a.rowsFiltered = rows
	a.body.Refresh()
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
		region := s.System.Constellation.Region
		var color fyne.ThemeColorName
		switch s.State {
		case app.StructureStateAnchoring, app.StructureStateAnchorVulnerable, app.StructureStateDeployVulnerable:
			color = theme.ColorNameWarning
		case app.StructureStateArmorReinforce, app.StructureStateHullReinforce:
			color = theme.ColorNameError
		case app.StructureStateShieldVulnerable:
			color = theme.ColorNameSuccess
		default:
			color = theme.ColorNameForeground
		}
		rows = append(rows, corporationStructureRow{
			corporationID:      corporationID,
			corporationName:    a.u.scs.CorporationName(corporationID),
			regionID:           region.ID,
			regionName:         region.Name,
			solarSystemDisplay: s.System.DisplayRichText(),
			solarSystemID:      s.System.ID,
			solarSystemName:    s.System.Name,
			stateDisplay:       s.State.Display(),
			stateColor:         color,
			structureName:      s.Name,
			structureNameShort: s.NameShort(),
			typeID:             s.Type.ID,
			typeName:           s.Type.Name,
			structureID:        s.StructureID,
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
		corporationName,
	)
	if !created {
		w.Show()
		return
	}
	fi := []*widget.FormItem{
		widget.NewFormItem("Owner", makeCorporationActionLabel(
			corporationID,
			corporationName,
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("System", makeLinkLabel(s.System.Name, func() {
			u.ShowInfoWindow(app.EveEntitySolarSystem, s.System.ID)
		})),
		widget.NewFormItem("Region", makeLinkLabel(s.System.Constellation.Region.Name, func() {
			u.ShowInfoWindow(app.EveEntityRegion, s.System.Constellation.Region.ID)
		})),
		widget.NewFormItem("Type", makeLinkLabelWithWrap(s.Type.Name, func() {
			u.ShowTypeInfoWindow(s.Type.ID)
		})),
		widget.NewFormItem("Name", widget.NewLabel(s.NameShort())),
		widget.NewFormItem("State", widget.NewLabel(s.State.Display())),
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

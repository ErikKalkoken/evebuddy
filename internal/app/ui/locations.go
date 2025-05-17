package ui

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

	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type locationRow struct {
	characterName   string
	locationDisplay []widget.RichTextSegment
	locationID      int64
	locationName    string
	regionName      string
	solarSystemName string
	shipName        string
}

type locations struct {
	widget.BaseWidget

	columnSorter      *columnSorter
	rows              []locationRow
	rowsFiltered      []locationRow
	body              fyne.CanvasObject
	top               *widget.Label
	u                 *BaseUI
	selectSolarSystem *selectFilter
	selectRegion      *selectFilter
	sortButton        *sortButton
}

func newLocations(u *BaseUI) *locations {
	headers := []headerDef{
		{Text: "Character", Width: columnWidthCharacter},
		{Text: "Location", Width: columnWidthLocation},
		{Text: "Region", Width: columnWidthRegion},
		{Text: "Ship", Width: 150},
	}
	a := &locations{
		columnSorter: newColumnSorterWithInit(headers, 0, sortAsc),
		rows:         make([]locationRow, 0),
		rowsFiltered: make([]locationRow, 0),
		top:          makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	if a.u.isDesktop {
		a.body = makeDataTableWithSort(
			headers,
			&a.rowsFiltered,
			func(col int, r locationRow) []widget.RichTextSegment {
				switch col {
				case 0:
					return iwidget.NewRichTextSegmentFromText(r.characterName)
				case 1:
					return r.locationDisplay
				case 2:
					return iwidget.NewRichTextSegmentFromText(r.regionName)
				case 3:
					return iwidget.NewRichTextSegmentFromText(r.shipName)
				}
				return iwidget.NewRichTextSegmentFromText("?")
			},
			a.columnSorter,
			a.filterRows, func(_ int, r locationRow) {
				a.u.ShowLocationInfoWindow(r.locationID)
			})
	} else {
		a.body = a.makeDataList()
	}

	a.selectRegion = newSelectFilter("Any region", func() {
		a.filterRows(-1)
	})
	a.selectSolarSystem = newSelectFilter("Any system", func() {
		a.filterRows(-1)
	})

	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)
	return a
}

func (a *locations) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectSolarSystem, a.selectRegion)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(container.NewHScroll(filter), nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *locations) makeDataList() *widget.List {
	p := theme.Padding()
	var l *widget.List
	l = widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Template", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			title.Wrapping = fyne.TextWrapWord
			location := widget.NewRichTextWithText("Template")
			location.Wrapping = fyne.TextWrapWord
			ship := widget.NewLabel("Template")
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				title,
				location,
				ship,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			c := co.(*fyne.Container).Objects
			c[0].(*widget.Label).SetText(r.characterName)
			iwidget.SetRichText(c[1].(*widget.RichText), r.locationDisplay...)
			c[2].(*widget.Label).SetText(r.shipName)
			l.SetItemHeight(id, co.(*fyne.Container).MinSize().Height)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		a.u.ShowLocationInfoWindow(a.rowsFiltered[id].locationID)
	}
	return l
}

func (a *locations) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	a.selectRegion.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(r locationRow) bool {
			return r.regionName == selected
		})
	})
	a.selectSolarSystem.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(r locationRow) bool {
			return r.solarSystemName == selected
		})
	})
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b locationRow) int {
			var x int
			switch sortCol {
			case 0:
				x = strings.Compare(a.characterName, b.characterName)
			case 1:
				x = strings.Compare(a.locationName, b.locationName)
			case 2:
				x = strings.Compare(a.regionName, b.regionName)
			case 3:
				x = strings.Compare(a.shipName, b.shipName)
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	a.selectRegion.setOptions(xiter.MapSlice(rows, func(r locationRow) string {
		return r.regionName
	}))
	a.selectSolarSystem.setOptions(xiter.MapSlice(rows, func(r locationRow) string {
		return r.solarSystemName
	}))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *locations) update() {
	rows := make([]locationRow, 0)
	t, i, err := func() (string, widget.Importance, error) {
		cc, count, err := a.fetchData(a.u.services())
		if err != nil {
			return "", 0, err
		}
		if len(cc) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		rows = cc
		s := fmt.Sprintf("%d characters • %d locations", len(cc), count)
		return s, widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh locations UI", "err", err)
		t = "ERROR: " + a.u.humanizeError(err)
		i = widget.DangerImportance
	}
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRows(-1)
	})
}

func (*locations) fetchData(s services) ([]locationRow, int, error) {
	ctx := context.TODO()
	characters, err := s.cs.ListCharacters(ctx)
	if err != nil {
		return nil, 0, err
	}
	var locationIDs set.Set[int64]
	rows := make([]locationRow, 0)
	for _, c := range characters {
		if c.EveCharacter == nil || c.Location == nil {
			continue
		}
		r := locationRow{
			characterName:   c.EveCharacter.Name,
			locationDisplay: c.Location.DisplayRichText(),
			locationID:      c.Location.ID,
			locationName:    c.Location.DisplayName(),
		}
		if c.Location.SolarSystem != nil {
			r.regionName = c.Location.SolarSystem.Constellation.Region.Name
			r.solarSystemName = c.Location.SolarSystem.Name
		}
		if c.Ship != nil {
			r.shipName = c.Ship.Name
		}
		rows = append(rows, r)
		locationIDs.Add(c.Location.ID)
	}
	return rows, locationIDs.Size(), nil
}

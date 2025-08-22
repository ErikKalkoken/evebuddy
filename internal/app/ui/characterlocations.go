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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type characterLocationRow struct {
	characterID     int32
	characterName   string
	location        *app.EveLocationShort
	locationDisplay []widget.RichTextSegment
	locationID      int64
	locationName    string
	regionID        int32
	regionName      string
	shipName        string
	shipTypeID      int32
	solarSystemName string
	tags            set.Set[string]
}

type characterLocations struct {
	widget.BaseWidget

	body              fyne.CanvasObject
	columnSorter      *columnSorter
	rows              []characterLocationRow
	rowsFiltered      []characterLocationRow
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortButton        *sortButton
	bottom            *widget.Label
	u                 *baseUI
}

func newCharacterLocations(u *baseUI) *characterLocations {
	headers := []headerDef{
		{label: "Character", width: columnWidthEntity},
		{label: "Location", width: columnWidthLocation},
		{label: "Region", width: columnWidthRegion},
		{label: "Ship", width: 150},
	}
	a := &characterLocations{
		columnSorter: newColumnSorterWithInit(headers, 0, sortAsc),
		rows:         make([]characterLocationRow, 0),
		rowsFiltered: make([]characterLocationRow, 0),
		bottom:       makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	if a.u.isDesktop {
		a.body = makeDataTable(
			headers,
			&a.rowsFiltered,
			func(col int, r characterLocationRow) []widget.RichTextSegment {
				switch col {
				case 0:
					return iwidget.RichTextSegmentsFromText(r.characterName)
				case 1:
					if r.locationID == 0 {
						r.locationDisplay = iwidget.RichTextSegmentsFromText("?")
					}
					return r.locationDisplay
				case 2:
					if r.regionName == "" {
						r.regionName = "?"
					}
					return iwidget.RichTextSegmentsFromText(r.regionName)
				case 3:
					if r.shipName == "" {
						r.shipName = "?"
					}
					return iwidget.RichTextSegmentsFromText(r.shipName)
				}
				return iwidget.RichTextSegmentsFromText("?")
			},
			a.columnSorter,
			a.filterRows, func(_ int, r characterLocationRow) {
				showCharacterLocationWindow(a.u, r)
			})
	} else {
		a.body = a.makeDataList()
	}

	a.selectRegion = kxwidget.NewFilterChipSelectWithSearch("Region", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectSolarSystem = kxwidget.NewFilterChipSelectWithSearch("System", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)

	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		switch arg.section {
		case app.SectionCharacterLocation, app.SectionCharacterOnline, app.SectionCharacterShip:
			a.update()
		}
	})
	return a
}

func (a *characterLocations) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectSolarSystem, a.selectRegion, a.selectTag)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(container.NewHScroll(filter), a.bottom, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *characterLocations) makeDataList() *iwidget.StripedList {
	p := theme.Padding()
	var l *iwidget.StripedList
	l = iwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			character := widget.NewLabel("Template")
			character.Wrapping = fyne.TextWrapWord
			character.SizeName = theme.SizeNameSubHeadingText
			location := iwidget.NewRichTextWithText("Template")
			location.Wrapping = fyne.TextWrapWord
			ship := widget.NewLabel("Template")
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				character,
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
			c[1].(*iwidget.RichText).Set(r.locationDisplay)
			c[2].(*widget.Label).SetText(r.shipName)
			l.SetItemHeight(id, co.(*fyne.Container).MinSize().Height)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		showCharacterLocationWindow(a.u, r)
	}
	return l
}

func (a *characterLocations) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectRegion.Selected; x != "" {
		rows = xslices.Filter(rows, func(r characterLocationRow) bool {
			return r.regionName == x
		})
	}
	if x := a.selectSolarSystem.Selected; x != "" {
		rows = xslices.Filter(rows, func(r characterLocationRow) bool {
			return r.solarSystemName == x
		})
	}
	if x := a.selectTag.Selected; x != "" {
		rows = xslices.Filter(rows, func(r characterLocationRow) bool {
			return r.tags.Contains(x)
		})
	}
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b characterLocationRow) int {
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
	// set data & refresh
	a.selectTag.SetOptions(slices.Sorted(set.Union(xslices.Map(rows, func(r characterLocationRow) set.Set[string] {
		return r.tags
	})...).All()))
	a.selectRegion.SetOptions(xslices.Map(rows, func(r characterLocationRow) string {
		return r.regionName
	}))
	a.selectSolarSystem.SetOptions(xslices.Map(rows, func(r characterLocationRow) string {
		return r.solarSystemName
	}))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *characterLocations) update() {
	rows := make([]characterLocationRow, 0)
	t, i, err := func() (string, widget.Importance, error) {
		cc, err := a.fetchData(a.u.services())
		if err != nil {
			return "", 0, err
		}
		if len(cc) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		rows = cc
		return "", widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh locations UI", "err", err)
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

func (*characterLocations) fetchData(s services) ([]characterLocationRow, error) {
	ctx := context.TODO()
	characters, err := s.cs.ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]characterLocationRow, 0)
	for _, c := range characters {
		if c.EveCharacter == nil {
			continue
		}
		r := characterLocationRow{
			characterID:   c.EveCharacter.ID,
			characterName: c.EveCharacter.Name,
		}
		if c.Location != nil {
			r.locationDisplay = c.Location.DisplayRichText()
			r.locationID = c.Location.ID
			r.locationName = c.Location.DisplayName()
			if c.Location.SolarSystem != nil {
				r.regionID = c.Location.SolarSystem.Constellation.Region.ID
				r.regionName = c.Location.SolarSystem.Constellation.Region.Name
				r.solarSystemName = c.Location.SolarSystem.Name
			}
			r.location = c.Location.ToShort()
		}
		if c.Ship != nil {
			r.shipName = c.Ship.Name
			r.shipTypeID = c.Ship.ID
		}
		tags, err := s.cs.ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		r.tags = tags
		rows = append(rows, r)
	}
	return rows, nil
}

// showCharacterLocationWindow shows the location of a character in a new window.
func showCharacterLocationWindow(u *baseUI, r characterLocationRow) {
	w, ok := u.getOrCreateWindow(fmt.Sprintf("location-%d", r.characterID), "Character Location", r.characterName)
	if !ok {
		w.Show()
		return
	}
	ship := makeLinkLabelWithWrap(r.shipName, func() {
		u.ShowTypeInfoWindowWithCharacter(r.shipTypeID, r.characterID)
	})
	var location fyne.CanvasObject
	if r.location != nil {
		location = makeLocationLabel(r.location, u.ShowLocationInfoWindow)
	} else {
		location = widget.NewLabel("?")
	}
	var region fyne.CanvasObject
	if r.regionID != 0 {
		region = makeLinkLabel(r.regionName, func() {
			u.ShowInfoWindow(app.EveEntityRegion, r.regionID)
		})
	} else {
		region = widget.NewLabel(r.regionName)
	}
	fi := []*widget.FormItem{
		widget.NewFormItem("Owner", makeCharacterActionLabel(
			r.characterID,
			r.characterName,
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Location", location),
		widget.NewFormItem("Region", region),
		widget.NewFormItem("Ship", ship),
	}

	f := widget.NewForm(fi...)
	f.Orientation = widget.Adaptive
	subTitle := fmt.Sprintf("Location of %s", r.characterName)
	setDetailWindow(detailWindowParams{
		content: f,
		minSize: fyne.NewSize(500, 250),
		imageAction: func() {
			u.ShowInfoWindow(app.EveEntityCharacter, r.characterID)
		},
		imageLoader: func() (fyne.Resource, error) {
			return u.eis.CharacterPortrait(r.characterID, 512)
		},
		title:  subTitle,
		window: w,
	})
	w.Show()
}

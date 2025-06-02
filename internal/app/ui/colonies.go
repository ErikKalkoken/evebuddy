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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const (
	colonyStatusExtracting = "Extracting"
	colonyStatusOffline    = "Offline"
)

type colonyRow struct {
	characterID     int32
	due             time.Time
	dueRT           []widget.RichTextSegment
	isExpired       bool
	extracting      set.Set[string]
	extractingText  string
	name            string
	nameRT          []widget.RichTextSegment
	ownerName       string
	planetID        int32
	producing       set.Set[string]
	producingText   string
	regionName      string
	solarSystemName string
	typeName        string
}

type colonies struct {
	widget.BaseWidget

	OnUpdate func(total, expired int)

	body              fyne.CanvasObject
	columnSorter      *columnSorter
	rows              []colonyRow
	rowsFiltered      []colonyRow
	selectStatus      *iwidget.FilterChipSelect
	selectExtracting  *iwidget.FilterChipSelect
	selectOwner       *iwidget.FilterChipSelect
	selectProducing   *iwidget.FilterChipSelect
	selectRegion      *iwidget.FilterChipSelect
	selectSolarSystem *iwidget.FilterChipSelect
	selectPlanetType  *iwidget.FilterChipSelect
	sortButton        *sortButton
	top               *widget.Label
	u                 *baseUI
}

func newColonies(u *baseUI) *colonies {
	headers := []headerDef{
		{Text: "Planet", Width: 150},
		{Text: "Type", Width: 100},
		{Text: "Extracting", Width: 200, NotSortable: true},
		{Text: "Due", Width: 150},
		{Text: "Producing", Width: 200, NotSortable: true},
		{Text: "Region", Width: 150},
		{Text: "Character", Width: columnWidthCharacter},
	}
	a := &colonies{
		columnSorter: newColumnSorterWithInit(headers, 0, sortAsc),
		rows:         make([]colonyRow, 0),
		rowsFiltered: make([]colonyRow, 0),
		top:          makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	makeCell := func(col int, r colonyRow) []widget.RichTextSegment {
		switch col {
		case 0:
			return r.nameRT
		case 1:
			return iwidget.NewRichTextSegmentFromText(r.typeName)
		case 2:
			return iwidget.NewRichTextSegmentFromText(r.extractingText)
		case 3:
			return r.dueRT
		case 4:
			return iwidget.NewRichTextSegmentFromText(r.producingText)
		case 5:
			return iwidget.NewRichTextSegmentFromText(r.regionName)
		case 6:
			return iwidget.NewRichTextSegmentFromText(r.ownerName)
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	if a.u.isDesktop {
		a.body = makeDataTable(headers, &a.rowsFiltered, makeCell, a.columnSorter, a.filterRows, func(_ int, r colonyRow) {
			a.showColony(r)
		})
	} else {
		a.body = makeDataList(headers, &a.rowsFiltered, makeCell, a.showColony)
	}

	a.selectExtracting = iwidget.NewFilterChipSelectWithSearch("Extracted", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectOwner = iwidget.NewFilterChipSelect("Owner", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectProducing = iwidget.NewFilterChipSelectWithSearch("Produced", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectRegion = iwidget.NewFilterChipSelectWithSearch("Region", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectSolarSystem = iwidget.NewFilterChipSelectWithSearch("System", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)

	a.selectStatus = iwidget.NewFilterChipSelect("Status", []string{
		colonyStatusExtracting,
		colonyStatusOffline,
	}, func(string) {
		a.filterRows(-1)
	})

	a.selectPlanetType = iwidget.NewFilterChipSelect("Planet Type", []string{}, func(string) {
		a.filterRows(-1)
	})

	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)
	return a
}

func (a *colonies) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(
		a.selectSolarSystem,
		a.selectPlanetType,
		a.selectExtracting,
		a.selectStatus,
		a.selectProducing,
		a.selectRegion,
		a.selectOwner,
	)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewVBox(
			a.top,
			container.NewHScroll(filter),
		),
		nil,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *colonies) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectExtracting.Selected; x != "" {
		rows = xslices.Filter(rows, func(r colonyRow) bool {
			return r.extracting.Contains(x)
		})
	}
	if x := a.selectOwner.Selected; x != "" {
		rows = xslices.Filter(rows, func(r colonyRow) bool {
			return r.ownerName == x
		})
	}
	if x := a.selectProducing.Selected; x != "" {
		rows = xslices.Filter(rows, func(r colonyRow) bool {
			return r.producing.Contains(x)
		})
	}
	if x := a.selectRegion.Selected; x != "" {
		rows = xslices.Filter(rows, func(r colonyRow) bool {
			return r.regionName == x
		})
	}
	if x := a.selectSolarSystem.Selected; x != "" {
		rows = xslices.Filter(rows, func(r colonyRow) bool {
			return r.solarSystemName == x
		})
	}
	if x := a.selectStatus.Selected; x != "" {
		rows = xslices.Filter(rows, func(r colonyRow) bool {
			switch x {
			case colonyStatusExtracting:
				return !r.isExpired
			case colonyStatusOffline:
				return r.isExpired
			}
			return false
		})
	}
	if x := a.selectPlanetType.Selected; x != "" {
		rows = xslices.Filter(rows, func(r colonyRow) bool {
			return r.typeName == x
		})
	}
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b colonyRow) int {
			var x int
			switch sortCol {
			case 0:
				x = strings.Compare(a.name, b.name)
			case 1:
				x = strings.Compare(a.typeName, b.typeName)
			case 3:
				x = a.due.Compare(b.due)
			case 5:
				x = strings.Compare(a.regionName, b.regionName)
			case 6:
				x = strings.Compare(a.ownerName, b.ownerName)
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	a.selectOwner.SetOptions(xslices.Map(rows, func(r colonyRow) string {
		return r.ownerName
	}))
	a.selectRegion.SetOptions(xslices.Map(rows, func(r colonyRow) string {
		return r.regionName
	}))
	a.selectSolarSystem.SetOptions(xslices.Map(rows, func(r colonyRow) string {
		return r.solarSystemName
	}))
	a.selectPlanetType.SetOptions(xslices.Map(rows, func(r colonyRow) string {
		return r.typeName
	}))
	var extracting, producing set.Set[string]
	for _, r := range rows {
		extracting.AddSeq(r.extracting.All())
		producing.AddSeq(r.producing.All())
	}
	a.selectExtracting.SetOptions(extracting.Slice())
	a.selectProducing.SetOptions(producing.Slice())
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *colonies) update() {
	var s string
	var i widget.Importance
	var total, expired int
	rows := make([]colonyRow, 0)
	planets, err := a.u.cs.ListAllPlanets(context.Background())
	if err != nil {
		slog.Error("Failed to refresh colonies UI", "err", err)
		s = "ERROR"
		i = widget.DangerImportance
	} else {
		total = len(planets)
		for _, p := range planets {
			extracting := p.ExtractedTypeNames()
			producing := p.ProducedSchematicNames()
			r := colonyRow{
				characterID:     p.CharacterID,
				due:             p.ExtractionsExpiryTime(),
				dueRT:           p.DueRichText(),
				extracting:      set.Of(extracting...),
				isExpired:       p.IsExpired(),
				name:            p.EvePlanet.Name,
				nameRT:          p.NameRichText(),
				ownerName:       a.u.scs.CharacterName(p.CharacterID),
				planetID:        p.EvePlanet.ID,
				producing:       set.Of(producing...),
				regionName:      p.EvePlanet.SolarSystem.Constellation.Region.Name,
				solarSystemName: p.EvePlanet.SolarSystem.Name,
				typeName:        p.EvePlanet.TypeDisplay(),
			}
			if len(extracting) == 0 {
				r.extractingText = "-"
			} else {
				r.extractingText = strings.Join(extracting, ", ")
			}
			if len(producing) == 0 {
				r.producingText = "-"
			} else {
				r.producingText = strings.Join(producing, ", ")
			}
			rows = append(rows, r)
			if p.IsExpired() {
				expired++
			}
		}
		s = fmt.Sprintf("%d colonies", total)
		if expired > 0 {
			s += fmt.Sprintf(" • %d expired", expired)
		}
	}
	fyne.Do(func() {
		a.top.Text = s
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRows(-1)
	})
	if a.OnUpdate != nil {
		a.OnUpdate(total, expired)
	}
}

func (a *colonies) showColony(r colonyRow) {
	cp, err := a.u.cs.GetPlanet(context.Background(), r.characterID, r.planetID)
	if err != nil {
		a.u.ShowErrorDialog("Failed to show colony", err, a.u.window)
		return
	}

	fi := []*widget.FormItem{
		widget.NewFormItem("Planet", iwidget.NewTappableRichText(
			func() {
				a.u.ShowEveEntityInfoWindow(cp.EvePlanet.SolarSystem.ToEveEntity())
			},
			cp.NameRichText()...,
		)),
		widget.NewFormItem("Type", kxwidget.NewTappableLabel(cp.EvePlanet.TypeDisplay(), func() {
			a.u.ShowEveEntityInfoWindow(cp.EvePlanet.Type.ToEveEntity())
		})),
		widget.NewFormItem("Region", kxwidget.NewTappableLabel(
			cp.EvePlanet.SolarSystem.Constellation.Region.Name,
			func() {
				a.u.ShowEveEntityInfoWindow(cp.EvePlanet.SolarSystem.Constellation.Region.ToEveEntity())
			})),
		widget.NewFormItem("Installations", widget.NewLabel(fmt.Sprint(len(cp.Pins)))),
		widget.NewFormItem("Character", kxwidget.NewTappableLabel(r.ownerName, func() {
			a.u.ShowInfoWindow(app.EveEntityCharacter, cp.CharacterID)
		})),
	}
	infos := widget.NewForm(fi...)
	infos.Orientation = widget.Adaptive

	extracting := container.NewVBox()
	for pp := range cp.ActiveExtractors() {
		if pp.ExpiryTime.IsEmpty() {
			continue
		}
		expiryTime := pp.ExpiryTime.ValueOrZero()
		icon, _ := pp.ExtractorProductType.Icon()
		product := kxwidget.NewTappableLabel(pp.ExtractorProductType.Name, func() {
			a.u.ShowEveEntityInfoWindow(pp.ExtractorProductType.ToEveEntity())
		})
		row := container.NewHBox(
			iwidget.NewImageFromResource(icon, fyne.NewSquareSize(app.IconUnitSize)),
			product,
			container.NewHBox(widget.NewLabel(expiryTime.Format(app.DateTimeFormat))),
		)
		if expiryTime.Before(time.Now()) {
			l := widget.NewLabel("EXPIRED")
			l.Importance = widget.DangerImportance
			row.Add(l)
		}
		extracting.Add(row)
	}
	if len(extracting.Objects) == 0 {
		extracting.Add(widget.NewLabel("-"))
	}
	producing := container.NewVBox()
	for _, s := range cp.ProducedSchematics() {
		icon, _ := s.Icon()
		producing.Add(container.NewHBox(
			iwidget.NewImageFromResource(icon, fyne.NewSquareSize(app.IconUnitSize)),
			widget.NewLabel(s.Name),
		))
	}
	if len(producing.Objects) == 0 {
		producing.Add(widget.NewLabel("-"))
	}
	processes := widget.NewForm(
		widget.NewFormItem("Extracting", extracting),
		widget.NewFormItem("Producing", producing),
	)
	processes.Orientation = widget.Adaptive

	top := container.NewHBox(infos, layout.NewSpacer())
	if a.u.isDesktop {
		res, _ := cp.EvePlanet.Type.Icon()
		image := iwidget.NewImageFromResource(res, fyne.NewSquareSize(100))
		top.Add(container.NewVBox(container.NewPadded(image)))
	}
	c := container.NewVBox(top, processes)

	subTitle := fmt.Sprintf("%s - %s", cp.EvePlanet.Name, r.ownerName)
	w := a.u.makeDetailWindow("Colony", subTitle, c)
	w.Show()
}

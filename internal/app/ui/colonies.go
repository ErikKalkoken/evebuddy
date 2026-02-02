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
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

const (
	colonyStatusExtracting = "Extracting"
	colonyStatusOffline    = "Offline"
)

type colonyRow struct {
	characterID     int32
	due             time.Time
	extracting      set.Set[string]
	extractingText  string
	name            string
	nameDisplay     []widget.RichTextSegment
	ownerName       string
	planetID        int32
	planetName      string
	planetTypeID    int32
	planetTypeName  string
	producing       set.Set[string]
	producingText   string
	regionName      string
	solarSystemName string
	tags            set.Set[string]
}

func (r colonyRow) isExpired() bool {
	if r.due.IsZero() {
		return false
	}
	return r.due.Before(time.Now())
}

func (r colonyRow) DueRichText() []widget.RichTextSegment {
	if r.isExpired() {
		return iwidget.RichTextSegmentsFromText("OFFLINE", widget.RichTextStyle{ColorName: theme.ColorNameError})
	}
	if r.due.IsZero() {
		return iwidget.RichTextSegmentsFromText("-")
	}
	return iwidget.RichTextSegmentsFromText(r.due.Format(app.DateTimeFormat))
}

type colonies struct {
	widget.BaseWidget

	OnUpdate func(total, expired int)

	body              fyne.CanvasObject
	columnSorter      *iwidget.ColumnSorter
	rows              []colonyRow
	rowsFiltered      []colonyRow
	selectStatus      *kxwidget.FilterChipSelect
	selectExtracting  *kxwidget.FilterChipSelect
	selectOwner       *kxwidget.FilterChipSelect
	selectProducing   *kxwidget.FilterChipSelect
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectPlanetType  *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortButton        *iwidget.SortButton
	top               *widget.Label
	u                 *baseUI
}

const (
	coloniesColPlanet     = 0
	coloniesColType       = 1
	coloniesColExtracting = 2
	coloniesColDue        = 3
	coloniesColProducing  = 4
	coloniesColRegion     = 5
	coloniesColCharacter  = 6
)

func newColonies(u *baseUI) *colonies {
	headers := iwidget.NewDataColumns([]iwidget.DataColumn{{
		Col:   coloniesColPlanet,
		Label: "Planet",
		Width: 150,
	}, {
		Col:   coloniesColType,
		Label: "Type",
		Width: 100,
	}, {
		Col:    coloniesColExtracting,
		Label:  "Extracting",
		Width:  200,
		NoSort: true,
	}, {
		Col:   coloniesColDue,
		Label: "Due",
		Width: columnWidthDateTime,
	}, {
		Col:    coloniesColProducing,
		Label:  "Producing",
		Width:  200,
		NoSort: true,
	}, {
		Col:   coloniesColRegion,
		Label: "Region",
		Width: 150,
	}, {
		Col:   coloniesColCharacter,
		Label: "Character",
		Width: columnWidthEntity,
	}})
	a := &colonies{
		columnSorter: iwidget.NewColumnSorter(headers, coloniesColPlanet, iwidget.SortAsc),
		rows:         make([]colonyRow, 0),
		rowsFiltered: make([]colonyRow, 0),
		top:          makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	makeCell := func(col int, r colonyRow) []widget.RichTextSegment {
		switch col {
		case coloniesColPlanet:
			return r.nameDisplay
		case coloniesColType:
			return iwidget.RichTextSegmentsFromText(r.planetTypeName)
		case coloniesColExtracting:
			return iwidget.RichTextSegmentsFromText(r.extractingText)
		case coloniesColDue:
			return r.DueRichText()
		case coloniesColProducing:
			return iwidget.RichTextSegmentsFromText(r.producingText)
		case coloniesColRegion:
			return iwidget.RichTextSegmentsFromText(r.regionName)
		case coloniesColCharacter:
			return iwidget.RichTextSegmentsFromText(r.ownerName)
		}
		return iwidget.RichTextSegmentsFromText("?")
	}
	if !a.u.isMobile {
		a.body = iwidget.MakeDataTable(headers, &a.rowsFiltered, makeCell, a.columnSorter, a.filterRows, func(_ int, r colonyRow) {
			a.showColonyWindow(r)
		})
	} else {
		a.body = iwidget.MakeDataList(headers, &a.rowsFiltered, makeCell, a.showColonyWindow)
	}

	a.selectExtracting = kxwidget.NewFilterChipSelectWithSearch("Extracted", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectOwner = kxwidget.NewFilterChipSelect("Owner", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectProducing = kxwidget.NewFilterChipSelectWithSearch("Produced", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectRegion = kxwidget.NewFilterChipSelectWithSearch("Region", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectSolarSystem = kxwidget.NewFilterChipSelectWithSearch("System", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectStatus = kxwidget.NewFilterChipSelect("Status", []string{
		colonyStatusExtracting,
		colonyStatusOffline,
	}, func(string) {
		a.filterRows(-1)
	})
	a.selectPlanetType = kxwidget.NewFilterChipSelect("Planet Type", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRows(-1)
	}, a.u.window)

	// Signals
	a.u.refreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.body.Refresh()
		})
		a.setOnUpdate(a.rows)
	})
	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		if arg.section == app.SectionCharacterPlanets {
			a.update()
		}
	})
	a.u.characterAdded.AddListener(func(_ context.Context, _ *app.Character) {
		a.update()
	})
	a.u.characterRemoved.AddListener(func(_ context.Context, _ *app.EntityShort[int32]) {
		a.update()
	})
	a.u.tagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update()
	})
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
		a.selectTag,
	)
	if a.u.isMobile {
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
	extracting := a.selectExtracting.Selected
	owner := a.selectOwner.Selected
	producing := a.selectProducing.Selected
	region := a.selectRegion.Selected
	solarSystem := a.selectSolarSystem.Selected
	status := a.selectStatus.Selected
	planetType := a.selectPlanetType.Selected
	tag := a.selectTag.Selected
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		// filter
		if extracting != "" {
			rows = slices.DeleteFunc(rows, func(r colonyRow) bool {
				return !r.extracting.Contains(extracting)
			})
		}
		if owner != "" {
			rows = slices.DeleteFunc(rows, func(r colonyRow) bool {
				return r.ownerName != owner
			})
		}
		if producing != "" {
			rows = slices.DeleteFunc(rows, func(r colonyRow) bool {
				return !r.producing.Contains(producing)
			})
		}
		if region != "" {
			rows = slices.DeleteFunc(rows, func(r colonyRow) bool {
				return r.regionName != region
			})
		}
		if solarSystem != "" {
			rows = slices.DeleteFunc(rows, func(r colonyRow) bool {
				return r.solarSystemName != solarSystem
			})
		}
		if status != "" {
			rows = slices.DeleteFunc(rows, func(r colonyRow) bool {
				switch status {
				case colonyStatusExtracting:
					return r.isExpired()
				case colonyStatusOffline:
					return !r.isExpired()
				}
				return true
			})
		}
		if planetType != "" {
			rows = slices.DeleteFunc(rows, func(r colonyRow) bool {
				return r.planetTypeName != planetType
			})
		}
		if tag != "" {
			rows = slices.DeleteFunc(rows, func(r colonyRow) bool {
				return !r.tags.Contains(tag)
			})
		}
		// sort
		if doSort {
			slices.SortFunc(rows, func(a, b colonyRow) int {
				var x int
				switch sortCol {
				case coloniesColPlanet:
					x = strings.Compare(a.name, b.name)
				case coloniesColType:
					x = strings.Compare(a.planetTypeName, b.planetTypeName)
				case coloniesColDue:
					x = a.due.Compare(b.due)
				case coloniesColRegion:
					x = strings.Compare(a.regionName, b.regionName)
				case coloniesColCharacter:
					x = xstrings.CompareIgnoreCase(a.ownerName, b.ownerName)
				}
				if dir == iwidget.SortAsc {
					return x
				} else {
					return -1 * x
				}
			})
		}
		// set data & refresh
		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r colonyRow) set.Set[string] {
			return r.tags
		})...).All())
		ownerOptions := xslices.Map(rows, func(r colonyRow) string {
			return r.ownerName
		})
		regionOptions := xslices.Map(rows, func(r colonyRow) string {
			return r.regionName
		})
		solarSystemOptions := xslices.Map(rows, func(r colonyRow) string {
			return r.solarSystemName
		})
		planetTypeOptions := xslices.Map(rows, func(r colonyRow) string {
			return r.planetTypeName
		})
		var extracting2, producing2 set.Set[string]
		for _, r := range rows {
			extracting2.AddSeq(r.extracting.All())
			producing2.AddSeq(r.producing.All())
		}
		extractingOptions := slices.Collect(extracting2.All())
		producingOptions := slices.Collect(producing2.All())

		fyne.Do(func() {
			a.selectTag.SetOptions(tagOptions)
			a.selectOwner.SetOptions(ownerOptions)
			a.selectRegion.SetOptions(regionOptions)
			a.selectSolarSystem.SetOptions(solarSystemOptions)
			a.selectPlanetType.SetOptions(planetTypeOptions)
			a.selectExtracting.SetOptions(extractingOptions)
			a.selectProducing.SetOptions(producingOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	}()
}

func (a *colonies) update() {
	var s string
	var i widget.Importance
	rows, expired, err := a.fetchRows(a.u.services())
	if err != nil {
		slog.Error("Failed to refresh colony UI", "err", err)
		s = "ERROR: " + a.u.humanizeError(err)
		i = widget.DangerImportance
	} else {
		s = fmt.Sprintf("%d colonies", len(rows))
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
	a.setOnUpdate(rows)
}

func (a *colonies) setOnUpdate(rows []colonyRow) {
	var expired int
	for _, r := range rows {
		if r.isExpired() {
			expired++
		}
	}
	if a.OnUpdate != nil {
		a.OnUpdate(len(a.rows), expired)
	}
}

func (a *colonies) fetchRows(s services) ([]colonyRow, int, error) {
	var expired int
	rows := make([]colonyRow, 0)
	ctx := context.Background()

	planets, err := s.cs.ListAllPlanets(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, p := range planets {
		extracting := p.ExtractedTypeNames()
		producing := p.ProducedSchematicNames()
		r := colonyRow{
			characterID:     p.CharacterID,
			due:             p.ExtractionsExpiryTime(),
			extracting:      set.Of(extracting...),
			name:            p.EvePlanet.Name,
			nameDisplay:     p.NameRichText(),
			ownerName:       a.u.scs.CharacterName(p.CharacterID),
			planetID:        p.EvePlanet.ID,
			planetName:      p.EvePlanet.Name,
			producing:       set.Of(producing...),
			regionName:      p.EvePlanet.SolarSystem.Constellation.Region.Name,
			solarSystemName: p.EvePlanet.SolarSystem.Name,
			planetTypeName:  p.EvePlanet.TypeDisplay(),
			planetTypeID:    p.EvePlanet.Type.ID,
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
		tags, err := s.cs.ListTagsForCharacter(ctx, p.CharacterID)
		if err != nil {
			return nil, 0, err
		}
		r.tags = tags
		rows = append(rows, r)
		if r.isExpired() {
			expired++
		}
	}
	return rows, expired, nil
}

// showColonyWindow shows the details of a colony in a window.
func (a *colonies) showColonyWindow(r colonyRow) {
	title := fmt.Sprintf("Colony %s", r.planetName)
	w, ok := a.u.getOrCreateWindow(fmt.Sprintf("colony-%d-%d", r.characterID, r.planetID), title, r.ownerName)
	if !ok {
		w.Show()
		return
	}
	cp, err := a.u.cs.GetPlanet(context.Background(), r.characterID, r.planetID)
	if err != nil {
		a.u.showErrorDialog("Failed to show colony", err, a.u.window)
		return
	}

	fi := []*widget.FormItem{
		widget.NewFormItem("Owner", makeLinkLabel(r.ownerName, func() {
			a.u.ShowInfoWindow(app.EveEntityCharacter, cp.CharacterID)
		})),
		widget.NewFormItem("Planet", widget.NewLabel(cp.EvePlanet.Name)),
		widget.NewFormItem("Type", makeLinkLabel(cp.EvePlanet.TypeDisplay(), func() {
			a.u.ShowEveEntityInfoWindow(cp.EvePlanet.Type.EveEntity())
		})),
		widget.NewFormItem("System", makeLinkLabel(cp.EvePlanet.SolarSystem.Name, func() {
			a.u.ShowEveEntityInfoWindow(cp.EvePlanet.SolarSystem.EveEntity())
		})),
		widget.NewFormItem("Region", makeLinkLabel(
			cp.EvePlanet.SolarSystem.Constellation.Region.Name,
			func() {
				a.u.ShowEveEntityInfoWindow(cp.EvePlanet.SolarSystem.Constellation.Region.EveEntity())
			})),
		widget.NewFormItem("Installations", widget.NewLabel(fmt.Sprint(len(cp.Pins)))),
	}
	infos := widget.NewForm(fi...)
	infos.Orientation = widget.Adaptive

	extracting := container.NewVBox()
	for pp := range cp.ActiveExtractors() {
		if pp.ExpiryTime.IsEmpty() {
			continue
		}
		expiryTime := pp.ExpiryTime.ValueOrZero()
		due := widget.NewLabel("")
		if expiryTime.Before(time.Now()) {
			due.Text = "OFFLINE"
			due.Importance = widget.DangerImportance
		} else {
			due.Text = expiryTime.Format(app.DateTimeFormat)
		}
		icon, _ := pp.ExtractorProductType.Icon()
		product := makeLinkLabel(pp.ExtractorProductType.Name, func() {
			a.u.ShowEveEntityInfoWindow(pp.ExtractorProductType.EveEntity())
		})
		row := container.NewHBox(
			container.NewHBox(
				iwidget.NewImageFromResource(icon, fyne.NewSquareSize(app.IconUnitSize)), product,
			),
			due,
		)
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

	c := container.NewVBox(infos, processes)
	setDetailWindow(detailWindowParams{
		content: c,
		title:   title,
		imageAction: func() {
			a.u.ShowTypeInfoWindow(cp.EvePlanet.Type.ID)
		},
		imageLoader: func(setter func(r fyne.Resource)) {
			r, _ := cp.EvePlanet.Type.Icon()
			setter(r)
		},
		window: w,
	})
	w.Show()
}

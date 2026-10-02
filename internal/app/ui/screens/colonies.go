package screens

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type colonyRow struct {
	characterID     int64
	extracting      set.Set[string]
	extractingText  string
	name            string
	nameDisplay     []widget.RichTextSegment
	ownerName       string
	planet          *app.CharacterPlanet
	planetID        int64
	planetName      string
	planetTypeID    int64
	planetTypeName  string
	producing       set.Set[string]
	producingText   string
	regionName      string
	searchTarget    string
	solarSystemName string
	status          app.ColonyStatus
	tags            set.Set[string]
	titleDisplay    []widget.RichTextSegment
	workEndsAt      optional.Optional[time.Time]
	worksBeyond     bool // still working at the forecast horizon
}

// colonyBeyondHorizonText is shown for colonies which work beyond the forecast horizon.
var colonyBeyondHorizonText = fmt.Sprintf("> %d days", int(app.ColonyForecastHorizon.Hours()/24))

// setForecast updates the row with a new forecast for its colony.
func (r *colonyRow) setForecast(f *app.ColonyForecast) {
	r.status = f.Status
	r.workEndsAt = f.WorkEndsAt
	r.worksBeyond = f.WorksBeyondHorizon
}

// compareWorkEnds orders colonies by when they stop working:
// not working first, then by work end, then working beyond the horizon.
func (r colonyRow) compareWorkEnds(other colonyRow) int {
	rank := func(x colonyRow) int {
		switch {
		case x.worksBeyond:
			return 2
		case x.workEndsAt.IsEmpty():
			return 0
		}
		return 1
	}
	if c := cmp.Compare(rank(r), rank(other)); c != 0 {
		return c
	}
	return optional.CompareFunc(r.workEndsAt, other.workEndsAt, func(x, y time.Time) int {
		return x.Compare(y)
	})
}

func (r colonyRow) needsAttention() bool {
	return !r.status.IsWorking()
}

func (r colonyRow) statusDisplay() []widget.RichTextSegment {
	return xwidget.RichTextSegmentsFromText(r.status.Display(), widget.RichTextStyle{
		ColorName: r.status.Color(),
		Inline:    true,
	})
}

func (r colonyRow) workEndsDisplay() string {
	if r.worksBeyond {
		return colonyBeyondHorizonText
	}
	return r.workEndsAt.StringFunc("-", func(v time.Time) string {
		return v.Format(app.DateTimeFormat)
	})
}

type Colonies struct {
	widget.BaseWidget

	OnUpdate func(total, expired int)

	body              fyne.CanvasObject
	columnSorter      *xwidget.ColumnSorter[colonyRow]
	filterRun         latestRun
	footer            *widget.Label
	forecastRun       latestRun
	rows              []colonyRow
	rowsGen           int // incremented when Update replaces rows
	rowsRun           latestRun
	rowsFiltered      []colonyRow
	searchEntry       *xwidget.SearchEntry
	selectExtracting  *kxwidget.FilterChipSelect
	selectOwner       *kxwidget.FilterChipSelect
	selectPlanetType  *kxwidget.FilterChipSelect
	selectProducing   *kxwidget.FilterChipSelect
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectStatus      *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	showHelp          *xwidget.IconButton
	sortChip          *kxwidget.SortChip
	u                 baseUI
}

// colonyStatusesHelpText explains the colony statuses.
const colonyStatusesHelpText = `• Extracting: At least one extractor is running.
• Producing: No extractor is running, but at least one factory is producing.
• Idle: Nothing is extracting or producing.
• Needs Attention: An extractor has expired or stopped, or a storage is full.
• Not Setup: A facility is not configured, e.g. a factory without a schematic or with an input or output not routed.`

// colonyEstimateHelpText returns a note explaining that values are estimates.
func colonyEstimateHelpText(values string) string {
	return fmt.Sprintf("NOTE: %s are estimates. They are calculated from the last time the colony was updated in game, and can differ from the actual state.", values)
}

// coloniesHelpText returns the help text for the colonies screen.
func coloniesHelpText(isMobile bool) string {
	var layout, notWorking string
	if isMobile {
		layout = `Each colony shows:
• Top: The planet the colony is on.
• Extractor icon: The resources the extractors are set to extract, followed by the colony's status and the time until work ends.
• Factory icon: The products the factories are set to produce.
• Person icon: The character who owns the colony.`
	} else {
		layout = `Planet: The planet the colony is on.

Extracting: The resources the extractors are set to extract.

Producing: The products the factories are set to produce.

Character: The character who owns the colony.`
		notWorking = ` and "-" when it is not working`
	}
	return fmt.Sprintf(`%s

Status: The estimated current status of the colony:
%s

Work ends: When the colony is estimated to stop working, e.g. when the last extractor expires or factories run out of inputs. Shows "%s" when the colony keeps working beyond that%s.

%s`, layout, colonyStatusesHelpText, colonyBeyondHorizonText, notWorking, colonyEstimateHelpText("Status and work end"))
}

func NewColonies(u baseUI) *Colonies {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[colonyRow]{{
		Label: "Planet",
		Width: 200,
		Sort: func(a, b colonyRow) int {
			return strings.Compare(a.name, b.name)
		},
		Create: func() fyne.CanvasObject {
			icon := xwidget.NewImageFromResource(
				icons.BlankSvg,
				fyne.NewSquareSize(ui.IconUnitSize),
			)
			name := xwidget.NewRichText()
			name.Truncation = fyne.TextTruncateClip
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		Update: func(r colonyRow, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			border[0].(*xwidget.RichText).Set(r.nameDisplay)
			x := border[1].(*canvas.Image)
			u.EVEImage().InventoryTypeIconAsync(r.planetTypeID, ui.IconPixelSize, func(r fyne.Resource) {
				x.Resource = r
				x.Refresh()
			})
		},
	}, {
		Label: "Status (est.)",
		Width: 150,
		Sort: func(a, b colonyRow) int {
			return cmp.Compare(a.status, b.status)
		},
		Update: func(r colonyRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.statusDisplay())
		},
	}, {
		Label: "Work ends (est.)",
		Width: ui.ColumnWidthDateTime,
		Sort: func(a, b colonyRow) int {
			return a.compareWorkEnds(b)
		},
		Update: func(r colonyRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.workEndsDisplay())
		},
	}, {
		Label: "Extracting",
		Width: 200,
		Update: func(r colonyRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.extractingText)
		},
	}, {
		Label: "Producing",
		Width: 200,
		Update: func(r colonyRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.producingText)
		},
	}, {
		Label: "Character",
		Width: ui.ColumnWidthEntity,
		Sort: func(a, b colonyRow) int {
			return xstrings.CompareIgnoreCase(a.ownerName, b.ownerName)
		},
		Update: func(r colonyRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.ownerName)
		},
	}})
	a := &Colonies{
		footer:       ui.NewLabelWithTruncation(""),
		columnSorter: xwidget.NewColumnSorter(columns, "Work ends (est.)", xwidget.SortAsc),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	if a.u.IsMobile() {
		a.body = a.makeDataList()
	} else {
		a.body = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync, func(_ int, r colonyRow) {
				showColonyDetailsWindow(a.u, r)
			})
	}

	a.selectExtracting = kxwidget.NewFilterChipSelectWithSearch("Extracted", []string{}, func(string) {
		a.filterRowsAsync("")
	}, a.u.MainWindow())
	a.selectOwner = kxwidget.NewFilterChipSelect("Owner", []string{}, func(string) {
		a.filterRowsAsync("")
	})
	a.selectProducing = kxwidget.NewFilterChipSelectWithSearch("Produced", []string{}, func(string) {
		a.filterRowsAsync("")
	}, a.u.MainWindow())
	a.selectRegion = kxwidget.NewFilterChipSelect("Region", []string{}, func(string) {
		a.filterRowsAsync("")
	})
	a.selectSolarSystem = kxwidget.NewFilterChipSelectWithSearch("System", []string{}, func(string) {
		a.filterRowsAsync("")
	}, a.u.MainWindow())
	a.selectStatus = kxwidget.NewFilterChipSelect("Status", []string{}, func(string) {
		a.filterRowsAsync("")
	})
	a.selectPlanetType = kxwidget.NewFilterChipSelect("Planet Type", []string{}, func(string) {
		a.filterRowsAsync("")
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync("")
	})
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	})

	a.searchEntry = xwidget.NewSearchEntry("Search systems & output", func(_ string) {
		a.filterRowsAsync("")
	})

	a.showHelp = xwidget.NewIconButton(theme.QuestionIcon(), func() {
		showHelpPopUp(coloniesHelpText(a.u.IsMobile()), a.u.IsMobile(), a.showHelp)
	})
	a.showHelp.SetToolTip("Show explanation for columns")

	// Signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.Update(ctx)
	})

	a.u.Signals().RefreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.refreshForecasts()
		})
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if arg.Section == app.SectionCharacterPlanets {
			a.Update(ctx)
		}
	})
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.Update(ctx)
	})
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.Update(ctx)
	})
	a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) {
		a.Update(ctx)
	})

	return a
}

func (a *Colonies) CreateRenderer() fyne.WidgetRenderer {
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
	if a.u.IsMobile() {
		filter.Add(a.sortChip)
	}
	var top *fyne.Container
	if a.u.IsMobile() {
		top = container.NewVBox(
			a.searchEntry,
			container.NewHScroll(filter),
		)
	} else {
		top = container.NewBorder(nil, nil, filter, nil, a.searchEntry)
	}
	c := container.NewBorder(
		top,
		container.NewBorder(nil, nil, nil, a.showHelp, a.footer),
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *Colonies) makeDataList() *xwidget.StripedList {
	l := xwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newColonyListItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			co.(*colonyListItem).set(a.rowsFiltered[id])
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		showColonyDetailsWindow(a.u, a.rowsFiltered[id])
	}
	return l
}

type colonyListItem struct {
	widget.BaseWidget

	character  *widget.Label
	extracting *widget.Label
	producing  *widget.Label
	status     *xwidget.RichText
	title      *xwidget.RichText
}

func newColonyListItem() *colonyListItem {
	character := widget.NewLabel("Template")
	character.Truncation = fyne.TextTruncateClip
	extracting := widget.NewLabel("Template")
	extracting.Truncation = fyne.TextTruncateClip
	producing := widget.NewLabel("Template")
	producing.Truncation = fyne.TextTruncateClip
	status := xwidget.NewRichText()
	w := &colonyListItem{
		character:  character,
		extracting: extracting,
		producing:  producing,
		status:     status,
		title:      xwidget.NewRichText(),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *colonyListItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	iconSize := fyne.NewSquareSize(theme.Size(theme.SizeNameInlineIcon))
	c := container.New(layout.NewCustomPaddedVBoxLayout(-p),
		w.title,
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				xwidget.NewSpacer(fyne.NewSize(p/2, 1)),
				newColonyPinIcon(pinTypeExtractor.icon(), iconSize),
			),
			w.status,
			w.extracting,
		),
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				xwidget.NewSpacer(fyne.NewSize(p/2, 1)),
				newColonyPinIcon(pinTypeBasicProcessor.icon(), iconSize),
			),
			nil,
			w.producing,
		),
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				xwidget.NewSpacer(fyne.NewSize(p/2, 1)),
				widget.NewIcon(theme.AccountIcon()),
			),
			nil,
			w.character,
		),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *colonyListItem) set(r colonyRow) {
	w.character.SetText(r.ownerName)
	w.extracting.SetText(r.extractingText)
	w.title.Set(r.titleDisplay)
	w.producing.SetText(r.producingText)
	status := r.statusDisplay()
	if v, ok := r.workEndsAt.Value(); ok {
		status = slices.Concat(status, xwidget.RichTextSegmentsFromText(" • "+ihumanize.Duration(time.Until(v))))
	} else if r.worksBeyond {
		status = slices.Concat(status, xwidget.RichTextSegmentsFromText(" • "+colonyBeyondHorizonText))
	}
	w.status.Set(status)
}

func (a *Colonies) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.start()
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	extracting := a.selectExtracting.Selected
	owner := a.selectOwner.Selected
	producing := a.selectProducing.Selected
	region := a.selectRegion.Selected
	solarSystem := a.selectSolarSystem.Selected
	status := a.selectStatus.Selected
	planetType := a.selectPlanetType.Selected
	tag := a.selectTag.Selected
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
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
				return r.status.Display() != status
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
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r colonyRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

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
		statusOptions := xslices.Map(rows, func(r colonyRow) string {
			return r.status.Display()
		})
		var extracting2, producing2 set.Set[string]
		for _, r := range rows {
			extracting2.AddSeq(r.extracting.All())
			producing2.AddSeq(r.producing.All())
		}
		extractingOptions := slices.Collect(extracting2.All())
		producingOptions := slices.Collect(producing2.All())

		footer := fmt.Sprintf("Showing %d / %d colonies", len(rows), totalRows)
		var attention int
		for _, r := range rows {
			if r.needsAttention() {
				attention++
			}
		}
		if attention > 0 {
			footer += fmt.Sprintf(" • %d not working", attention)
		}

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectTag.SetOptions(tagOptions)
			a.selectOwner.SetOptions(ownerOptions)
			a.selectRegion.SetOptions(regionOptions)
			a.selectSolarSystem.SetOptions(solarSystemOptions)
			a.selectPlanetType.SetOptions(planetTypeOptions)
			a.selectStatus.SetOptions(statusOptions)
			a.selectExtracting.SetOptions(extractingOptions)
			a.selectProducing.SetOptions(producingOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	})
}

func (a *Colonies) Update(ctx context.Context) {
	isLatest := a.rowsRun.start()
	rows, err := a.fetchRows(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh colony UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + a.u.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
	}
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.rows = rows
		a.rowsGen++
		a.filterRowsAsync("")
		a.setOnUpdate()
	})
}

// refreshForecasts recalculates the forecasts for all colonies.
// A refresh never discards rows from Update, but is discarded when Update replaced them.
func (a *Colonies) refreshForecasts() {
	isLatest := a.forecastRun.start()
	gen := a.rowsGen
	rows := slices.Clone(a.rows)
	runAsync(func() {
		now := time.Now()
		for i := range rows {
			rows[i].setForecast(a.u.Character().ForecastPlanet(rows[i].planet, now))
		}
		fyne.Do(func() {
			if !isLatest() || a.rowsGen != gen {
				return
			}
			a.rows = rows
			a.filterRowsAsync("")
			a.setOnUpdate()
		})
	})
}

func (a *Colonies) setOnUpdate() {
	var attention int
	for _, r := range a.rows {
		if r.needsAttention() {
			attention++
		}
	}
	if a.OnUpdate != nil {
		a.OnUpdate(len(a.rows), attention)
	}
}

func (a *Colonies) fetchRows(ctx context.Context) ([]colonyRow, error) {
	planets, err := a.u.Character().ListAllPlanets(ctx)
	if err != nil {
		return nil, err
	}
	characters, err := a.u.Character().CharacterNames(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	var rows []colonyRow
	for _, p := range planets {
		extracting := set.Collect(xiter.MapSlice(p.ExtractedTypes(), func(x *app.EveType) string {
			return x.Name
		}))
		producing := set.Collect(xiter.MapSlice(p.ProducedSchematics(), func(x *app.EveSchematic) string {
			return x.Name
		}))
		titleDisplay := xwidget.ModifyRichTextStyle(p.NameRichText(), func(x *widget.RichTextStyle) {
			x.SizeName = theme.SizeNameSubHeadingText
		})
		name := p.EvePlanet.Name
		searchTargets := slices.Collect(xiter.Map(set.Union(set.Of(name), extracting, producing).All(), strings.ToLower))
		r := colonyRow{
			characterID:     p.CharacterID,
			extracting:      extracting,
			name:            name,
			nameDisplay:     p.NameRichText(),
			ownerName:       characters[p.CharacterID],
			planet:          p,
			planetID:        p.EvePlanet.ID,
			planetName:      p.EvePlanet.Name,
			producing:       producing,
			regionName:      p.EvePlanet.SolarSystem.Constellation.Region.Name,
			solarSystemName: p.EvePlanet.SolarSystem.Name,
			planetTypeName:  p.EvePlanet.TypeDisplay(),
			planetTypeID:    p.EvePlanet.Type.ID,
			titleDisplay:    titleDisplay,
			searchTarget:    strings.Join(searchTargets, "~"),
		}
		r.setForecast(a.u.Character().ForecastPlanet(p, now))
		if extracting.Size() == 0 {
			r.extractingText = "-"
		} else {
			r.extractingText = strings.Join(slices.Sorted(extracting.All()), ", ")
		}
		if producing.Size() == 0 {
			r.producingText = "-"
		} else {
			r.producingText = strings.Join(slices.Sorted(producing.All()), ", ")
		}
		tags, err := a.u.Character().ListTagsForCharacter(ctx, p.CharacterID)
		if err != nil {
			return nil, err
		}
		r.tags = tags
		rows = append(rows, r)
	}
	return rows, nil
}

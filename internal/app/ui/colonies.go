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
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/uiservices"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	colonyStatusExtracting = "Active"
	colonyStatusAllIdle    = "Idle"
	colonyStatusSomeIdle   = "Partially idle"
)

type colonyRow struct {
	characterID       int64
	extracting        set.Set[string]
	extractingText    string
	extractorExpiries []time.Time
	extractorExpiry   optional.Optional[time.Time]
	name              string
	nameDisplay       []widget.RichTextSegment
	ownerName         string
	planetID          int64
	planetName        string
	planetTypeID      int64
	planetTypeName    string
	producing         set.Set[string]
	producingText     string
	regionName        string
	searchTarget      string
	solarSystemName   string
	tags              set.Set[string]
	titleDisplay      []widget.RichTextSegment
}

func (r colonyRow) remaining() time.Duration {
	if v, ok := r.extractorExpiry.Value(); ok {
		return time.Until(v)
	}
	return 0
}

func (r colonyRow) isExpired() bool {
	return r.remaining() <= 0
}

func (r colonyRow) statusDisplay() []widget.RichTextSegment {
	return colonyStatusDisplay(r.extractorExpiries)
}

func colonyStatusDisplay(extractorExpiries []time.Time) []widget.RichTextSegment {
	if len(extractorExpiries) == 0 {
		return xwidget.RichTextSegmentsFromText("-")
	}
	var expired int
	for _, v := range extractorExpiries {
		if v.Before(time.Now()) {
			expired++
		}
	}
	if expired == len(extractorExpiries) {
		return xwidget.RichTextSegmentsFromText(colonyStatusAllIdle, widget.RichTextStyle{
			ColorName: theme.ColorNameError,
		})
	}
	if expired > 0 {
		return xwidget.RichTextSegmentsFromText(colonyStatusSomeIdle, widget.RichTextStyle{
			ColorName: theme.ColorNameWarning,
		})
	}
	earliest := slices.MinFunc(extractorExpiries, func(a, b time.Time) int {
		return a.Compare(b)
	})
	return xwidget.RichTextSegmentsFromText(ihumanize.Duration(time.Until(earliest)), widget.RichTextStyle{
		ColorName: theme.ColorNameForeground,
	})
}

type Colonies struct {
	widget.BaseWidget

	body              fyne.CanvasObject
	columnSorter      *xwidget.ColumnSorter[colonyRow]
	footer            *widget.Label
	onUpdate          func(total, expired int)
	rows              []colonyRow
	rowsFiltered      []colonyRow
	search            *widget.Entry
	selectExtracting  *kxwidget.FilterChipSelect
	selectOwner       *kxwidget.FilterChipSelect
	selectPlanetType  *kxwidget.FilterChipSelect
	selectProducing   *kxwidget.FilterChipSelect
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectStatus      *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortButton        *xwidget.SortButton
	u         uiservices.UIServices
}

const (
	coloniesColPlanet = iota + 1
	coloniesColStatus
	coloniesColExtracting
	coloniesColEndDate
	coloniesColProducing
	coloniesColRegion
	coloniesColCharacter
)

func NewColonies(u         uiservices.UIServices) *Colonies {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[colonyRow]{{
		ID:    coloniesColPlanet,
		Label: "Planet",
		Width: 200,
		Sort: func(a, b colonyRow) int {
			return strings.Compare(a.name, b.name)
		},
		Create: func() fyne.CanvasObject {
			icon := xwidget.NewImageFromResource(
				icons.BlankSvg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			name := xwidget.NewRichText()
			name.Truncation = fyne.TextTruncateClip
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		Update: func(r colonyRow, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			border[0].(*xwidget.RichText).Set(r.nameDisplay)
			x := border[1].(*canvas.Image)
			u.EVEImage().InventoryTypeIconAsync(r.planetTypeID, app.IconPixelSize, func(r fyne.Resource) {
				x.Resource = r
				x.Refresh()
			})
		},
	}, {
		ID:    coloniesColStatus,
		Label: "Status",
		Width: 100,
		Sort: func(a, b colonyRow) int {
			return cmp.Compare(a.remaining(), b.remaining())
		},
		Update: func(r colonyRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.statusDisplay())
		},
	}, {
		ID:    coloniesColExtracting,
		Label: "Extracting",
		Width: 200,
		Update: func(r colonyRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.extractingText)
		},
	}, {
		ID:    coloniesColEndDate,
		Label: "End data",
		Width: columnWidthDateTime,
		Sort: func(a, b colonyRow) int {
			return optional.CompareFunc(a.extractorExpiry, b.extractorExpiry, func(x, y time.Time) int {
				return x.Compare(y)
			})
		},
		Update: func(r colonyRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.extractorExpiry.StringFunc("-", func(v time.Time) string {
				return v.Format(app.DateTimeFormat)
			}))
		},
	}, {
		ID:    coloniesColProducing,
		Label: "Producing",
		Width: 200,
		Update: func(r colonyRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.producingText)
		},
	}, {
		ID:    coloniesColCharacter,
		Label: "Character",
		Width: columnWidthEntity,
		Sort: func(a, b colonyRow) int {
			return xstrings.CompareIgnoreCase(a.ownerName, b.ownerName)
		},
		Update: func(r colonyRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.ownerName)
		},
	}})
	a := &Colonies{
		footer:       newLabelWithTruncation(),
		columnSorter: xwidget.NewColumnSorter(columns, coloniesColEndDate, xwidget.SortAsc),
		search:       widget.NewEntry(),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	if app.IsMobile() {
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
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectOwner = kxwidget.NewFilterChipSelect("Owner", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectProducing = kxwidget.NewFilterChipSelectWithSearch("Produced", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectRegion = kxwidget.NewFilterChipSelect("Region", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectSolarSystem = kxwidget.NewFilterChipSelectWithSearch("System", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectStatus = kxwidget.NewFilterChipSelect("Status", []string{
		colonyStatusExtracting,
		colonyStatusAllIdle,
	}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectPlanetType = kxwidget.NewFilterChipSelect("Planet Type", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRowsAsync(-1)
	})
	a.search.OnChanged = func(s string) {
		a.filterRowsAsync(-1)
	}
	a.search.PlaceHolder = "Search systems & output"

	// Signals
	a.u.Signals().RefreshTickerExpired.AddListener(func(ctx context.Context, _ struct{}) {
		fyne.Do(func() {
			a.body.Refresh()
			a.setOnUpdate()
		})
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if arg.Section == app.SectionCharacterPlanets {
			a.update(ctx)
		}
	})
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update(ctx)
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
	if app.IsMobile() {
		filter.Add(a.sortButton)
	}
	var top *fyne.Container
	if app.IsMobile() {
		top = container.NewVBox(
			a.search,
			container.NewHScroll(filter),
		)
	} else {
		top = container.NewBorder(nil, nil, filter, nil, a.search)
	}
	c := container.NewBorder(
		top,
		a.footer,
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
	c := container.New(layout.NewCustomPaddedVBoxLayout(-p),
		w.title,
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				newSpacer(fyne.NewSize(p/2, 1)),
				widget.NewIcon(eveicon.FromName(eveicon.PIExtractor)),
			),
			w.status,
			w.extracting,
		),
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				newSpacer(fyne.NewSize(p/2, 1)),
				widget.NewIcon(eveicon.FromName(eveicon.PIProcessor)),
			),
			nil,
			w.producing,
		),
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				newSpacer(fyne.NewSize(p/2, 1)),
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
	w.status.Set(r.statusDisplay())
}

func (a *Colonies) filterRowsAsync(sortCol int) {
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
	search := strings.ToLower(a.search.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
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
				case colonyStatusAllIdle:
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
		var extracting2, producing2 set.Set[string]
		for _, r := range rows {
			extracting2.AddSeq(r.extracting.All())
			producing2.AddSeq(r.producing.All())
		}
		extractingOptions := slices.Collect(extracting2.All())
		producingOptions := slices.Collect(producing2.All())

		footer := fmt.Sprintf("Showing %d / %d colonies", len(rows), totalRows)
		var expired int
		for _, r := range rows {
			if r.isExpired() {
				expired++
			}
		}
		if expired > 0 {
			footer += fmt.Sprintf(" • %d expired", expired)
		}

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
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

func (a *Colonies) update(ctx context.Context) {
	rows, err := a.fetchRows(ctx)
	if err != nil {
		slog.Error("Failed to refresh colony UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + app.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync(-1)
		a.setOnUpdate()
	})
}

func (a *Colonies) setOnUpdate() {
	var expired int
	for _, r := range a.rows {
		if r.isExpired() {
			expired++
		}
	}
	if a.onUpdate != nil {
		a.onUpdate(len(a.rows), expired)
	}
}

func (a *Colonies) fetchRows(ctx context.Context) ([]colonyRow, error) {
	planets, err := a.u.Character().ListAllPlanets(ctx)
	if err != nil {
		return nil, err
	}

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
			characterID:       p.CharacterID,
			extractorExpiries: p.ExtractionsExpiryTimes(),
			extractorExpiry:   p.ExtractionsEarliestExpiry(),
			extracting:        extracting,
			name:              name,
			nameDisplay:       p.NameRichText(),
			ownerName:         a.u.StatusCache().CharacterName(p.CharacterID),
			planetID:          p.EvePlanet.ID,
			planetName:        p.EvePlanet.Name,
			producing:         producing,
			regionName:        p.EvePlanet.SolarSystem.Constellation.Region.Name,
			solarSystemName:   p.EvePlanet.SolarSystem.Name,
			planetTypeName:    p.EvePlanet.TypeDisplay(),
			planetTypeID:      p.EvePlanet.Type.ID,
			titleDisplay:      titleDisplay,
			searchTarget:      strings.Join(searchTargets, "~"),
		}
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

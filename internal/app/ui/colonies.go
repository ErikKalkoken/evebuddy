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
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
)

// TODO: Add custom mobile view for list of colonies

const (
	colonyStatusExtracting = "Extracting"
	colonyStatusOffline    = "Offline"
)

const (
	colonyIdle = "Idle"
)

type colonyRow struct {
	characterID     int64
	extractorExpiry optional.Optional[time.Time]
	extracting      set.Set[string]
	extractingText  string
	name            string
	nameDisplay     []widget.RichTextSegment
	ownerName       string
	planetID        int64
	planetName      string
	planetTypeID    int64
	planetTypeName  string
	producing       set.Set[string]
	producingText   string
	regionName      string
	solarSystemName string
	tags            set.Set[string]
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
	return colonyStatusDisplay(r.extractorExpiry)
}

func colonyStatusDisplay(extractorExpiry optional.Optional[time.Time]) []widget.RichTextSegment {
	v, ok := extractorExpiry.Value()
	if !ok {
		return iwidget.RichTextSegmentsFromText("-")
	}
	if v.Before(time.Now()) {
		return iwidget.RichTextSegmentsFromText(colonyIdle, widget.RichTextStyle{ColorName: theme.ColorNameError})
	}
	return iwidget.RichTextSegmentsFromText(ihumanize.Duration(time.Until(v)), widget.RichTextStyle{
		ColorName: theme.ColorNameForeground,
	})
}

type colonies struct {
	widget.BaseWidget

	OnUpdate func(total, expired int)

	body              fyne.CanvasObject
	footer            *widget.Label
	columnSorter      *iwidget.ColumnSorter[colonyRow]
	rows              []colonyRow
	rowsFiltered      []colonyRow
	selectExtracting  *kxwidget.FilterChipSelect
	selectOwner       *kxwidget.FilterChipSelect
	selectPlanetType  *kxwidget.FilterChipSelect
	selectProducing   *kxwidget.FilterChipSelect
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectStatus      *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortButton        *iwidget.SortButton
	u                 *baseUI
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

func newColonies(u *baseUI) *colonies {
	columns := iwidget.NewDataColumns([]iwidget.DataColumn[colonyRow]{{
		ID:    coloniesColPlanet,
		Label: "Planet",
		Width: 200,
		Sort: func(a, b colonyRow) int {
			return strings.Compare(a.name, b.name)
		},
		Create: func() fyne.CanvasObject {
			icon := iwidget.NewImageFromResource(
				icons.BlankSvg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			name := iwidget.NewRichText()
			name.Truncation = fyne.TextTruncateClip
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		Update: func(r colonyRow, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			border[0].(*iwidget.RichText).Set(r.nameDisplay)
			x := border[1].(*canvas.Image)
			u.eis.InventoryTypeIconAsync(r.planetTypeID, app.IconPixelSize, func(r fyne.Resource) {
				x.Resource = r
				x.Refresh()
			})
		},
	},
		{
			ID:    coloniesColStatus,
			Label: "Status",
			Width: 100,
			Sort: func(a, b colonyRow) int {
				return cmp.Compare(a.remaining(), b.remaining())
			},
			Update: func(r colonyRow, co fyne.CanvasObject) {
				co.(*iwidget.RichText).Set(r.statusDisplay())
			},
		}, {
			ID:    coloniesColExtracting,
			Label: "Extracting",
			Width: 200,
			Update: func(r colonyRow, co fyne.CanvasObject) {
				co.(*iwidget.RichText).SetWithText(r.extractingText)
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
				co.(*iwidget.RichText).SetWithText(r.extractorExpiry.StringFunc("-", func(v time.Time) string {
					return v.Format(app.DateTimeFormat)
				}))
			},
		}, {
			ID:    coloniesColProducing,
			Label: "Producing",
			Width: 200,
			Update: func(r colonyRow, co fyne.CanvasObject) {
				co.(*iwidget.RichText).SetWithText(r.producingText)
			},
		}, {
			ID:    coloniesColCharacter,
			Label: "Character",
			Width: columnWidthEntity,
			Sort: func(a, b colonyRow) int {
				return xstrings.CompareIgnoreCase(a.ownerName, b.ownerName)
			},
			Update: func(r colonyRow, co fyne.CanvasObject) {
				co.(*iwidget.RichText).SetWithText(r.ownerName)
			},
		}})
	a := &colonies{
		footer:       newLabelWithTruncation(),
		columnSorter: iwidget.NewColumnSorter(columns, coloniesColEndDate, iwidget.SortAsc),
		rows:         make([]colonyRow, 0),
		rowsFiltered: make([]colonyRow, 0),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	if !a.u.isMobile {
		a.body = iwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := iwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync, func(_ int, r colonyRow) {
				a.showColonyDetailsWindow(r)
			})
	} else {
		a.body = iwidget.MakeDataList(
			columns,
			&a.rowsFiltered,
			func(col int, r colonyRow) []widget.RichTextSegment {
				switch col {
				case coloniesColPlanet:
					return r.nameDisplay
				case coloniesColStatus:
					return iwidget.RichTextSegmentsFromText(r.planetTypeName)
				case coloniesColExtracting:
					return iwidget.RichTextSegmentsFromText(r.extractingText)
				case coloniesColEndDate:
					return r.statusDisplay()
				case coloniesColProducing:
					return iwidget.RichTextSegmentsFromText(r.producingText)
				case coloniesColRegion:
					return iwidget.RichTextSegmentsFromText(r.regionName)
				case coloniesColCharacter:
					return iwidget.RichTextSegmentsFromText(r.ownerName)
				}
				return iwidget.RichTextSegmentsFromText("?")
			},
			a.showColonyDetailsWindow,
		)
	}

	a.selectExtracting = kxwidget.NewFilterChipSelectWithSearch("Extracted", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.window)
	a.selectOwner = kxwidget.NewFilterChipSelect("Owner", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectProducing = kxwidget.NewFilterChipSelectWithSearch("Produced", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.window)
	a.selectRegion = kxwidget.NewFilterChipSelect("Region", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectSolarSystem = kxwidget.NewFilterChipSelectWithSearch("System", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.window)
	a.selectStatus = kxwidget.NewFilterChipSelect("Status", []string{
		colonyStatusExtracting,
		colonyStatusOffline,
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
	}, a.u.window)

	// Signals
	a.u.refreshTickerExpired.AddListener(func(ctx context.Context, _ struct{}) {
		fyne.Do(func() {
			a.body.Refresh()
			a.setOnUpdate()
		})
	})
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		if arg.section == app.SectionCharacterPlanets {
			a.update(ctx)
		}
	})
	a.u.characterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.characterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	a.u.tagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update(ctx)
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
		container.NewHScroll(filter),
		a.footer,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *colonies) filterRowsAsync(sortCol int) {
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
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
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

func (a *colonies) update(ctx context.Context) {
	rows, err := a.fetchRows(ctx)
	if err != nil {
		slog.Error("Failed to refresh colony UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + a.u.humanizeError(err)
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

func (a *colonies) setOnUpdate() {
	var expired int
	for _, r := range a.rows {
		if r.isExpired() {
			expired++
		}
	}
	if a.OnUpdate != nil {
		a.OnUpdate(len(a.rows), expired)
	}
}

func (a *colonies) fetchRows(ctx context.Context) ([]colonyRow, error) {
	planets, err := a.u.cs.ListAllPlanets(ctx)
	if err != nil {
		return nil, err
	}

	var rows []colonyRow
	for _, p := range planets {
		extracting := p.ExtractedTypeNames()
		producing := p.ProducedSchematicNames()
		r := colonyRow{
			characterID:     p.CharacterID,
			extractorExpiry: p.ExtractionsExpiryTime(),
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
		tags, err := a.u.cs.ListTagsForCharacter(ctx, p.CharacterID)
		if err != nil {
			return nil, err
		}
		r.tags = tags
		rows = append(rows, r)
	}
	return rows, nil
}

// showColonyDetailsWindow shows the details of a colony in a window.
func (a *colonies) showColonyDetailsWindow(r colonyRow) {
	title := fmt.Sprintf("Colony %s", r.planetName)
	key := fmt.Sprintf("colony-%d-%d", r.characterID, r.planetID)
	w, ok, onClosed := a.u.getOrCreateWindowWithOnClosed(key, title, r.ownerName)
	if !ok {
		w.Show()
		return
	}

	b := newColonyDetails(a.u, r.characterID, r.planetID)
	err := b.update(context.Background())
	if err != nil {
		slog.Error("Failed to show colony details", "characterID", r.characterID, "planetID", r.planetID, "error", err)
		a.u.showErrorDialog("Failed to show colony details", err, a.u.MainWindow())
		return
	}

	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		b.stop()
	})

	setDetailWindow(detailWindowParams{
		content: b,
		title:   title,
		window:  w,
	})
	w.Show()
}

type colonyDetails struct {
	widget.BaseWidget

	characterID   int64
	cp            *app.CharacterPlanet
	icon          *iwidget.TappableImage
	infos         *widget.Form
	installations *widget.List
	issue         *widget.Label
	planetID      int64
	signalKey     string
	status        *iwidget.RichText
	u             *baseUI
}

func newColonyDetails(u *baseUI, characterID, planetID int64) *colonyDetails {
	if characterID == 0 || planetID == 0 {
		panic(app.ErrInvalid)
	}
	issue := widget.NewLabel("")
	issue.Wrapping = fyne.TextWrapWord
	issue.Importance = widget.DangerImportance
	issue.Hide()
	a := &colonyDetails{
		characterID: characterID,
		infos:       widget.NewForm(),
		issue:       issue,
		planetID:    planetID,
		signalKey:   fmt.Sprintf("colony-detail-%d-%d-%s", characterID, planetID, uniqueID()),
		status:      iwidget.NewRichText(),
		u:           u,
	}
	a.ExtendBaseWidget(a)

	a.infos.Orientation = widget.Adaptive

	a.icon = iwidget.NewTappableImage(icons.BlankSvg, func() {
		if a.cp == nil {
			return
		}
		a.u.ShowTypeInfoWindow(a.cp.EvePlanet.Type.ID)
	})
	a.icon.SetFillMode(canvas.ImageFillContain)
	a.icon.SetMinSize(fyne.NewSquareSize(64))

	list := widget.NewList(
		func() int {
			if a.cp == nil {
				return 0
			}
			return len(a.cp.Pins)
		},
		func() fyne.CanvasObject {
			return newColonyPinItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if a.cp == nil || id >= len(a.cp.Pins) {
				return
			}
			co.(*colonyPinItem).Set(a.cp, a.cp.Pins[id])
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if id >= len(a.cp.Pins) {
			return
		}
		a.u.ShowTypeInfoWindow(a.cp.Pins[id].Type.ID)
	}
	a.installations = list

	// signals
	a.u.refreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.Refresh()
		})
	}, a.signalKey)
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		if arg.characterID == a.characterID && arg.section == app.SectionCharacterPlanets {
			err := a.update(ctx)
			if err != nil {
				slog.Error("failed to update colony installations", "error", err)
				fyne.Do(func() {
					a.cp = nil
					a.setIssue("ERROR: " + a.u.humanizeError(err))
					a.Refresh()
				})
			}
		}
	})
	a.u.characterRemoved.AddListener(func(ctx context.Context, o *app.EntityShort) {
		if o.ID == a.characterID {
			fyne.Do(func() {
				a.cp = nil
				a.setIssue("Character has been removed")
				a.Refresh()
			})
		}
	})
	return a
}

func (a *colonyDetails) CreateRenderer() fyne.WidgetRenderer {
	installations := container.NewBorder(
		widget.NewLabelWithStyle("Installations", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil,
		nil,
		nil,
		a.installations,
	)
	infos := container.NewBorder(
		a.issue,
		newStandardSpacer(),
		nil,
		container.NewPadded(container.NewVBox((a.icon))),
		a.infos,
	)
	content := container.NewBorder(
		infos,
		nil,
		nil,
		nil,
		installations,
	)
	return widget.NewSimpleRenderer(content)
}

func (a *colonyDetails) stop() {
	a.u.refreshTickerExpired.RemoveListener(a.signalKey)
}

func (a *colonyDetails) setIssue(s string) {
	a.issue.SetText(s)
	a.issue.Show()
}

func (a *colonyDetails) Refresh() {
	if a.cp != nil {
		a.status.Set(colonyStatusDisplay(a.cp.ExtractionsExpiryTime()))
	}
	a.infos.Refresh()
	a.BaseWidget.Refresh()
}

func (a *colonyDetails) update(ctx context.Context) error {
	cp, err := a.u.cs.GetPlanet(context.Background(), a.characterID, a.planetID)
	if err != nil {
		return err
	}

	ownerName := a.u.scs.CharacterName(a.characterID)

	fyne.Do(func() {
		a.issue.Hide()
		a.cp = cp

		a.u.eis.InventoryTypeIconAsync(cp.EvePlanet.Type.ID, app.IconPixelSize, func(res fyne.Resource) {
			a.icon.SetResource(res)
		})

		p := theme.Padding()
		location := container.New(layout.NewCustomPaddedHBoxLayout(-2*p),
			iwidget.NewRichText(cp.EvePlanet.SolarSystem.SecurityStatusRichText()...),
			makeLinkLabel(cp.EvePlanet.Name, func() {
				a.u.ShowInfoWindow(app.EveEntitySolarSystem, cp.EvePlanet.SolarSystem.ID)
			}),
			widget.NewLabel(fmt.Sprintf("(%s)", cp.EvePlanet.SolarSystem.Constellation.Region.Name)),
		)
		fi := []*widget.FormItem{
			widget.NewFormItem("Owner", makeLinkLabel(ownerName, func() {
				a.u.ShowInfoWindow(app.EveEntityCharacter, cp.CharacterID)
			})),
			widget.NewFormItem("Planet", location),
			widget.NewFormItem("Type", container.NewHBox(
				makeLinkLabel(cp.EvePlanet.TypeDisplay(), func() {
					a.u.ShowEveEntityInfoWindow(cp.EvePlanet.Type.EveEntity())
				}),
			)),
			widget.NewFormItem("Status", a.status),
		}
		a.infos.Items = fi

		slices.SortFunc(cp.Pins, func(a, b *app.PlanetPin) int {
			return strings.Compare(a.Type.Group.Name, b.Type.Group.Name)
		})
		a.Refresh()
	})
	return nil
}

type colonyPinItem struct {
	widget.BaseWidget

	info   *widget.Label
	name   *widget.Label
	output *widget.Label
	status *widget.Label
	symbol *planetPinSymbol
}

func newColonyPinItem() *colonyPinItem {
	status := widget.NewLabel("")
	status.Alignment = fyne.TextAlignTrailing
	name := widget.NewLabel("")
	name.TextStyle.Bold = true
	w := &colonyPinItem{
		info:   widget.NewLabel(""),
		name:   name,
		output: widget.NewLabel(""),
		status: status,
		symbol: newPlanetPin(),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *colonyPinItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		nil,
		container.NewCenter(w.symbol),
		nil,
		container.New(layout.NewCustomPaddedVBoxLayout(-p),
			container.NewHBox(w.name, layout.NewSpacer(), w.status),
			container.NewHBox(w.output, layout.NewSpacer(), w.info),
		),
	)
	return widget.NewSimpleRenderer(c)
}

var installationShortNames = map[string]string{
	"Extractor Control Unit":     "Extractor",
	"Basic Industry Facility":    "Basic Processor",
	"Advanced Industry Facility": "Advanced Processor",
	"High-Tech Production Plant": "High-Tech Processor",
	"Storage Facility":           "Storage",
}

func (w *colonyPinItem) Set(cp *app.CharacterPlanet, p *app.PlanetPin) {
	prefix := cp.EvePlanet.TypeDisplay() + " "
	name, _ := strings.CutPrefix(p.Type.Name, prefix)
	if short, ok := installationShortNames[name]; ok {
		name = short
	}
	w.name.SetText(name)

	var iconColor, statusColor fyne.ThemeColorName
	var iconName eveicon.Name
	switch p.Type.Group.ID {
	case app.EveGroupCommandCenters:
		iconName = eveicon.PICommandCenter
		iconColor = colorNameInfo
		statusColor = iconColor
	case app.EveGroupExtractorControlUnits:
		iconName = eveicon.PIExtractor
		iconColor = colorNameSystem
		if v, ok := p.ExpiryTime.Value(); ok && time.Now().After(v) {
			statusColor = theme.ColorNameError
		} else {
			statusColor = iconColor
		}
	case app.EveGroupProcessors:
		iconName = eveicon.PIProcessor
		if strings.Contains(name, "Advanced") {
			iconColor = colorNameAttention
		} else {
			iconColor = theme.ColorNameWarning
		}
		statusColor = iconColor
	case app.EveGroupSpaceports:
		iconName = eveicon.PILaunchpad
		iconColor = theme.ColorNamePrimary
		statusColor = iconColor
	case app.EveGroupStorageFacilities:
		iconName = eveicon.PIStorage
		iconColor = theme.ColorNamePrimary
		statusColor = iconColor
	default:
		iconName = eveicon.Undefined
		iconColor = theme.ColorNameDisabled
		statusColor = iconColor
	}
	w.symbol.Set(eveicon.FromName(iconName), iconColor, statusColor)

	var output string
	switch p.Type.Group.ID {
	case app.EveGroupExtractorControlUnits:
		output = p.ExtractorProductType.StringFunc("-", func(v *app.EveType) string {
			return v.Name
		})
	case app.EveGroupProcessors:
		output = p.Schematic.StringFunc("-", func(v *app.EveSchematic) string {
			return v.Name
		})
	case app.EveGroupCommandCenters:
		output = fmt.Sprintf("Level %d", cp.UpgradeLevel)
	}
	w.output.SetText(output)

	var status, endDate string
	var i widget.Importance
	if p.Type.Group.ID == app.EveGroupExtractorControlUnits {
		if v, ok := p.ExpiryTime.Value(); !ok {
			status = "-"
			i = widget.LowImportance
		} else {
			endDate = v.Format(app.DateTimeFormat)
			if time.Now().After(v) {
				status = colonyIdle
				i = widget.DangerImportance
			} else {
				status = ihumanize.Duration(time.Until(v))
			}
		}
	}
	w.status.Text, w.status.Importance = status, i
	w.info.SetText(endDate)
	w.Refresh()
}

var planetPinCache xsync.Map[string, fyne.Resource]

type planetPinSymbol struct {
	widget.BaseWidget

	icon        fyne.Resource
	iconColor   fyne.ThemeColorName
	statusColor fyne.ThemeColorName
}

func newPlanetPin() *planetPinSymbol {
	w := &planetPinSymbol{
		icon:        icons.BlankSvg,
		iconColor:   theme.ColorNameForeground,
		statusColor: theme.ColorNameDisabled,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *planetPinSymbol) Set(icon fyne.Resource, iconColor fyne.ThemeColorName, statusColor fyne.ThemeColorName) {
	key := icon.Name() + string(iconColor)
	icon2, ok := planetPinCache.Load(key)
	if !ok {
		r, err := fynetools.ThemedPNG(icon, theme.Color(iconColor))
		if err != nil {
			fyne.LogError("Failed theme PNG", err)
			icon2 = icons.BlankSvg
		} else {
			planetPinCache.Store(key, r)
			icon2 = r
		}
	}
	w.icon = icon2
	w.iconColor = iconColor
	w.statusColor = statusColor
	w.Refresh()
}

func (w *planetPinSymbol) CreateRenderer() fyne.WidgetRenderer {
	c1 := canvas.NewCircle(theme.Color(w.iconColor))               // Outer
	c2 := canvas.NewCircle(theme.Color(theme.ColorNameBackground)) // Middle
	c3 := canvas.NewCircle(theme.Color(theme.ColorNameSeparator))  // Inner

	ic := canvas.NewImageFromResource(w.icon)
	ic.FillMode = canvas.ImageFillContain

	return &tripleCircleRenderer{
		circles: []*canvas.Circle{c1, c2, c3},
		icon:    ic,
		widget:  w,
	}
}

type tripleCircleRenderer struct {
	widget  *planetPinSymbol
	circles []*canvas.Circle
	icon    *canvas.Image
}

func (r *tripleCircleRenderer) Layout(size fyne.Size) {
	center := fyne.NewPos(size.Width/2, size.Height/2)
	diameter := fyne.Min(size.Width, size.Height)

	diameters := []float32{
		1.0 * diameter,
		0.85 * diameter,
		0.6 * diameter,
	}

	// Layout circles
	for i, circle := range r.circles {
		currentDim := diameters[i]

		circle.Resize(fyne.NewSize(currentDim, currentDim))
		circle.Move(fyne.NewPos(
			center.X-(currentDim/2),
			center.Y-(currentDim/2),
		))
	}

	// Layout the Icon in the center of the smallest circle
	innerCircleDim := diameters[2]
	iconDim := innerCircleDim * 0.7

	r.icon.Resize(fyne.NewSize(iconDim, iconDim))
	r.icon.Move(fyne.NewPos(
		center.X-(iconDim/2),
		center.Y-(iconDim/2),
	))
}

func (r *tripleCircleRenderer) MinSize() fyne.Size {
	return fyne.NewSquareSize(50)
}

func (r *tripleCircleRenderer) Refresh() {
	r.circles[0].FillColor = theme.Color(r.widget.statusColor)
	r.circles[0].Refresh()
	r.icon.Resource = r.widget.icon
	r.icon.Refresh()
	canvas.Refresh(r.widget)
}

func (r *tripleCircleRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.circles[0], r.circles[1], r.circles[2], r.icon}
}

func (r *tripleCircleRenderer) Destroy() {}

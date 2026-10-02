package screens

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type colonyDetailsRow struct {
	expiryTime        optional.Optional[time.Time]
	groupID           int64
	groupName         string
	info              string
	name              string
	output            string
	pinID             int64
	pinStatus         app.PinStatus
	progress          optional.Optional[float64] // 0-1, shown in the symbol
	searchTarget      string
	status            []widget.RichTextSegment
	symbolIcon        fyne.Resource
	symbolIconColor   fyne.ThemeColorName
	symbolStatusColor fyne.ThemeColorName
}

// needsAttention reports whether the pin has a problem, i.e. its ring is red.
func (r colonyDetailsRow) needsAttention() bool {
	return r.pinStatus.IsProblem()
}

const (
	colonyDetailsFilterAttention = "Needs attention"
	colonyDetailsFilterStatus    = "Status"
	colonyDetailsFilterType      = "Type"
)

const colonyDetailsIconSize = 104 // matches the height of the header text

type colonyDetails struct {
	widget.BaseWidget

	characterID   atomic.Int64
	colony        *app.CharacterPlanet
	columnSorter  *xwidget.ColumnSorter[colonyDetailsRow]
	filterChip    *xwidget.FilterChipCompact
	filterRun     latestRun
	footer        *widget.Label
	forecastRun   latestRun
	icon          *canvas.Image
	iconAttention *canvas.Image // shown over the planet icon when the colony has problems
	iconStack     *fyne.Container
	installations *widget.List
	owner         *widget.Hyperlink
	planet        *xwidget.TappableRichText
	planetID      atomic.Int64
	planetType    *widget.Hyperlink
	region        *widget.Label
	rows          []colonyDetailsRow
	rowsFiltered  []colonyDetailsRow
	rowsGen       int // incremented when Update replaces rows
	rowsRun       latestRun
	searchEntry   *xwidget.SearchEntry
	security      *xwidget.RichText
	showHelp      *xwidget.IconButton
	signalKey     string
	sortChip      *kxwidget.SortChip
	status        *xwidget.RichText
	u             baseUI
}

var colonyDetailsHelpText = fmt.Sprintf(`Status: The estimated current status of the colony and when it will stop working.
%s

Installations: Each installation shows its estimated current state.

Symbols:
• Icon: The type of installation.
• Outer ring: Green when working, gray when idle or not producing anything, red when it needs attention.
• Inner ring: The progress of the extractor program or the processor cycle, or how full a storage is.
• Red symbol and grayed-out planet: The colony needs attention or is not set up.

Extractor: The resource being extracted, the time left until the program ends and the date when it ends.

Processors: The product being produced and whether the processor is producing or idle, e.g. because it is waiting for inputs.

Storage Facility, Launchpad, Command Center: The largest contents, how full it is in percent and the used and total capacity. The Command Center also shows its upgrade level.

Problems:
• Expired: The extractor program has ended.
• Inactive: The extractor is not running.
• Not Setup: The extractor has no program or the processor has no schematic.
• Input Not Routed: An input of the processor has no route to it.
• Output Not Routed: The output has no route to another installation.
• Storage Full: The storage can not take any more incoming products.

%s

Processors which were set up shortly before the colony was last updated in game may show one batch of products, even if they never received any inputs.`, colonyStatusesHelpText, colonyEstimateHelpText("Status and contents"))

var grayscalePlanetIconCache xsync.Map[int64, fyne.Resource] // by icon ID

// showColonyDetailsWindow shows the details of a colony in a window.
func showColonyDetailsWindow(u baseUI, r colonyRow) {
	title := fmt.Sprintf("Colony %s", r.planetName)
	windowID := fmt.Sprintf("colony-%d-%d", r.characterID, r.planetID)
	w, ok, onClosed := u.GetOrCreateWindowWithOnClosed(windowID, title, r.ownerName)
	if !ok {
		w.Show()
		return
	}

	b := newColonyDetails(u, r.characterID, r.planetID)
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		b.stop()
	})

	ui.MakeDetailWindow(ui.MakeDetailWindowParams{
		Content: showWhenLoaded(b, func() {
			if err := b.Update(context.Background()); err != nil {
				slog.Error("Failed to show colony details", "characterID", r.characterID, "planetID", r.planetID, "error", err)
			}
		}),
		Title:   title,
		Window:  w,
		MinSize: fyne.NewSize(600, 600),
	})
	w.Show()
}

func newColonyDetails(u baseUI, characterID, planetID int64) *colonyDetails {
	if characterID == 0 || planetID == 0 {
		panic(app.ErrInvalid)
	}
	makeHyperLink := func() *widget.Hyperlink {
		x := widget.NewHyperlink("", nil)
		x.Wrapping = fyne.TextWrapWord
		return x
	}
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[colonyDetailsRow]{{
		Label: "Group",
		Sort: func(a, b colonyDetailsRow) int {
			return strings.Compare(a.groupName, b.groupName)
		},
	}, {
		Label: "Type",
		Sort: func(a, b colonyDetailsRow) int {
			return strings.Compare(a.name, b.name)
		},
	}, {
		Label: "End date",
		Sort: func(a, b colonyDetailsRow) int {
			return optional.CompareFunc(a.expiryTime, b.expiryTime, func(a, b time.Time) int {
				return a.Compare(b)
			})
		},
	}}),
		"Group",
		xwidget.SortAsc,
	)
	planet := xwidget.NewTappableRichText(nil, nil)
	planet.Wrapping = fyne.TextWrapWord
	a := &colonyDetails{
		columnSorter: columnSorter,
		footer:       ui.NewLabelWithTruncation(""),
		icon:         xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(colonyDetailsIconSize)),
		owner:        makeHyperLink(),
		planet:       planet,
		planetType:   makeHyperLink(),
		region:       widget.NewLabel(""),
		security:     xwidget.NewRichText(),
		signalKey:    u.Signals().UniqueKey(),
		status:       xwidget.NewRichText(),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	a.characterID.Store(characterID)
	a.planetID.Store(planetID)

	a.icon.CornerRadius = theme.InputRadiusSize()
	a.iconAttention = xwidget.NewImageFromResource(
		theme.NewColoredResource(icons.CancelSvg, theme.ColorNameError),
		fyne.NewSquareSize(theme.IconInlineSize()*2),
	)
	a.iconAttention.Hide()
	a.iconStack = container.NewStack(a.icon, container.NewCenter(a.iconAttention))

	list := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newColonyPinWidget()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			co.(*colonyPinWidget).Set(a.rowsFiltered[id])
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		title := r.name
		if a.colony != nil {
			title = fmt.Sprintf("%s on %s", r.name, a.colony.EvePlanet.Name)
		}
		showColonyPinWindow(a.u, a.characterID.Load(), a.planetID.Load(), r.pinID, title, a.owner.Text)
	}
	a.installations = list

	// filters
	a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
		a.filterRowsAsync()
	})
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync()
	})

	a.searchEntry = xwidget.NewSearchEntry("Search pins and products", func(_ string) {
		a.filterRowsAsync()
	})

	a.showHelp = xwidget.NewIconButton(theme.QuestionIcon(), func() {
		showHelpPopUp(colonyDetailsHelpText, a.u.IsMobile(), a.showHelp)
	})
	a.showHelp.SetToolTip("Show explanation")

	// signals
	a.u.Signals().RefreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.refreshForecast()
		})
	}, a.signalKey)
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if arg.CharacterID == a.characterID.Load() && arg.Section == app.SectionCharacterPlanets {
			err := a.Update(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Error("failed to update colony installations", "error", err)
				fyne.Do(func() {
					a.setIssue("ERROR: " + a.u.ErrorDisplay(err))
				})
			}
		}
	}, a.signalKey)
	a.u.Signals().CharacterRemoved.AddListener(func(_ context.Context, o *app.EntityShort) {
		if o.ID == a.characterID.Load() {
			fyne.Do(func() {
				a.setIssue("Character has been removed")
			})
		}
	}, a.signalKey)
	return a
}

func (a *colonyDetails) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	header := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			// aligns the icon with the first text line, which has inner padding
			container.New(
				layout.NewCustomPaddedLayout(theme.InnerPadding(), 0, 0, 0),
				a.iconStack,
			),
		),
		nil,
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p), a.planet, a.planetType, a.owner, a.status),
	)

	filter := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(a.filterChip, a.sortChip),
		a.searchEntry,
	)

	installations := container.NewBorder(
		container.NewVBox(
			widget.NewSeparator(),
			xwidget.NewStandardSpacer(),
			filter,
		),
		container.NewBorder(nil, nil, nil, a.showHelp, a.footer),
		nil,
		nil,
		a.installations,
	)

	content := container.NewBorder(
		header,
		nil,
		nil,
		nil,
		installations,
	)
	return widget.NewSimpleRenderer(content)
}

func (a *colonyDetails) stop() {
	a.u.Signals().RefreshTickerExpired.RemoveListener(a.signalKey)
	a.u.Signals().CharacterSectionChanged.RemoveListener(a.signalKey)
	a.u.Signals().CharacterRemoved.RemoveListener(a.signalKey)
}

func (a *colonyDetails) setIssue(s string) {
	a.footer.Text = s
	a.footer.Importance = widget.DangerImportance
	a.footer.Refresh()
}

// refreshForecast recalculates the forecast for the colony.
// A refresh never discards rows from Update, but is discarded when Update replaced them.
func (a *colonyDetails) refreshForecast() {
	cp := a.colony
	if cp == nil {
		return
	}
	isLatest := a.forecastRun.start()
	gen := a.rowsGen
	runAsync(func() {
		colonyStatus, status, rows := a.makeRows(cp, time.Now())
		fyne.Do(func() {
			if !isLatest() || a.rowsGen != gen {
				return
			}
			a.setStatus(colonyStatus, status)
			a.rows = rows
			a.filterRowsAsync()
		})
	})
}

func (a *colonyDetails) filterRowsAsync() {
	isLatest := a.filterRun.start()
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	filter := a.filterChip.Selected()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort("")

	runAsync(func() {
		if filter[colonyDetailsFilterAttention] != "" {
			rows = slices.DeleteFunc(rows, func(r colonyDetailsRow) bool {
				return !r.needsAttention()
			})
		}
		if x := filter[colonyDetailsFilterType]; x != "" {
			rows = slices.DeleteFunc(rows, func(r colonyDetailsRow) bool {
				return r.name != x
			})
		}
		if x := filter[colonyDetailsFilterStatus]; x != "" {
			rows = slices.DeleteFunc(rows, func(r colonyDetailsRow) bool {
				return r.pinStatus.Display() != x
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r colonyDetailsRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}

		typeOptions := xslices.Map(rows, func(r colonyDetailsRow) string {
			return r.name
		})
		var statusOptions []string
		for _, r := range rows {
			switch r.pinStatus {
			case app.PinStatic, app.PinStatusUndefined:
				// not shown to users
			default:
				statusOptions = append(statusOptions, r.pinStatus.Display())
			}
		}
		options := []xwidget.FilterOption{
			xwidget.NewFilterOptionToogle(colonyDetailsFilterAttention),
			xwidget.NewFilterOptionSeparator(),
			xwidget.NewFilterOptionMultiChoice(colonyDetailsFilterType, typeOptions),
			xwidget.NewFilterOptionMultiChoice(colonyDetailsFilterStatus, statusOptions),
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		footer := fmt.Sprintf("Showing %d / %d installations", len(rows), totalRows)

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.rowsFiltered = rows
			a.filterChip.SetOptions(options...)
			a.installations.Refresh()

		})
	})
}

func (a *colonyDetails) Update(ctx context.Context) error {
	reset := func() {
		fyne.Do(func() {
			xslices.Clear(&a.rows)
			a.filterRowsAsync()
		})
	}
	setInfo := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.footer.Text, a.footer.Importance = s, i
			a.footer.Refresh()
		})
	}
	characterID := a.characterID.Load()
	if characterID == 0 {
		reset()
		setInfo("No character", widget.WarningImportance)
		return nil
	}
	c, err := a.u.Character().GetCharacter(ctx, characterID)
	if err != nil {
		reset()
		setInfo("Error: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return err
	}
	cp, err := a.u.Character().GetPlanet(ctx, c.ID, a.planetID.Load())
	if err != nil {
		reset()
		setInfo("Error: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return err
	}
	isLatest := a.rowsRun.start()
	colonyStatus, status, rows := a.makeRows(cp, time.Now())

	planetIcon := colonyPlanetIcon(cp.EvePlanet.Type.IconID.ValueOrZero(), colonyStatus.IsProblem())

	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.icon.Resource = planetIcon
		a.icon.Refresh()
		a.security.Set(cp.EvePlanet.SolarSystem.SecurityStatusRichText())
		a.planet.Set(cp.NameRichText())
		a.planet.OnTapped = func() {
			a.u.InfoViewer().Show(cp.EvePlanet.SolarSystem.ToEveEntity())
		}
		a.region.SetText(fmt.Sprintf("(%s)", cp.EvePlanet.SolarSystem.Constellation.Region.Name))
		a.planetType.SetText(cp.EvePlanet.TypeDisplay())
		a.planetType.OnTapped = func() {
			a.u.InfoViewer().Show(cp.EvePlanet.Type.ToEveEntity())
		}
		a.owner.SetText(c.NameOrZero())
		a.owner.OnTapped = func() {
			a.u.InfoViewer().Show(&app.EveEntity{Category: app.EveEntityCharacter, ID: cp.CharacterID})
		}

		a.colony = cp
		a.setStatus(colonyStatus, status)
		a.rows = rows
		a.rowsGen++
		a.filterRowsAsync()
	})
	return nil
}

// setStatus shows the status of the colony.
func (a *colonyDetails) setStatus(s app.ColonyStatus, display []widget.RichTextSegment) {
	a.status.Set(display)
	if s.IsProblem() {
		a.iconAttention.Show()
		a.iconStack.Refresh() // needs Refresh after Show; Fyne won't repaint never-visible objects
	} else {
		a.iconAttention.Hide()
	}
}

// makeRows returns the colony status, its display and the rows for all pins of a colony forecasted at now.
func (a *colonyDetails) makeRows(cp *app.CharacterPlanet, now time.Time) (app.ColonyStatus, []widget.RichTextSegment, []colonyDetailsRow) {
	f := a.u.Character().ForecastPlanet(cp, now)
	status := xwidget.RichTextSegmentsFromText(f.Status.Display(), widget.RichTextStyle{
		ColorName: f.Status.Color(),
		Inline:    true,
	})
	if v, ok := f.WorkEndsAt.Value(); ok {
		status = slices.Concat(status, xwidget.RichTextSegmentsFromText(
			fmt.Sprintf(" until %s (in %s)", v.Format(app.DateTimeFormat), ihumanize.Duration(v.Sub(now))),
		))
	} else if f.WorksBeyondHorizon {
		status = slices.Concat(status, xwidget.RichTextSegmentsFromText(" for "+colonyBeyondHorizonText))
	}
	typeNames := cp.TypeNames()
	var rows []colonyDetailsRow
	for _, p := range cp.Pins {
		pinType := colonyPinTypeOf(cp, p)

		name := string(pinType)
		searchTargets := []string{strings.ToLower(name)}

		icon, iconColor := pinType.icon(), pinType.color()

		pf := f.Pins[p.ID]
		if pf == nil {
			pf = &app.PinForecast{} // pin not simulated
		}

		var output, info string
		var statusText string
		var progress optional.Optional[float64]
		statusColor := pf.Status.Color()
		switch p.Type.Group.ID {
		case app.EveGroupExtractorControlUnits:
			if v, ok := p.ExtractorProductType.Value(); ok {
				output = v.Name
				searchTargets = append(searchTargets, strings.ToLower(v.Name))
			} else {
				output = "-"
			}
			if v, ok := p.ExpiryTime.Value(); ok {
				info = v.Format(app.DateTimeFormat)
			}
			if v, ok := p.ExpiryTime.Value(); ok && pf.Status == app.PinExtracting {
				statusText = ihumanize.Duration(v.Sub(now))
				if install, ok := p.InstallTime.Value(); ok && v.After(install) {
					progress = optional.New(colonyProgress(now.Sub(install), v.Sub(install))) // of the program
				}
			} else {
				statusText = pf.Status.Display()
			}
		case app.EveGroupProcessors:
			if v, ok := p.Schematic.Value(); ok {
				output = v.Name
				searchTargets = append(searchTargets, strings.ToLower(v.Name))
				if last, ok := pf.LastRunTime.Value(); ok && pf.IsActive && v.CycleTime > 0 {
					progress = optional.New(colonyProgress(now.Sub(last), time.Duration(v.CycleTime)*time.Second)) // of the cycle
				}
			} else {
				output = "-"
			}
			statusText = pf.Status.Display()
		default:
			if v, ok := pf.Capacity.Value(); ok && v > 0 {
				progress = optional.New(min(max(pf.CapacityUsed/v, 0), 1))
				info = fmt.Sprintf("%s / %s m3", ihumanize.Comma(int64(math.Round(pf.CapacityUsed))), ihumanize.Comma(int64(v)))
				if pf.Status == app.PinStorageFull {
					statusText = pf.Status.Display()
				} else {
					statusText = fmt.Sprintf("%.0f%%", pf.CapacityUsed/v*100)
				}
			}
			contents := colonyContentsDisplay(pf.Contents, typeNames)
			if p.Type.Group.ID == app.EveGroupCommandCenters {
				output = fmt.Sprintf("Level %d", cp.UpgradeLevel)
				if contents != "" {
					output += " • " + contents
				}
			} else {
				output = contents
			}
			if output == "" {
				output = "Empty"
			}
			for id := range pf.Contents {
				if n, ok := typeNames[id]; ok {
					searchTargets = append(searchTargets, strings.ToLower(n))
				}
			}
		}
		status := xwidget.RichTextSegmentsFromText(statusText, widget.RichTextStyle{
			ColorName: statusColor,
		})

		rows = append(rows, colonyDetailsRow{
			expiryTime:        p.ExpiryTime,
			groupID:           p.Type.Group.ID,
			groupName:         p.Type.Group.Name,
			info:              info,
			name:              name,
			output:            output,
			pinID:             p.ID,
			pinStatus:         pf.Status,
			progress:          progress,
			status:            status,
			symbolIcon:        icon,
			symbolIconColor:   iconColor,
			symbolStatusColor: pf.Status.IndicatorColor(),
			searchTarget:      strings.Join(searchTargets, "~"),
		})
	}
	return f.Status, status, rows
}

// colonyPlanetIcon returns the planet icon for iconID, in grayscale when the colony needs attention.
func colonyPlanetIcon(iconID int64, needsAttention bool) fyne.Resource {
	icon, _ := eveicon.FromID(iconID)
	if !needsAttention {
		return icon
	}
	if r, ok := grayscalePlanetIconCache.Load(iconID); ok {
		return r
	}
	r, err := fynetools.ImageToGrayscale(icon)
	if err != nil {
		slog.Warn("Failed to convert planet icon to grayscale", "iconID", iconID, "error", err)
		return icon
	}
	grayscalePlanetIconCache.Store(iconID, r)
	return r
}

// colonyProgress returns the ratio of elapsed to total, clamped to 0-1.
func colonyProgress(elapsed, total time.Duration) float64 {
	if total <= 0 {
		return 0
	}
	return min(max(float64(elapsed)/float64(total), 0), 1)
}

// colonyContentsDisplay returns a short summary of the largest contents of a pin.
func colonyContentsDisplay(contents map[int64]int64, typeNames map[int64]string) string {
	const maxItems = 3
	type item struct {
		name   string
		amount int64
	}
	var items []item
	for id, amount := range contents {
		n, ok := typeNames[id]
		if !ok {
			n = fmt.Sprintf("Type #%d", id)
		}
		items = append(items, item{n, amount})
	}
	slices.SortFunc(items, func(a, b item) int {
		return cmp.Or(cmp.Compare(b.amount, a.amount), strings.Compare(a.name, b.name))
	})
	var parts []string
	for i, x := range items {
		if i == maxItems {
			parts = append(parts, fmt.Sprintf("+%d more", len(items)-maxItems))
			break
		}
		parts = append(parts, fmt.Sprintf("%s %s", x.name, ihumanize.Comma(x.amount)))
	}
	return strings.Join(parts, ", ")
}

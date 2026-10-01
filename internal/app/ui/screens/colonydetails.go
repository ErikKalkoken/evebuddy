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
	symbolIconColor   fyne.ThemeColorName
	symbolIconName    eveicon.Name
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

Installations: Each installation shows its estimated current state. The ring around the icon is green when working, yellow when idle, red when it needs attention and gray for storage.

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
		icon:         xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(ui.IconUnitSize)),
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

	list := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newColonyPinItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			co.(*colonyPinItem).Set(a.rowsFiltered[id])
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
	planet := container.NewBorder(nil, nil, a.icon, nil, a.planet)
	infos := widget.NewForm(
		widget.NewFormItem("Planet", planet),
		widget.NewFormItem("Type", a.planetType),
		widget.NewFormItem("Owner", a.owner),
		widget.NewFormItem("Status", a.status),
	)
	// infos.Orientation = widget.Adaptive

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
		infos,
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
		status, rows := a.makeRows(cp, time.Now())
		fyne.Do(func() {
			if !isLatest() || a.rowsGen != gen {
				return
			}
			a.status.Set(status)
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
	status, rows := a.makeRows(cp, time.Now())

	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.u.EVEImage().InventoryTypeIconAsync(cp.EvePlanet.Type.ID, ui.IconPixelSize, func(res fyne.Resource) {
			a.icon.Resource = res
			a.icon.Refresh()
		})
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
		a.status.Set(status)
		a.rows = rows
		a.rowsGen++
		a.filterRowsAsync()
	})
	return nil
}

type colonyPinType string

const (
	pinTypeAdvancedProcessor colonyPinType = "Advanced Processor"
	pinTypeBasicProcessor    colonyPinType = "Basic Processor"
	pinTypeCommandCenter     colonyPinType = "Command Center"
	pinTypeExtractor         colonyPinType = "Extractor"
	pinTypeHighTechProcessor colonyPinType = "High-Tech Processor"
	pinTypeSpacePort         colonyPinType = "Launchpad"
	pinTypeStorage           colonyPinType = "Storage"
	pinTypeUnknown           colonyPinType = "???"
)

var installationShortNames = map[string]colonyPinType{
	"Advanced Industry Facility": pinTypeAdvancedProcessor,
	"Basic Industry Facility":    pinTypeBasicProcessor,
	"Command Center":             pinTypeCommandCenter,
	"Extractor Control Unit":     pinTypeExtractor,
	"High-Tech Production Plant": pinTypeHighTechProcessor,
	"Launchpad":                  pinTypeSpacePort,
	"Storage Facility":           pinTypeStorage,
}

// colonyPinTypeOf returns the short type of a pin, e.g. "Extractor".
func colonyPinTypeOf(cp *app.CharacterPlanet, p *app.PlanetPin) colonyPinType {
	n, _ := strings.CutPrefix(p.Type.Name, cp.EvePlanet.TypeDisplay()+" ")
	pinType, ok := installationShortNames[n]
	if !ok {
		return pinTypeUnknown
	}
	return pinType
}

// makeRows returns the colony status and the rows for all pins of a colony forecasted at now.
func (a *colonyDetails) makeRows(cp *app.CharacterPlanet, now time.Time) ([]widget.RichTextSegment, []colonyDetailsRow) {
	f := a.u.Character().ForecastPlanet(cp, now)
	status := xwidget.RichTextSegmentsFromText(f.Status.Display(), widget.RichTextStyle{
		ColorName: f.Status.Color(),
	})
	if v, ok := f.WorkEndsAt.Value(); ok {
		status = slices.Concat(status, xwidget.RichTextSegmentsFromText(
			fmt.Sprintf(" • work ends in %s (%s)", ihumanize.Duration(v.Sub(now)), v.Format(app.DateTimeFormat)),
		))
	} else if f.WorksBeyondHorizon {
		status = slices.Concat(status, xwidget.RichTextSegmentsFromText(" • works "+colonyBeyondHorizonText))
	}
	typeNames := colonyTypeNames(cp)
	var rows []colonyDetailsRow
	for _, p := range cp.Pins {
		pinType := colonyPinTypeOf(cp, p)

		name := string(pinType)
		searchTargets := []string{strings.ToLower(name)}

		iconName, iconColor := colonyPinIconName(pinType)

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
			symbolIconColor:   iconColor,
			symbolIconName:    iconName,
			symbolStatusColor: pinSymbolStatusColor(pf.Status),
			searchTarget:      strings.Join(searchTargets, "~"),
		})
	}
	return status, rows
}

// colonyPinIconName returns the icon and its color for a pin type.
func colonyPinIconName(pinType colonyPinType) (eveicon.Name, fyne.ThemeColorName) {
	switch pinType {
	case pinTypeCommandCenter:
		return eveicon.PICommandCenter, ui.ColorNameInfo
	case pinTypeExtractor:
		return eveicon.PIExtractor, ui.ColorNameSystem
	case pinTypeBasicProcessor:
		return eveicon.PIProcessor, theme.ColorNameWarning
	case pinTypeAdvancedProcessor:
		return eveicon.PIProcessor, ui.ColorNameAttention
	case pinTypeHighTechProcessor:
		return eveicon.PIProcessor, ui.ColorNameCreative
	case pinTypeSpacePort:
		return eveicon.PILaunchpad, theme.ColorNamePrimary
	case pinTypeStorage:
		return eveicon.PIStorage, theme.ColorNamePrimary
	}
	return eveicon.Undefined, theme.ColorNameDisabled
}

// colonyPinIconResource returns icon tinted in color.
func colonyPinIconResource(icon fyne.Resource, color fyne.ThemeColorName) fyne.Resource {
	key := icon.Name() + string(color)
	if r, ok := planetPinSymbolCache.Load(key); ok {
		return r
	}
	r, err := fynetools.ThemedPNG(icon, theme.Color(color))
	if err != nil {
		fyne.LogError("Failed theme PNG", err)
		return icons.BlankSvg
	}
	planetPinSymbolCache.Store(key, r)
	return r
}

// colonyProgress returns the ratio of elapsed to total, clamped to 0-1.
func colonyProgress(elapsed, total time.Duration) float64 {
	if total <= 0 {
		return 0
	}
	return min(max(float64(elapsed)/float64(total), 0), 1)
}

// pinSymbolStatusColor returns the color of the outer ring of a pin symbol.
func pinSymbolStatusColor(s app.PinStatus) fyne.ThemeColorName {
	switch s {
	case app.PinExtracting, app.PinProducing:
		return theme.ColorNameSuccess
	case app.PinStatic, app.PinStatusUndefined:
		return theme.ColorNameButton
	}
	return s.Color()
}

// colonyTypeNames returns the names of all types known to a colony by type ID.
func colonyTypeNames(cp *app.CharacterPlanet) map[int64]string {
	m := make(map[int64]string)
	for _, p := range cp.Pins {
		for _, c := range p.Contents {
			m[c.Type.ID] = c.Type.Name
		}
		if v, ok := p.ExtractorProductType.Value(); ok {
			m[v.ID] = v.Name
		}
	}
	for _, r := range cp.Routes {
		m[r.ContentType.ID] = r.ContentType.Name
	}
	return m
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

type colonyPinItem struct {
	widget.BaseWidget

	info   *widget.Label
	name   *widget.Label
	output *widget.Label
	status *xwidget.RichText
	symbol *planetPinSymbol
}

func newColonyPinItem() *colonyPinItem {
	status := xwidget.NewRichText()
	name := widget.NewLabel("")
	name.TextStyle.Bold = true
	name.Truncation = fyne.TextTruncateClip
	output := widget.NewLabel("")
	output.Truncation = fyne.TextTruncateClip
	w := &colonyPinItem{
		info:   widget.NewLabel(""),
		name:   name,
		output: output,
		status: status,
		symbol: newPlanetPinSymbol(),
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
			container.NewBorder(nil, nil, nil, w.status, w.name),
			container.NewBorder(nil, nil, nil, w.info, w.output),
		),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *colonyPinItem) Set(r colonyDetailsRow) {
	w.info.SetText(r.info)
	w.name.SetText(r.name)
	w.output.SetText(r.output)
	w.status.Set(r.status)
	w.symbol.Set(eveicon.FromName(r.symbolIconName), r.symbolIconColor, r.symbolStatusColor, r.progress)
	w.Refresh()
}

var planetPinSymbolCache xsync.Map[string, fyne.Resource]

type planetPinSymbol struct {
	widget.BaseWidget

	icon        fyne.Resource
	iconColor   fyne.ThemeColorName
	progress    optional.Optional[float64] // 0-1, shown as arc
	statusColor fyne.ThemeColorName
}

func newPlanetPinSymbol() *planetPinSymbol {
	w := &planetPinSymbol{
		icon:        icons.BlankSvg,
		iconColor:   theme.ColorNameForeground,
		statusColor: theme.ColorNameDisabled,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *planetPinSymbol) Set(icon fyne.Resource, iconColor fyne.ThemeColorName, statusColor fyne.ThemeColorName, progress optional.Optional[float64]) {
	w.icon = colonyPinIconResource(icon, iconColor)
	w.iconColor = iconColor
	w.progress = progress
	w.statusColor = statusColor
	w.Refresh()
}

func (w *planetPinSymbol) CreateRenderer() fyne.WidgetRenderer {
	c1 := canvas.NewCircle(theme.Color(w.iconColor))               // Outer
	c2 := canvas.NewCircle(theme.Color(theme.ColorNameBackground)) // Middle
	c3 := canvas.NewCircle(theme.Color(theme.ColorNameSeparator))  // Inner

	ic := canvas.NewImageFromResource(w.icon)
	ic.FillMode = canvas.ImageFillContain

	r := &tripleCircleRenderer{
		circles:  []*canvas.Circle{c1, c2, c3},
		icon:     ic,
		progress: canvas.NewArc(0, 0, planetPinSymbolArcCutout, theme.Color(theme.ColorNameForeground)),
		track:    canvas.NewArc(0, 360, planetPinSymbolArcCutout, theme.Color(theme.ColorNameSeparator)),
		widget:   w,
	}
	r.Refresh()
	return r
}

const (
	planetPinSymbolArcCutout = 0.87 // thin ring
	planetPinMinSize         = 50
)

type tripleCircleRenderer struct {
	widget   *planetPinSymbol
	circles  []*canvas.Circle
	icon     *canvas.Image
	progress *canvas.Arc
	track    *canvas.Arc
}

func (r *tripleCircleRenderer) Layout(size fyne.Size) {
	center := fyne.NewPos(size.Width/2, size.Height/2)
	diameter := fyne.Min(size.Width, size.Height)

	diameters := []float32{
		1.0 * diameter,
		0.85 * diameter,
		0.58 * diameter,
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

	// Layout the arcs in the gap between the middle and the inner circle.
	// Despite canvas.Arc's doc, its position is the top-left of its bounding box like a circle.
	arcDim := 0.72 * diameter
	for _, arc := range []*canvas.Arc{r.track, r.progress} {
		arc.Resize(fyne.NewSquareSize(arcDim))
		arc.Move(center.Subtract(fyne.NewSquareOffsetPos(arcDim / 2)))
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
	return fyne.NewSquareSize(planetPinMinSize)
}

func (r *tripleCircleRenderer) Refresh() {
	r.circles[0].FillColor = theme.Color(r.widget.statusColor)
	r.circles[0].Refresh()
	r.track.FillColor = theme.Color(theme.ColorNameSeparator)
	r.track.Refresh()
	if v, ok := r.widget.progress.Value(); ok && v > 0 {
		r.progress.FillColor = theme.Color(theme.ColorNameForeground)
		r.progress.EndAngle = float32(360 * min(v, 1))
		r.progress.Show()
		r.progress.Refresh()
	} else {
		r.progress.Hide()
	}
	r.icon.Resource = r.widget.icon
	r.icon.Refresh()
	canvas.Refresh(r.widget)
}

func (r *tripleCircleRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.circles[0], r.circles[1], r.track, r.progress, r.circles[2], r.icon}
}

func (r *tripleCircleRenderer) Destroy() {}

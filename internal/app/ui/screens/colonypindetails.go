package screens

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"log/slog"
	"maps"
	"math"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	"github.com/nathabonfim59/fyneline"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// colonyPinInfo is the information shown for an installation.
type colonyPinInfo struct {
	found bool

	header colonyDetailsRow

	// tabs
	inputs       []colonyInputItem // only for processors, nil otherwise
	inputsEmpty  string            // shown when a processor has no inputs
	main         []ui.AttributeItem
	program      []colonyCycle // only for extractors, nil otherwise
	programTitle string
	routes       []colonyRouteItem
	storage      []colonyStorageItem // only for pins with storage, nil otherwise
}

type colonyPinDetails struct {
	widget.BaseWidget

	body           fyne.CanvasObject
	characterID    int64
	colony         *app.CharacterPlanet
	content        *fyne.Container
	extraTypeNames map[int64]string // of input types not referenced by the colony
	footer         *widget.Label
	forecastRun    latestRun
	header         *colonyPinWidget
	inputs         *colonyInputList
	inputsTab      *container.TabItem
	main           *ui.AttributeList
	mainTab        *container.TabItem
	pinID          int64
	planetID       int64
	program        *fyneline.AreaChart[colonyProgramPoint]
	programLegend  *seriesLegend
	programTab     *container.TabItem
	programTitle   *widget.Label
	routes         *colonyRouteList
	routesTab      *container.TabItem
	showPin        func(pinID int64, title string)
	rowsGen        int // incremented when Update replaces the colony
	rowsRun        latestRun
	signalKey      string
	storage        *colonyStorageList
	storageTab     *container.TabItem
	tabs           *container.AppTabs
	u              baseUI
}

// newColonyPinDetails returns a new page for an installation.
// showPin is called to show another installation and can be nil.
func newColonyPinDetails(u baseUI, characterID, planetID, pinID int64, showPin func(pinID int64, title string)) *colonyPinDetails {
	if characterID == 0 || planetID == 0 || pinID == 0 {
		panic(app.ErrInvalid)
	}
	a := &colonyPinDetails{
		characterID: characterID,
		content:     container.NewStack(),
		footer:      ui.NewLabelWithTruncation(""),
		header:      newColonyPinWidget(),
		inputs:      newColonyInputList(u),
		main:        ui.NewAttributeList(),
		pinID:       pinID,
		showPin:     showPin, // set before listeners are added, which read it
		planetID:    planetID,
		program: fyneline.NewAreaChart(nil, fyneline.TimeAccessor(func(p colonyProgramPoint) time.Time {
			return p.at
		})),
		programLegend: newSeriesLegend(),
		programTitle:  newChartTitleLabel(),
		routes:        newColonyRouteList(u),
		signalKey:     u.Signals().UniqueKey(),
		storage:       newColonyStorageList(u),
		u:             u,
	}
	a.ExtendBaseWidget(a)
	a.mainTab = container.NewTabItem("Main", a.main)
	a.programTab = container.NewTabItem("Program", newChartCard(a.programTitle, a.programLegend, a.program))
	a.inputsTab = container.NewTabItem("Inputs", a.inputs)
	a.storageTab = container.NewTabItem("Storage", a.storage)
	a.routesTab = container.NewTabItem("Routes", a.routes)

	a.program.SetCurve(fyneline.CurveStepBefore) // output is constant until the next cycle
	a.tabs = container.NewAppTabs(a.mainTab, a.routesTab)

	a.body = container.NewBorder(a.header, nil, nil, nil, a.tabs)

	a.u.Signals().RefreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.refreshForecast()
		})
	}, a.signalKey)
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if arg.CharacterID != a.characterID || arg.Section != app.SectionCharacterPlanets {
			return
		}
		if err := a.Update(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to update colony installation", "error", err)
			fyne.Do(func() {
				a.setIssue("ERROR: " + a.u.ErrorDisplay(err))
			})
		}
	}, a.signalKey)
	a.u.Signals().CharacterRemoved.AddListener(func(_ context.Context, o *app.EntityShort) {
		if o.ID == a.characterID {
			fyne.Do(func() {
				a.colony = nil // so a refresh can't forecast it again
				a.setIssue("Character has been removed")
			})
		}
	}, a.signalKey)
	return a
}

func (a *colonyPinDetails) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(nil, a.footer, nil, nil, a.content))
}

func (a *colonyPinDetails) stop() {
	a.u.Signals().RefreshTickerExpired.RemoveListener(a.signalKey)
	a.u.Signals().CharacterSectionChanged.RemoveListener(a.signalKey)
	a.u.Signals().CharacterRemoved.RemoveListener(a.signalKey)
}

func (a *colonyPinDetails) setIssue(s string) {
	a.footer.Text = s
	a.footer.Importance = widget.DangerImportance
	a.footer.Refresh()
}

func (a *colonyPinDetails) clearIssue() {
	a.footer.Text = ""
	a.footer.Importance = widget.MediumImportance
	a.footer.Refresh()
}

// Update reloads the colony and shows the installation.
func (a *colonyPinDetails) Update(ctx context.Context) error {
	isLatest := a.rowsRun.start() // before fetching, so a slower earlier fetch can't win
	cp, err := a.u.Character().GetPlanet(ctx, a.characterID, a.planetID)
	if err != nil {
		notFound := errors.Is(err, app.ErrNotFound)
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.colony = nil // so a refresh can't forecast it again
			a.rowsGen++
			if notFound {
				a.set(colonyPinInfo{}) // shows that the installation no longer exists
				a.clearIssue()
			}
		})
		if notFound {
			return nil
		}
		return err
	}
	now := time.Now()
	f := a.u.Character().ForecastPlanet(cp, now)
	extraTypeNames := a.fetchExtraTypeNames(ctx, cp, f)
	info := a.makeInfo(cp, f, extraTypeNames, now)
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.colony = cp
		a.extraTypeNames = extraTypeNames
		a.rowsGen++
		a.set(info)
		a.clearIssue()
	})
	return nil
}

// fetchExtraTypeNames returns the names of the input types of the pin, which are not referenced by the colony.
// Types which can not be found are skipped.
func (a *colonyPinDetails) fetchExtraTypeNames(ctx context.Context, cp *app.CharacterPlanet, f *app.ColonyForecast) map[int64]string {
	m := make(map[int64]string)
	pf := f.Pins[a.pinID]
	if pf == nil {
		return m
	}
	known := cp.TypeNames()
	for id := range pf.Demands {
		if _, ok := known[id]; ok {
			continue
		}
		et, err := a.u.EVEUniverse().GetType(ctx, id)
		if err != nil {
			slog.Warn("Failed to get type for colony installation", "typeID", id, "error", err)
			continue
		}
		m[id] = et.Name
	}
	return m
}

// refreshForecast recalculates the forecast for the installation.
// A refresh never discards data from Update, but is discarded when Update replaced it.
func (a *colonyPinDetails) refreshForecast() {
	cp := a.colony
	if cp == nil {
		return
	}
	extraTypeNames := a.extraTypeNames
	isLatest := a.forecastRun.start()
	gen := a.rowsGen
	runAsync(func() {
		now := time.Now()
		f := a.u.Character().ForecastPlanet(cp, now)
		info := a.makeInfo(cp, f, extraTypeNames, now)
		fyne.Do(func() {
			if !isLatest() || a.rowsGen != gen {
				return
			}
			a.set(info)
		})
	})
}

// set shows info. Must be called on the main thread.
func (a *colonyPinDetails) set(info colonyPinInfo) {
	if !info.found {
		a.content.Objects = []fyne.CanvasObject{widget.NewLabel("Installation no longer exists")}
		a.content.Refresh()
		return
	}
	a.header.Set(info.header)

	a.main.Set(info.main)
	if info.program != nil {
		a.setProgram(info.program, info.programTitle)
	}
	a.inputs.set(info.inputs, info.inputsEmpty)
	a.storage.set(info.storage)
	a.routes.set(info.routes)

	tabs := []*container.TabItem{a.mainTab}
	if info.program != nil {
		tabs = append(tabs, a.programTab)
	}
	if info.inputs != nil {
		tabs = append(tabs, a.inputsTab)
	}
	if info.storage != nil {
		tabs = append(tabs, a.storageTab)
	}
	tabs = append(tabs, a.routesTab)
	if !slices.Equal(tabs, a.tabs.Items) {
		a.tabs.SetItems(tabs) // only changes when the installation type changes
	}

	if len(a.content.Objects) != 1 || a.content.Objects[0] != a.body {
		a.content.Objects = []fyne.CanvasObject{a.body}
		a.content.Refresh()
	}
}

type colonyCyclePhase uint8

const (
	cycleCompleted colonyCyclePhase = iota
	cycleCurrent
	cycleUpcoming
)

// colonyCycle is a cycle of an extractor program.
type colonyCycle struct {
	index  int
	output int64
	phase  colonyCyclePhase
	start  time.Time
}

// colonyProgramPoint is a point of the program chart.
type colonyProgramPoint struct {
	at     time.Time
	index  int
	output int64
}

// setProgram shows the cycles of an extractor program as stepped area chart. Must be called on the main thread.
func (a *colonyPinDetails) setProgram(cycles []colonyCycle, title string) {
	a.programTitle.SetText(title)

	// each cycle is a step from its start to the next point, so the last cycle needs an end point
	var end time.Time
	if n := len(cycles); n > 0 {
		last := cycles[n-1]
		end = last.start
		if n > 1 {
			end = end.Add(last.start.Sub(cycles[n-2].start))
		}
	}
	cycles = groupCycles(cycles, colonyProgramMaxSteps)
	var points []colonyProgramPoint
	var completed int
	var hasCurrent bool
	var maxOutput int64
	for _, c := range cycles {
		points = append(points, colonyProgramPoint{at: c.start, index: c.index, output: c.output})
		switch c.phase {
		case cycleCompleted:
			completed++
		case cycleCurrent:
			hasCurrent = true
		}
		maxOutput = max(maxOutput, c.output)
	}
	if n := len(cycles); n > 0 {
		points = append(points, colonyProgramPoint{at: end, index: n, output: cycles[n-1].output})
	}

	// neighboring phases share their boundary point, so the areas join
	upcomingFrom := completed
	if hasCurrent {
		upcomingFrom++
	}
	th := a.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	phases := []struct {
		name    string
		color   fyne.ThemeColorName
		opacity float32 // of the fill
		defined func(i int) bool
	}{
		{"Completed", theme.ColorNamePlaceHolder, 0.4, func(i int) bool { return i <= completed }}, // readable, but recedes
		{"Current", theme.ColorNameSuccess, 1, func(i int) bool { return hasCurrent && (i == completed || i == completed+1) }},
		{"Upcoming", theme.ColorNamePrimary, 1, func(i int) bool { return i >= upcomingFrom }},
	}
	var series []fyneline.AreaSeries[colonyProgramPoint]
	var entries []*legendEntry
	for _, p := range phases {
		c := th.Color(p.color, v)
		series = append(series, fyneline.NewOptionalAreaSeries(p.name, func(x colonyProgramPoint) (int64, bool) {
			return x.output, p.defined(x.index)
		}).WithStyle(fyneline.AreaStyle{
			Fill:   fyneline.FillStyle{Color: c, Opacity: p.opacity},
			Stroke: fyneline.StrokeStyle{Color: c, Width: 1},
		}))
		swatch := color.NRGBAModel.Convert(c).(color.NRGBA)
		swatch.A = uint8(float32(swatch.A) * p.opacity) // matches the fill
		entries = append(entries, newLegendEntry(p.name, swatch))
	}
	a.programLegend.SetEntries(entries...)

	if len(points) > 1 {
		start, end := points[0].at, points[len(points)-1].at
		a.program.SetXAxis(fyneline.NewTimeAxis(colonyProgramDateFormat, time.UTC).
			WithDomain(float64(start.Unix()), float64(end.Unix())).
			WithTickCount(4)) // fits on mobile
		// room for the last label, which is centered on the right edge
		w := fyne.MeasureText(end.UTC().Format(colonyProgramDateFormat), theme.CaptionTextSize(), fyne.TextStyle{}).Width
		a.program.SetPadding(fyneline.Insets{Right: w / 2})
	}
	axisMax, tickCount := niceAxisBounds(float64(maxOutput), 5)
	a.program.SetYAxis(fyneline.NewNumericAxis().
		WithFormatter(func(v float64) string { return ihumanize.Comma(int64(v)) }).
		WithDomain(0, axisMax).
		WithTickCount(tickCount))
	a.program.SetSeries(series...)
	a.program.SetData(points)
}

// colonyProgramMaxSteps is the maximum number of steps in the program chart.
// Each step is drawn with chart-sized shapes, so many steps make every repaint slow.
const colonyProgramMaxSteps = 48

// colonyProgramDateFormat is the format of the time axis in the program chart.
// It includes the date, because a program can lie in the past.
const colonyProgramDateFormat = "Jan 02 15:04"

// groupCycles returns cycles merged into at most about maxSteps steps with their average output.
// Steps never span different phases.
func groupCycles(cycles []colonyCycle, maxSteps int) []colonyCycle {
	size := (len(cycles) + maxSteps - 1) / max(maxSteps, 1)
	if size <= 1 {
		return cycles
	}
	var steps []colonyCycle
	for i := 0; i < len(cycles); {
		j := i + 1
		for j < len(cycles) && j-i < size && cycles[j].phase == cycles[i].phase {
			j++
		}
		var total int64
		for _, c := range cycles[i:j] {
			total += c.output
		}
		steps = append(steps, colonyCycle{
			index:  len(steps),
			output: int64(math.Round(float64(total) / float64(j-i))),
			phase:  cycles[i].phase,
			start:  cycles[i].start,
		})
		i = j
	}
	return steps
}

// makeInfo returns the information for the installation from forecast f at now.
func (a *colonyPinDetails) makeInfo(cp *app.CharacterPlanet, f *app.ColonyForecast, extraTypeNames map[int64]string, now time.Time) colonyPinInfo {
	pins := make(map[int64]*app.PlanetPin)
	for _, p := range cp.Pins {
		pins[p.ID] = p
	}
	p, ok := pins[a.pinID]
	if !ok {
		return colonyPinInfo{}
	}
	pf := f.Pins[p.ID]
	if pf == nil {
		pf = &app.PinForecast{} // pin not simulated
	}
	typeNames := cp.TypeNames()
	maps.Copy(typeNames, extraTypeNames)
	typeName := func(id int64) string {
		if n, ok := typeNames[id]; ok {
			return n
		}
		return fmt.Sprintf("Type #%d", id)
	}
	showType := func(id int64) func() {
		return func() {
			a.u.InfoViewer().ShowType(id, 0)
		}
	}
	formatTime := func(t time.Time) string {
		return t.Format(app.DateTimeFormat)
	}
	formatRelative := func(t time.Time) string {
		if t.After(now) {
			return fmt.Sprintf("%s (in %s)", formatTime(t), ihumanize.Duration(t.Sub(now)))
		}
		return fmt.Sprintf("%s (%s ago)", formatTime(t), ihumanize.Duration(now.Sub(t)))
	}

	info := colonyPinInfo{
		found:  true,
		header: makeColonyDetailsRow(cp, p, pf, typeNames, cp.TypeVolumes(), now),
	}

	// main
	status := ui.AttributeItem{Label: "Status"}
	switch {
	case pf.Status == app.PinStatusUndefined:
		status.Value = "-"
	case pf.Status != app.PinStatic:
		status.Value = pf.Status.Display()
	case len(pf.Contents) == 0:
		status.Value = "Empty"
	default:
		status.Value = "Has space" // all incoming routes fit
	}
	top := []ui.AttributeItem{
		{Label: "Type", Value: p.Type.Name, InfoAction: showType(p.Type.ID)},
		status,
	}
	lastActivity := pf.LastRunTime
	var idleFor time.Duration
	if es, ok := p.ProcessorSchematic(); ok {
		// an idle processor keeps checking for inputs, so show its last production instead
		lastActivity = pf.LastCycleStart
		if v, ok := pf.LastCycleStart.Value(); ok && !pf.IsActive {
			end := v.Add(time.Duration(es.CycleTime) * time.Second)
			lastActivity = optional.New(end)
			idleFor = now.Sub(end)
		}
	}
	switch p.Type.Group.ID {
	case app.EveGroupExtractorControlUnits:
		product := ui.AttributeItem{Label: "Product", Value: "-"}
		if v, ok := p.ExtractorProductType.Value(); ok {
			product.Value = v.Name
			product.InfoAction = showType(v.ID)
		}
		expires := ui.AttributeItem{Label: "Expires", Value: "-"}
		if v, ok := p.ExpiryTime.Value(); ok {
			if v.After(now) {
				expires.Value = formatRelative(v)
			} else {
				expires.Value = formatTime(v) + " (expired)"
				expires.Importance = widget.DangerImportance
			}
		}
		info.main = []ui.AttributeItem{
			product,
			{Label: "Installed", Value: p.InstallTime.StringFunc("-", formatTime)},
			expires,
			{Label: "Cycle time", Value: p.ExtractorCycleTime.StringFunc("-", ihumanize.Duration)},
			{Label: "Heads", Value: p.ExtractorNumHeads.StringFunc("-", ihumanize.Comma)},
			{Label: "Base yield", Value: p.ExtractorQtyPerCycle.StringFunc("-", ihumanize.Comma)},
		}
		info.program = []colonyCycle{}
		info.programTitle = "No program"
		install, ok1 := p.InstallTime.Value()
		expiry, ok2 := p.ExpiryTime.Value()
		cycle, ok3 := p.ExtractorCycleTime.Value()
		if len(pf.ExtractorOutputs) > 0 && ok1 && ok2 && ok3 && cycle > 0 {
			total := pf.ExtractorTotalOutput()
			perHour := float64(total) / expiry.Sub(install).Hours()
			current := int(now.Sub(install) / cycle)
			info.programTitle = fmt.Sprintf(
				"Total %s • Avg. %s / h • Current %s",
				ihumanize.Comma(total),
				ihumanize.Comma(int64(math.Round(perHour))),
				pf.ExtractorCycleOutput(current).StringFunc("-", ihumanize.Comma),
			)
			for i, v := range pf.ExtractorOutputs {
				c := colonyCycle{index: i, output: v, start: install.Add(time.Duration(i) * cycle)}
				switch {
				case i < current:
					c.phase = cycleCompleted
				case i == current:
					c.phase = cycleCurrent
				default:
					c.phase = cycleUpcoming
				}
				info.program = append(info.program, c)
			}
		}
	case app.EveGroupProcessors:
		info.inputs = []colonyInputItem{}
		info.inputsEmpty = "No inputs"
		es, ok := p.ProcessorSchematic()
		if !ok {
			info.main = []ui.AttributeItem{{Label: "Schematic", Value: "-"}}
			info.inputsEmpty = "No schematic"
			break
		}
		output := ui.AttributeItem{Label: "Schematic", Value: es.Name, InfoAction: showType(pf.OutputTypeID)}
		if pf.OutputQuantity > 0 {
			output.Value += " x " + ihumanize.Comma(pf.OutputQuantity)
		}
		cycle := time.Duration(es.CycleTime) * time.Second
		nextOutput := "Waiting for inputs"
		if pf.IsActive {
			nextOutput = pf.LastRunTime.StringFunc("-", func(v time.Time) string {
				return formatRelative(v.Add(cycle))
			})
		}
		info.main = []ui.AttributeItem{
			output,
			{Label: "Cycle time", Value: ihumanize.Duration(cycle)},
			{Label: "Next output", Value: nextOutput},
		}
		if idleFor > 0 {
			info.main = append(info.main, ui.AttributeItem{Label: "Idle for", Value: ihumanize.Duration(idleFor)})
		}

		incoming := make(map[int64]bool)
		for _, r := range cp.Routes {
			if r.DestinationPinID == p.ID {
				incoming[r.ContentType.ID] = true
			}
		}
		for id, quantity := range pf.Demands {
			info.inputs = append(info.inputs, colonyInputItem{
				demand:   quantity,
				inStock:  pf.Contents[id],
				isRouted: incoming[id],
				name:     typeName(id),
				typeID:   id,
			})
		}
	default:
		if p.Type.Group.ID == app.EveGroupCommandCenters {
			info.main = append(info.main, ui.AttributeItem{Label: "Upgrade level", Value: fmt.Sprint(cp.UpgradeLevel)})
		}
		if v, ok := pf.Capacity.Value(); ok && v > 0 {
			info.main = append(info.main, ui.AttributeItem{
				Label: "Capacity",
				Value: fmt.Sprintf(
					"%s / %s m3 (%.0f%%)",
					humanize.FormatFloat("#,###.##", pf.CapacityUsed),
					ihumanize.Comma(int64(v)),
					pf.CapacityUsed/v*100,
				),
			})
		}
		volumes := cp.TypeVolumes()
		groups := cp.TypeGroupNames()
		info.storage = []colonyStorageItem{}
		for id, amount := range pf.Contents {
			info.storage = append(info.storage, colonyStorageItem{
				group:    groups[id],
				name:     typeName(id),
				quantity: amount,
				typeID:   id,
				volume:   volumes[id] * float64(amount),
			})
		}
	}
	info.main = slices.Concat(top, info.main)
	info.main = append(info.main,
		ui.AttributeItem{Label: "Last activity", Value: lastActivity.StringFunc("-", formatTime)},
		ui.AttributeItem{Label: "Data from", Value: formatRelative(cp.LastUpdate)},
	)

	// routes
	makeRoute := func(r *app.PlanetRoute, otherID int64, isIncoming bool) colonyRouteItem {
		x := colonyRouteItem{
			isIncoming: isIncoming,
			name:       typeName(r.ContentType.ID),
			quantity:   r.Quantity,
			typeID:     r.ContentType.ID,
		}
		other, ok := pins[otherID]
		if !ok {
			return x
		}
		x.otherName = cp.PinName(other)
		if a.showPin != nil {
			title := fmt.Sprintf("%s on %s", x.otherName, cp.EvePlanet.Name)
			x.onSelected = func() {
				a.showPin(otherID, title)
			}
		}
		return x
	}
	info.routes = []colonyRouteItem{}
	for _, r := range cp.Routes {
		if r.DestinationPinID == p.ID {
			info.routes = append(info.routes, makeRoute(r, r.SourcePinID, true))
		}
		if r.SourcePinID == p.ID {
			info.routes = append(info.routes, makeRoute(r, r.DestinationPinID, false))
		}
	}
	return info
}

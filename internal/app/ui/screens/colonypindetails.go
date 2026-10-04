package screens

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"image/color"
	"log/slog"
	"maps"
	"math"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	"github.com/nathabonfim59/fyneline"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// colonyPinInfo is the information shown for an installation.
type colonyPinInfo struct {
	found bool

	// header
	name        string
	onName      func()
	onProduct   func()
	product     string // empty when the installation has no product
	progress    optional.Optional[float64]
	status      []widget.RichTextSegment
	symbolColor fyne.ThemeColorName
	symbolIcon  fyne.Resource
	symbolType  colonyPinType

	// tabs
	main         []ui.AttributeItem
	program      []colonyCycle // only for extractors, nil otherwise
	programTitle string
	routes       []ui.AttributeItem
	storage      []ui.AttributeItem // only for pins with contents
	storageTitle string
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
	main           *ui.AttributeList
	mainTab        *container.TabItem
	name           *widget.Hyperlink
	pinID          int64
	planetID       int64
	product        *widget.Hyperlink
	program        *fyneline.AreaChart[colonyProgramPoint]
	programLegend  *seriesLegend
	programTab     *container.TabItem
	programTitle   *widget.Label
	routes         *ui.AttributeList
	routesTab      *container.TabItem
	showPin        func(pinID int64, title string)
	rowsGen        int // incremented when Update replaces the colony
	rowsRun        latestRun
	signalKey      string
	status         *xwidget.RichText
	storage        *ui.AttributeList
	storageTab     *container.TabItem
	symbol         *planetPinSymbol
	tabs           *container.AppTabs
	u              baseUI
}

// newColonyPinDetails returns a new page for an installation.
// showPin is called to show another installation and can be nil.
func newColonyPinDetails(u baseUI, characterID, planetID, pinID int64, showPin func(pinID int64, title string)) *colonyPinDetails {
	if characterID == 0 || planetID == 0 || pinID == 0 {
		panic(app.ErrInvalid)
	}
	makeHyperLink := func() *widget.Hyperlink {
		x := widget.NewHyperlink("", nil)
		x.Wrapping = fyne.TextWrapWord
		return x
	}
	a := &colonyPinDetails{
		characterID: characterID,
		content:     container.NewStack(),
		footer:      ui.NewLabelWithTruncation(""),
		main:        ui.NewAttributeList(),
		name:        makeHyperLink(),
		pinID:       pinID,
		showPin:     showPin, // set before listeners are added, which read it
		product:     makeHyperLink(),
		planetID:    planetID,
		program: fyneline.NewAreaChart(nil, fyneline.TimeAccessor(func(p colonyProgramPoint) time.Time {
			return p.at
		})),
		programLegend: newSeriesLegend(),
		programTitle:  newChartTitleLabel(),
		routes:        ui.NewAttributeList(),
		signalKey:     u.Signals().UniqueKey(),
		status:        xwidget.NewRichText(),
		storage:       ui.NewAttributeList(),
		symbol:        newPlanetPinSymbol(),
		u:             u,
	}
	a.ExtendBaseWidget(a)
	a.name.TextStyle.Bold = true

	a.mainTab = container.NewTabItem("Main", a.main)
	a.programTab = container.NewTabItem("Program", newChartCard(a.programTitle, a.programLegend, a.program))
	a.storageTab = container.NewTabItem("Storage", a.storage)
	a.routesTab = container.NewTabItem("Routes", a.routes)

	a.program.SetCurve(fyneline.CurveStepBefore) // output is constant until the next cycle
	a.tabs = container.NewAppTabs(a.mainTab, a.routesTab)

	p := theme.Padding()
	header := container.NewBorder(
		nil,
		nil,
		// aligns the symbol with the first text line, which has inner padding
		container.New(
			layout.NewCustomPaddedLayout(theme.InnerPadding(), 0, 0, 0),
			container.NewVBox(a.symbol),
		),
		nil,
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p), a.name, a.product, a.status),
	)
	a.body = container.NewBorder(header, nil, nil, nil, a.tabs)

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
	a.name.SetText(info.name)
	a.name.OnTapped = info.onName
	a.product.SetText(info.product)
	a.product.OnTapped = info.onProduct
	if info.product != "" {
		a.product.Show()
	} else {
		a.product.Hide()
	}
	a.status.Set(info.status)
	a.symbol.Set(info.symbolIcon, info.symbolType.color(), info.symbolColor, info.progress)

	a.main.Set(info.main)
	if info.program != nil {
		a.setProgram(info.program, info.programTitle)
	}
	a.storage.Set(info.storage)
	a.routes.Set(info.routes)

	tabs := []*container.TabItem{a.mainTab}
	if info.program != nil {
		tabs = append(tabs, a.programTab)
	}
	if info.storage != nil {
		tabs = append(tabs, a.storageTab)
	}
	tabs = append(tabs, a.routesTab)
	if !slices.Equal(tabs, a.tabs.Items) {
		a.tabs.SetItems(tabs) // only changes when the installation type changes
	}
	if a.storageTab.Text != info.storageTitle && info.storageTitle != "" {
		a.storageTab.Text = info.storageTitle
		a.tabs.Refresh()
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

	// header
	pinType := colonyPinTypeOf(cp, p)
	statusText := "-"
	var statusColor fyne.ThemeColorName
	if pf.Status != app.PinStatic && pf.Status != app.PinStatusUndefined {
		statusText = pf.Status.Display()
		statusColor = pf.Status.Color()
	}
	info := colonyPinInfo{
		found:       true,
		name:        cp.PinName(p),
		onName:      showType(p.Type.ID),
		progress:    colonyPinProgress(p, pf, now),
		status:      xwidget.RichTextSegmentsFromText(statusText, widget.RichTextStyle{ColorName: statusColor}),
		symbolColor: pf.Status.IndicatorColor(),
		symbolIcon:  pinType.icon(),
		symbolType:  pinType,
	}

	// main
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
			info.product = v.Name
			info.onProduct = showType(v.ID)
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
		info.storageTitle = "Inputs"
		info.storage = []ui.AttributeItem{}
		es, ok := p.ProcessorSchematic()
		if !ok {
			info.main = []ui.AttributeItem{{Label: "Schematic", Value: "-"}}
			info.storage = []ui.AttributeItem{{Label: "No schematic"}}
			break
		}
		output := ui.AttributeItem{Label: "Schematic", Value: es.Name, InfoAction: showType(pf.OutputTypeID)}
		info.product = es.Name
		info.onProduct = showType(pf.OutputTypeID)
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
		var inputs []quantityItem
		for id, quantity := range pf.Demands {
			it := ui.AttributeItem{
				Label:      fmt.Sprintf("%s x %s", typeName(id), ihumanize.Comma(quantity)),
				Value:      ihumanize.Comma(pf.Contents[id]) + " in stock",
				InfoAction: showType(id),
			}
			if !incoming[id] {
				it.Value += " (not routed)"
				it.Importance = widget.DangerImportance
			}
			inputs = append(inputs, quantityItem{name: typeName(id), quantity: quantity, item: it})
		}
		info.storage = sortByNameAndQuantity(inputs)
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
		info.storageTitle = "Storage"
		volumes := cp.TypeVolumes()
		var contents []quantityItem
		for id, amount := range pf.Contents {
			contents = append(contents, quantityItem{name: typeName(id), quantity: amount, item: ui.AttributeItem{
				Label:      fmt.Sprintf("%s x %s", typeName(id), ihumanize.Comma(amount)),
				Value:      humanize.FormatFloat("#,###.##", volumes[id]*float64(amount)) + " m3",
				InfoAction: showType(id),
			}})
		}
		info.storage = sortByNameAndQuantity(contents)
		if len(info.storage) == 0 {
			info.storage = append(info.storage, ui.AttributeItem{Label: "Empty"})
		}
	}
	info.main = append(info.main,
		ui.AttributeItem{Label: "Last activity", Value: lastActivity.StringFunc("-", formatTime)},
		ui.AttributeItem{Label: "Data from", Value: formatRelative(cp.LastUpdate)},
	)

	// routes
	makeRoute := func(r *app.PlanetRoute, otherID int64) quantityItem {
		name := typeName(r.ContentType.ID)
		it := ui.AttributeItem{
			Label: fmt.Sprintf("%s x %s", name, ihumanize.Comma(r.Quantity)),
			Value: "Unknown installation",
		}
		q := quantityItem{name: name, quantity: r.Quantity}
		other, ok := pins[otherID]
		if !ok {
			q.item = it
			return q
		}
		otherName := cp.PinName(other)
		it.Value = otherName
		if a.showPin != nil {
			title := fmt.Sprintf("%s on %s", otherName, cp.EvePlanet.Name)
			it.InfoAction = func() {
				a.showPin(otherID, title)
			}
		}
		q.item = it
		return q
	}
	var in, out []quantityItem
	for _, r := range cp.Routes {
		if r.DestinationPinID == p.ID {
			in = append(in, makeRoute(r, r.SourcePinID))
		}
		if r.SourcePinID == p.ID {
			out = append(out, makeRoute(r, r.DestinationPinID))
		}
	}
	info.routes = []ui.AttributeItem{}
	for _, x := range []struct {
		heading string
		items   []quantityItem
	}{{"Incoming", in}, {"Outgoing", out}} {
		if len(x.items) == 0 {
			continue
		}
		info.routes = append(info.routes, ui.AttributeItem{Label: x.heading, IsHeading: true})
		info.routes = append(info.routes, sortByNameAndQuantity(x.items)...)
	}
	if len(info.routes) == 0 {
		info.routes = []ui.AttributeItem{{Label: "No routes"}}
	}
	return info
}

// quantityItem is a list item for a quantity of a type.
type quantityItem struct {
	name     string // of the type
	quantity int64
	item     ui.AttributeItem
}

// sortByNameAndQuantity returns the items ordered by name ascending, then by quantity descending.
func sortByNameAndQuantity(s []quantityItem) []ui.AttributeItem {
	slices.SortFunc(s, func(a, b quantityItem) int {
		return cmp.Or(
			strings.Compare(a.name, b.name),
			cmp.Compare(b.quantity, a.quantity),
			strings.Compare(a.item.Value, b.item.Value),
		)
	})
	items := make([]ui.AttributeItem, 0, len(s))
	for _, x := range s {
		items = append(items, x.item)
	}
	return items
}

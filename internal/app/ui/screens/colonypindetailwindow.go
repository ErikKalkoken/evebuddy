package screens

import (
	"cmp"
	"context"
	"fmt"
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

// showColonyPinWindow shows the details of an installation of a colony in a window.
func showColonyPinWindow(u baseUI, characterID, planetID, pinID int64, title, ownerName string) {
	windowID := fmt.Sprintf("colony-pin-%d-%d-%d", characterID, planetID, pinID)
	w, ok, onClosed := u.GetOrCreateWindowWithOnClosed(windowID, title, ownerName)
	if !ok {
		w.Show()
		return
	}
	a := newColonyPinDetails(u, characterID, planetID, pinID)
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		a.stop()
	})
	ui.MakeDetailWindow(ui.MakeDetailWindowParams{
		Content: showWhenLoaded(a, func() {
			if err := a.Update(context.Background()); err != nil {
				slog.Error("Failed to show colony installation", "characterID", characterID, "planetID", planetID, "pinID", pinID, "error", err)
				fyne.Do(func() {
					a.setIssue("ERROR: " + a.u.ErrorDisplay(err))
				})
			}
		}),
		Title:  title,
		Window: w,
	})
	w.Show()
}

// colonyPinInfo is the information shown for an installation.
type colonyPinInfo struct {
	found bool

	// header
	colony      string
	name        string
	onColony    func()
	onName      func()
	onOwner     func()
	owner       string
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
	colonyLink     *widget.Hyperlink
	content        *fyne.Container
	extraTypeNames map[int64]string // of input types not referenced by the colony
	footer         *widget.Label
	forecastRun    latestRun
	main           *ui.AttributeList
	mainTab        *container.TabItem
	name           *widget.Hyperlink
	owner          *widget.Hyperlink
	ownerName      string
	pinID          int64
	planetID       int64
	program        *fyneline.AreaChart[colonyProgramPoint]
	programLegend  *seriesLegend
	programTab     *container.TabItem
	programTitle   *widget.Label
	routes         *ui.AttributeList
	routesTab      *container.TabItem
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

func newColonyPinDetails(u baseUI, characterID, planetID, pinID int64) *colonyPinDetails {
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
		colonyLink:  makeHyperLink(),
		content:     container.NewStack(),
		footer:      ui.NewLabelWithTruncation(""),
		main:        ui.NewAttributeList(),
		name:        makeHyperLink(),
		owner:       makeHyperLink(),
		pinID:       pinID,
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
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p), a.name, a.colonyLink, a.owner, a.status),
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

// Update reloads the colony and shows the installation.
func (a *colonyPinDetails) Update(ctx context.Context) error {
	isLatest := a.rowsRun.start() // before fetching, so a slower earlier fetch can't win
	c, err := a.u.Character().GetCharacter(ctx, a.characterID)
	if err != nil {
		return err
	}
	cp, err := a.u.Character().GetPlanet(ctx, a.characterID, a.planetID)
	if err != nil {
		return err
	}
	ownerName := c.NameOrZero()
	now := time.Now()
	f := a.u.Character().ForecastPlanet(cp, now)
	extraTypeNames := a.fetchExtraTypeNames(ctx, cp, f)
	info := a.makeInfo(cp, f, ownerName, extraTypeNames, now)
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.colony = cp
		a.extraTypeNames = extraTypeNames
		a.ownerName = ownerName
		a.rowsGen++
		a.set(info)
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
	ownerName := a.ownerName
	extraTypeNames := a.extraTypeNames
	isLatest := a.forecastRun.start()
	gen := a.rowsGen
	runAsync(func() {
		now := time.Now()
		f := a.u.Character().ForecastPlanet(cp, now)
		info := a.makeInfo(cp, f, ownerName, extraTypeNames, now)
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
	a.colonyLink.SetText(info.colony)
	a.colonyLink.OnTapped = info.onColony
	a.owner.SetText(info.owner)
	a.owner.OnTapped = info.onOwner
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
	n := len(cycles)
	if n > 0 {
		last := cycles[n-1]
		end := last.start
		if n > 1 {
			end = end.Add(last.start.Sub(cycles[n-2].start))
		}
		points = append(points, colonyProgramPoint{at: end, index: n, output: last.output})
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
		defined func(i int) bool
	}{
		{"Completed", theme.ColorNameDisabled, func(i int) bool { return i <= completed }},
		{"Current", theme.ColorNameSuccess, func(i int) bool { return hasCurrent && (i == completed || i == completed+1) }},
		{"Upcoming", theme.ColorNamePrimary, func(i int) bool { return i >= upcomingFrom }},
	}
	var series []fyneline.AreaSeries[colonyProgramPoint]
	var entries []*legendEntry
	for _, p := range phases {
		c := th.Color(p.color, v)
		series = append(series, fyneline.NewOptionalAreaSeries(p.name, func(x colonyProgramPoint) (int64, bool) {
			return x.output, p.defined(x.index)
		}).WithStyle(fyneline.AreaStyle{
			Fill:   fyneline.FillStyle{Color: c, Opacity: 1},
			Stroke: fyneline.StrokeStyle{Color: c, Width: 1},
		}))
		entries = append(entries, newLegendEntry(p.name, c))
	}
	a.programLegend.SetEntries(entries...)

	if len(points) > 1 {
		start, end := points[0].at, points[len(points)-1].at
		format := "15:04"
		if end.Sub(start) > 24*time.Hour {
			format = "01-02 15:04"
		}
		a.program.SetXAxis(fyneline.NewTimeAxis(format, nil).
			WithDomain(float64(start.Unix()), float64(end.Unix())).
			WithTickCount(4)) // fits on mobile
		// room for the last label, which is centered on the right edge
		w := fyne.MeasureText(end.Format(format), theme.CaptionTextSize(), fyne.TextStyle{}).Width
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

// makeInfo returns the information for the installation from forecast f at now.
func (a *colonyPinDetails) makeInfo(cp *app.CharacterPlanet, f *app.ColonyForecast, ownerName string, extraTypeNames map[int64]string, now time.Time) colonyPinInfo {
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
		colony: cp.EvePlanet.Name,
		found:  true,
		name:   colonyPinLabel(cp, p),
		onColony: func() {
			a.u.InfoViewer().Show(cp.EvePlanet.SolarSystem.ToEveEntity())
		},
		onName: showType(p.Type.ID),
		onOwner: func() {
			a.u.InfoViewer().Show(&app.EveEntity{ID: cp.CharacterID, Name: ownerName, Category: app.EveEntityCharacter})
		},
		owner:       ownerName,
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
			it := ui.AttributeItem{
				Label:      typeName(id),
				Value:      fmt.Sprintf("%s / %s", ihumanize.Comma(pf.Contents[id]), ihumanize.Comma(quantity)),
				InfoAction: showType(id),
			}
			if !incoming[id] {
				it.Value += " (not routed)"
				it.Importance = widget.DangerImportance
			}
			info.storage = append(info.storage, it)
		}
		sortAttributeItems(info.storage)
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
		info.storage = []ui.AttributeItem{}
		volumes := cp.TypeVolumes()
		ids := slices.Collect(maps.Keys(pf.Contents))
		slices.SortFunc(ids, func(a, b int64) int {
			return cmp.Or(cmp.Compare(pf.Contents[b], pf.Contents[a]), strings.Compare(typeName(a), typeName(b)))
		})
		for _, id := range ids {
			amount := pf.Contents[id]
			info.storage = append(info.storage, ui.AttributeItem{
				Label:      typeName(id),
				Value:      fmt.Sprintf("%s (%s m3)", ihumanize.Comma(amount), humanize.FormatFloat("#,###.##", volumes[id]*float64(amount))),
				InfoAction: showType(id),
			})
		}
		if len(info.storage) == 0 {
			info.storage = append(info.storage, ui.AttributeItem{Label: "Empty"})
		}
	}
	info.main = append(info.main,
		ui.AttributeItem{Label: "Last activity", Value: lastActivity.StringFunc("-", formatTime)},
		ui.AttributeItem{Label: "Data from", Value: formatRelative(cp.LastUpdate)},
	)

	// routes
	makeRoute := func(r *app.PlanetRoute, direction string, otherID int64) ui.AttributeItem {
		it := ui.AttributeItem{
			Label: typeName(r.ContentType.ID),
			Value: fmt.Sprintf("x %s %s Unknown installation", ihumanize.Comma(r.Quantity), direction),
		}
		other, ok := pins[otherID]
		if !ok {
			return it
		}
		it.Value = fmt.Sprintf("x %s %s %s", ihumanize.Comma(r.Quantity), direction, colonyPinLabel(cp, other))
		it.InfoAction = func() {
			title := fmt.Sprintf("%s on %s", colonyPinTypeOf(cp, other), cp.EvePlanet.Name)
			showColonyPinWindow(a.u, a.characterID, a.planetID, otherID, title, ownerName)
		}
		return it
	}
	var in, out []ui.AttributeItem
	for _, r := range cp.Routes {
		if r.DestinationPinID == p.ID {
			in = append(in, makeRoute(r, "from", r.SourcePinID))
		}
		if r.SourcePinID == p.ID {
			out = append(out, makeRoute(r, "to", r.DestinationPinID))
		}
	}
	sortAttributeItems(in)
	sortAttributeItems(out)
	info.routes = slices.Concat(in, out)
	if len(info.routes) == 0 {
		info.routes = []ui.AttributeItem{{Label: "No routes"}}
	}
	return info
}

// colonyPinLabel returns a label for a pin which tells it apart from other pins of the same type,
// e.g. "Basic Processor (Biofuels)".
func colonyPinLabel(cp *app.CharacterPlanet, p *app.PlanetPin) string {
	s := string(colonyPinTypeOf(cp, p))
	if v, ok := p.ExtractorProductType.Value(); ok {
		return fmt.Sprintf("%s (%s)", s, v.Name)
	}
	es, ok := p.Schematic.Value()
	if !ok {
		es, ok = p.FactorySchematic.Value()
	}
	if ok {
		return fmt.Sprintf("%s (%s)", s, es.Name)
	}
	return s
}

func sortAttributeItems(s []ui.AttributeItem) {
	slices.SortFunc(s, func(a, b ui.AttributeItem) int {
		return cmp.Or(strings.Compare(a.Label, b.Label), strings.Compare(a.Value, b.Value))
	})
}

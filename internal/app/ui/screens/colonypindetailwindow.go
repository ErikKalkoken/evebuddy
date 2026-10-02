package screens

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

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

// colonyPinField is a labeled value shown for an installation.
type colonyPinField struct {
	label  string
	value  string              // shown when there are no lines
	color  fyne.ThemeColorName // optional
	action func()              // optional, shows value as link
	icon   fyne.Resource       // optional, shown in front of the value
	lines  []colonyPinItemLine // optional, shown instead of value
}

// colonyPinItemLine is a line about an item, e.g. "Water" + "x 20".
type colonyPinItemLine struct {
	name   string
	typeID int64 // links the name to the type when set
	detail string
}

func (l colonyPinItemLine) text() string {
	return strings.TrimSpace(l.name + " " + l.detail)
}

// colonyPinInfo is the information shown for an installation.
type colonyPinInfo struct {
	found    bool
	general  []colonyPinField
	specific []colonyPinField
	routes   []colonyPinField
}

type colonyPinDetails struct {
	widget.BaseWidget

	characterID    int64
	colony         *app.CharacterPlanet
	content        *fyne.Container
	extraTypeNames map[int64]string // of input types not referenced by the colony
	footer         *widget.Label
	forecastRun    latestRun
	ownerName      string
	pinID          int64
	planetID       int64
	rowsGen        int // incremented when Update replaces the colony
	rowsRun        latestRun
	signalKey      string
	u              baseUI
}

func newColonyPinDetails(u baseUI, characterID, planetID, pinID int64) *colonyPinDetails {
	if characterID == 0 || planetID == 0 || pinID == 0 {
		panic(app.ErrInvalid)
	}
	a := &colonyPinDetails{
		characterID: characterID,
		content:     container.NewVBox(),
		footer:      ui.NewLabelWithTruncation(""),
		pinID:       pinID,
		planetID:    planetID,
		signalKey:   u.Signals().UniqueKey(),
		u:           u,
	}
	a.ExtendBaseWidget(a)

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
	c, err := a.u.Character().GetCharacter(ctx, a.characterID)
	if err != nil {
		return err
	}
	cp, err := a.u.Character().GetPlanet(ctx, a.characterID, a.planetID)
	if err != nil {
		return err
	}
	isLatest := a.rowsRun.start()
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
	// one form for all sections, so labels have the same width
	f := widget.NewForm()
	f.Orientation = widget.Adaptive
	for _, x := range slices.Concat(info.general, info.specific, info.routes) {
		f.Append(x.label, a.makeFieldWidget(x))
	}
	a.content.Objects = []fyne.CanvasObject{f}
	a.content.Refresh()
}

// makeFieldWidget returns the widget for showing the value of a field.
func (a *colonyPinDetails) makeFieldWidget(x colonyPinField) fyne.CanvasObject {
	w := a.makeValueWidget(x)
	if x.icon == nil {
		return w
	}
	if h, ok := w.(*widget.Hyperlink); ok {
		h.Wrapping = fyne.TextWrapOff // would wrap every word in a HBox
	}
	icon := xwidget.NewImageFromResource(x.icon, fyne.NewSquareSize(theme.Size(theme.SizeNameInlineIcon)))
	// no extra spacing, as the value already has inner padding
	return container.New(layout.NewCustomPaddedHBoxLayout(0), icon, w)
}

func (a *colonyPinDetails) makeValueWidget(x colonyPinField) fyne.CanvasObject {
	switch {
	case len(x.lines) > 0:
		box := container.NewVBox()
		for _, l := range x.lines {
			var name fyne.CanvasObject
			if l.typeID != 0 {
				name = ui.MakeLinkLabel(l.name, func() {
					a.u.InfoViewer().ShowType(l.typeID, 0)
				})
			} else {
				name = widget.NewLabel(l.name)
			}
			detail := widget.NewLabel(l.detail)
			detail.Wrapping = fyne.TextWrapWord
			box.Add(container.NewBorder(nil, nil, name, nil, detail))
		}
		return box
	case x.action != nil:
		return ui.MakeLinkLabelWithWrap(x.value, x.action)
	case x.color != "":
		return xwidget.NewRichText(xwidget.RichTextSegmentsFromText(x.value, widget.RichTextStyle{
			ColorName: x.color,
		})...)
	}
	l := widget.NewLabel(x.value)
	l.Wrapping = fyne.TextWrapWord
	return l
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
	formatTime := func(t time.Time) string {
		return t.Format(app.DateTimeFormat)
	}
	formatRelative := func(t time.Time) string {
		if t.After(now) {
			return fmt.Sprintf("%s (in %s)", formatTime(t), ihumanize.Duration(t.Sub(now)))
		}
		return fmt.Sprintf("%s (%s ago)", formatTime(t), ihumanize.Duration(now.Sub(t)))
	}

	info := colonyPinInfo{found: true}

	// general
	status := colonyPinField{label: "Status", value: "-"}
	if pf.Status != app.PinStatic && pf.Status != app.PinStatusUndefined {
		status.value = pf.Status.Display()
		status.color = pf.Status.Color()
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
	pinType := colonyPinTypeOf(cp, p)
	icon, iconColor := pinType.iconAndColor()
	info.general = []colonyPinField{
		{label: "Installation", value: string(pinType), icon: colonyPinIconResource(icon, iconColor), action: func() {
			a.u.InfoViewer().ShowType(p.Type.ID, 0)
		}},
		{label: "Colony", value: cp.EvePlanet.Name, action: func() {
			a.u.InfoViewer().Show(cp.EvePlanet.SolarSystem.ToEveEntity())
		}},
		{label: "Owner", value: ownerName, action: func() {
			a.u.InfoViewer().Show(&app.EveEntity{ID: cp.CharacterID, Name: ownerName, Category: app.EveEntityCharacter})
		}},
		status,
		{label: "Last activity", value: lastActivity.StringFunc("-", formatTime)},
		{label: "Data from", value: formatRelative(cp.LastUpdate)},
	}

	// specific
	switch p.Type.Group.ID {
	case app.EveGroupExtractorControlUnits:
		product := colonyPinField{label: "Product", value: "-"}
		if v, ok := p.ExtractorProductType.Value(); ok {
			product.value = v.Name
			product.action = func() {
				a.u.InfoViewer().ShowType(v.ID, 0)
			}
		}
		expires := p.ExpiryTime.StringFunc("-", func(v time.Time) string {
			if !v.After(now) {
				return formatTime(v) + " (expired)"
			}
			return formatRelative(v)
		})
		info.specific = []colonyPinField{
			product,
			{label: "Installed", value: p.InstallTime.StringFunc("-", formatTime)},
			{label: "Expires", value: expires},
			{label: "Cycle time", value: p.ExtractorCycleTime.StringFunc("-", ihumanize.Duration)},
			{label: "Heads", value: p.ExtractorNumHeads.StringFunc("-", ihumanize.Comma)},
			{label: "Base yield", value: p.ExtractorQtyPerCycle.StringFunc("-", ihumanize.Comma)},
		}
	case app.EveGroupProcessors:
		es, ok := p.ProcessorSchematic()
		if !ok {
			info.specific = []colonyPinField{{label: "Schematic", value: "-"}}
			break
		}
		output := colonyPinItemLine{name: es.Name, typeID: pf.OutputTypeID}
		if pf.OutputQuantity > 0 {
			output.detail = "x " + ihumanize.Comma(pf.OutputQuantity)
		}
		cycle := time.Duration(es.CycleTime) * time.Second
		incoming := make(map[int64]bool)
		for _, r := range cp.Routes {
			if r.DestinationPinID == p.ID {
				incoming[r.ContentType.ID] = true
			}
		}
		var inputs []colonyPinItemLine
		for id, quantity := range pf.Demands {
			detail := fmt.Sprintf("%s / %s", ihumanize.Comma(pf.Contents[id]), ihumanize.Comma(quantity))
			if !incoming[id] {
				detail += " (not routed)"
			}
			inputs = append(inputs, colonyPinItemLine{name: typeName(id), typeID: id, detail: detail})
		}
		sortColonyPinItemLines(inputs)
		nextOutput := "Waiting for inputs"
		if pf.IsActive {
			nextOutput = pf.LastRunTime.StringFunc("-", func(v time.Time) string {
				return formatRelative(v.Add(cycle))
			})
		}
		info.specific = []colonyPinField{
			{label: "Schematic", lines: []colonyPinItemLine{output}},
			{label: "Cycle time", value: ihumanize.Duration(cycle)},
			{label: "Inputs", value: "-", lines: inputs},
			{label: "Next output", value: nextOutput},
		}
		if idleFor > 0 {
			info.specific = append(info.specific, colonyPinField{label: "Idle for", value: ihumanize.Duration(idleFor)})
		}
	default:
		if p.Type.Group.ID == app.EveGroupCommandCenters {
			info.specific = append(info.specific, colonyPinField{label: "Upgrade level", value: fmt.Sprint(cp.UpgradeLevel)})
		}
		if v, ok := pf.Capacity.Value(); ok && v > 0 {
			info.specific = append(info.specific, colonyPinField{
				label: "Capacity",
				value: fmt.Sprintf(
					"%s / %s m3 (%.0f%%)",
					humanize.FormatFloat("#,###.##", pf.CapacityUsed),
					ihumanize.Comma(int64(v)),
					pf.CapacityUsed/v*100,
				),
			})
		}
		volumes := cp.TypeVolumes()
		ids := slices.Collect(maps.Keys(pf.Contents))
		slices.SortFunc(ids, func(a, b int64) int {
			return cmp.Or(cmp.Compare(pf.Contents[b], pf.Contents[a]), strings.Compare(typeName(a), typeName(b)))
		})
		var contents []colonyPinItemLine
		for _, id := range ids {
			amount := pf.Contents[id]
			contents = append(contents, colonyPinItemLine{
				name:   typeName(id),
				typeID: id,
				detail: fmt.Sprintf("%s (%s m3)", ihumanize.Comma(amount), humanize.FormatFloat("#,###.##", volumes[id]*float64(amount))),
			})
		}
		info.specific = append(info.specific, colonyPinField{label: "Contents", value: "Empty", lines: contents})
	}

	// routes
	pinLabel := func(id int64) string {
		other, ok := pins[id]
		if !ok {
			return "Unknown installation"
		}
		return colonyPinLabel(cp, other)
	}
	var in, out []colonyPinItemLine
	for _, r := range cp.Routes {
		id := r.ContentType.ID
		quantity := "x " + ihumanize.Comma(r.Quantity)
		if r.DestinationPinID == p.ID {
			in = append(in, colonyPinItemLine{name: typeName(id), typeID: id, detail: quantity + " from " + pinLabel(r.SourcePinID)})
		}
		if r.SourcePinID == p.ID {
			out = append(out, colonyPinItemLine{name: typeName(id), typeID: id, detail: quantity + " to " + pinLabel(r.DestinationPinID)})
		}
	}
	sortColonyPinItemLines(in)
	sortColonyPinItemLines(out)
	info.routes = []colonyPinField{
		{label: "Incoming routes", value: "None", lines: in},
		{label: "Outgoing routes", value: "None", lines: out},
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

func sortColonyPinItemLines(s []colonyPinItemLine) {
	slices.SortFunc(s, func(a, b colonyPinItemLine) int {
		return cmp.Or(strings.Compare(a.name, b.name), strings.Compare(a.detail, b.detail))
	})
}

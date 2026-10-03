package colonysim

import (
	"math"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/evesde"
)

type pinKind uint

const (
	kindUnknown pinKind = iota
	kindCommandCenter
	kindExtractor
	kindFactory
	kindLaunchpad
	kindStorage
)

// Fallback capacities in m3 for when the type has no capacity.
var defaultCapacities = map[pinKind]float64{
	kindCommandCenter: 500,
	kindLaunchpad:     10_000,
	kindStorage:       12_000,
}

func pinKindFromGroupID(groupID int64) pinKind {
	switch groupID {
	case app.EveGroupCommandCenters:
		return kindCommandCenter
	case app.EveGroupExtractorControlUnits:
		return kindExtractor
	case app.EveGroupProcessors:
		return kindFactory
	case app.EveGroupSpaceports:
		return kindLaunchpad
	case app.EveGroupStorageFacilities:
		return kindStorage
	}
	return kindUnknown
}

// pin is the simulated state of a planet pin.
type pin struct {
	id          int64
	kind        pinKind
	contents    map[int64]int64 // amount by type ID
	capacity    float64         // m3, storage pins only
	isActive    bool
	lastRunTime time.Time // zero when unknown

	// extractor
	baseValue     int64
	cycleTime     time.Duration
	expiryTime    time.Time
	installTime   time.Time
	productTypeID int64

	// factory
	schematic               *evesde.PlanetSchematic
	demands                 map[int64]int64 // input quantity by type ID
	hasReceivedInputs       bool
	receivedInputsLastCycle bool
	lastCycleStartTime      time.Time
}

// newPin returns a new pin from a snapshot taken at lastUpdate
// or false if the pin is of an unknown kind.
func newPin(pp *app.PlanetPin, lastUpdate time.Time) (*pin, bool) {
	var kind pinKind
	if pp.Type != nil && pp.Type.Group != nil {
		kind = pinKindFromGroupID(pp.Type.Group.ID)
	}
	if kind == kindUnknown {
		return nil, false
	}
	p := &pin{
		id:          pp.ID,
		kind:        kind,
		contents:    make(map[int64]int64),
		lastRunTime: pp.LastCycleStart.ValueOrZero(),
	}
	for _, c := range pp.Contents {
		p.contents[c.Type.ID] += c.Amount
	}
	switch kind {
	case kindExtractor:
		p.baseValue = pp.ExtractorQtyPerCycle.ValueOrZero()
		p.cycleTime = pp.ExtractorCycleTime.ValueOrZero()
		p.expiryTime = pp.ExpiryTime.ValueOrZero()
		p.installTime = pp.InstallTime.ValueOrZero()
		if v, ok := pp.ExtractorProductType.Value(); ok {
			p.productTypeID = v.ID
		}
		p.isActive = p.isExtractorSetup() && lastUpdate.Before(p.expiryTime) && !pp.LastCycleStart.IsEmpty()
	case kindFactory:
		es, ok := pp.Schematic.Value()
		if !ok {
			es, ok = pp.FactorySchematic.Value()
		}
		if ok {
			if s, found := evesde.PlanetSchematicByID(es.ID); found {
				p.schematic = &s
				p.demands = make(map[int64]int64)
				for _, x := range s.Inputs {
					p.demands[x.TypeID] = x.Quantity
				}
			}
		}
		p.lastCycleStartTime = pp.LastCycleStart.ValueOrZero()
		if p.schematic != nil {
			// Known limitation, same as RIFT: ESI can report a last cycle start for a factory
			// that never ran (e.g. right after setting its schematic), which yields one phantom batch.
			p.isActive = lastUpdate.Sub(p.lastCycleStartTime) < p.schematic.CycleTime
		}
		// ensures the factory is evaluated at least once
		p.hasReceivedInputs = true
		p.receivedInputsLastCycle = true
	default:
		p.capacity = defaultCapacities[kind]
		if v := pp.Type.Capacity.ValueOrZero(); v > 0 {
			p.capacity = v
		}
	}
	return p, true
}

func (p *pin) isStorage() bool {
	switch p.kind {
	case kindCommandCenter, kindLaunchpad, kindStorage:
		return true
	}
	return false
}

// isExtractorSetup reports whether an extractor has a complete program.
func (p *pin) isExtractorSetup() bool {
	return !p.installTime.IsZero() && !p.expiryTime.IsZero() && p.cycleTime > 0 && p.baseValue > 0 && p.productTypeID != 0
}

func (p *pin) isFactory() bool {
	return p.kind == kindFactory
}

func (p *pin) cycle() time.Duration {
	switch p.kind {
	case kindExtractor:
		return p.cycleTime
	case kindFactory:
		if p.schematic != nil {
			return p.schematic.CycleTime
		}
	}
	return 0
}

func (p *pin) isActiveNow() bool {
	if p.kind == kindExtractor {
		return p.productTypeID != 0 && p.isActive
	}
	return p.isActive
}

func (p *pin) canActivate() bool {
	switch p.kind {
	case kindExtractor:
		return p.isActive && p.productTypeID != 0
	case kindFactory:
		if p.schematic == nil {
			return false
		}
		if p.isActive {
			return true
		}
		if p.hasReceivedInputs || p.receivedInputsLastCycle {
			return true
		}
		// idle factory with enough inputs is only restarted when receiving inputs (same as RIFT)
		return !p.hasEnoughInputs()
	}
	return false
}

// nextRunTime returns the next time the pin should run
// or false if it should run right away.
func (p *pin) nextRunTime() (time.Time, bool) {
	if p.kind == kindFactory && !p.isActive && p.hasEnoughInputs() {
		return time.Time{}, false
	}
	if p.lastRunTime.IsZero() {
		return time.Time{}, false
	}
	return p.lastRunTime.Add(p.cycle()), true
}

// canRun reports whether the pin can run until the given time.
func (p *pin) canRun(until time.Time) bool {
	if !p.isRunnable() {
		return false
	}
	t, ok := p.nextRunTime()
	return !ok || !t.After(until)
}

// isRunnable reports whether the pin can run at all.
func (p *pin) isRunnable() bool {
	switch p.kind {
	case kindExtractor:
		return p.canActivate()
	case kindFactory:
		return p.isActive || p.canActivate()
	}
	return false
}

// run runs the pin at runTime and returns the produced commodities.
func (p *pin) run(runTime time.Time) map[int64]int64 {
	products := make(map[int64]int64)
	switch p.kind {
	case kindExtractor:
		p.lastRunTime = runTime
		if p.productTypeID == 0 || !p.isActive {
			return products
		}
		if p.baseValue > 0 && !p.installTime.IsZero() && p.cycleTime > 0 {
			products[p.productTypeID] = extractorOutput(p.baseValue, p.installTime, runTime, p.cycleTime)
		}
		if !p.expiryTime.IsZero() && !p.expiryTime.After(runTime) {
			p.isActive = false
		}
	case kindFactory:
		if p.isActive && p.schematic != nil {
			products[p.schematic.OutputTypeID] = p.schematic.OutputQuantity
		}
		if p.hasEnoughInputs() {
			for typeID, quantity := range p.demands {
				p.removeCommodity(typeID, quantity)
			}
			p.isActive = true
			p.lastCycleStartTime = runTime
		} else {
			p.isActive = false
		}
		p.receivedInputsLastCycle = p.hasReceivedInputs
		p.hasReceivedInputs = false
		p.lastRunTime = runTime
	default:
		p.lastRunTime = runTime
		p.isActive = false
	}
	return products
}

func (p *pin) hasEnoughInputs() bool {
	for typeID, quantity := range p.demands {
		if p.contents[typeID] < quantity {
			return false
		}
	}
	return true
}

// inputBufferState returns the sort key for routing to factories. Fuller buffers have lower keys.
func (p *pin) inputBufferState() float64 {
	if len(p.demands) == 0 {
		return 0
	}
	var ratio float64
	for typeID, quantity := range p.demands {
		ratio += float64(p.contents[typeID]) / float64(quantity)
	}
	return (1 - ratio) / float64(len(p.demands))
}

func (p *pin) removeCommodity(typeID, quantity int64) int64 {
	current, ok := p.contents[typeID]
	if !ok {
		return 0
	}
	if current <= quantity {
		delete(p.contents, typeID)
		return current
	}
	p.contents[typeID] = current - quantity
	return quantity
}

// extractorProgram returns the predicted output per cycle of an extractor program.
func extractorProgram(baseValue int64, installTime, expiryTime time.Time, cycleTime time.Duration) []int64 {
	if cycleTime <= 0 || installTime.IsZero() || !expiryTime.After(installTime) {
		return nil
	}
	n := int(expiryTime.Sub(installTime) / cycleTime)
	s := make([]int64, n)
	for i := range n {
		s[i] = extractorOutput(baseValue, installTime, installTime.Add(time.Duration(i+1)*cycleTime), cycleTime)
	}
	return s
}

// extractorOutput returns the amount extracted by the cycle of an extractor program ending at runTime.
func extractorOutput(baseValue int64, installTime, runTime time.Time, cycleTime time.Duration) int64 {
	const sec = 10_000_000 // ticks per second
	const decayFactor = 0.012
	const noiseFactor = 0.8
	cycle := int64(cycleTime/time.Second) * sec
	if cycle <= 0 {
		return 0
	}
	timeDiff := (runTime.Unix() - installTime.Unix()) * sec
	cycleNum := max((timeDiff+sec)/cycle-1, 0)
	barWidth := float64(cycle/sec) / 900.0
	t := (float64(cycleNum) + 0.5) * barWidth
	decayValue := float64(baseValue) / (1 + t*decayFactor)
	phaseShift := math.Pow(float64(baseValue), 0.7)
	sinA := math.Cos(phaseShift + t*(1.0/12.0))
	sinB := math.Cos(phaseShift/2.0 + t*(1.0/5.0))
	sinC := math.Cos(t * (1.0 / 2.0))
	sinStuff := max(0, (sinA+sinB+sinC)/3.0)
	barHeight := decayValue * (1 + noiseFactor*sinStuff)
	output := barWidth * barHeight
	// integers are also rounded down, e.g. 123.0 -> 122
	if output == math.Trunc(output) {
		return int64(output) - 1
	}
	return int64(output)
}

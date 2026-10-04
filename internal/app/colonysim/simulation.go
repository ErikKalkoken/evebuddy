package colonysim

import (
	"cmp"
	"container/heap"
	"maps"
	"math"
	"slices"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

const maxEvents = 1_000_000 // guard against runaway simulations

// runResult reports why a simulation run stopped.
type runResult uint

const (
	runCompleted runResult = iota // reached the given time
	runWorkEnded                  // the colony stopped working
	runAborted                    // exceeded the event limit
)

type route struct {
	id            int64
	sourceID      int64
	destinationID int64
	typeID        int64
	quantity      int64
}

type sortedRoute struct {
	route
	key float64
}

// simulation is a simulation of a PI colony.
type simulation struct {
	pins    map[int64]*pin
	pinIDs  []int64 // sorted for deterministic iteration
	routes  []route // sorted by ID
	simTime time.Time
	volumes map[int64]float64 // m3 per unit by type ID
	atRest  bool              // nothing can change anymore, see isAtRest

	queue     eventQueue
	scheduled map[int64]event // currently valid event by pin ID
	seq       uint64
}

// newSimulation returns a new simulation of a colony, starting at the time of its last snapshot.
func newSimulation(cp *app.CharacterPlanet) *simulation {
	s := &simulation{
		pins:    make(map[int64]*pin),
		simTime: cp.LastUpdate,
		volumes: make(map[int64]float64),
	}
	addVolume := func(et *app.EveType) {
		if et != nil {
			s.volumes[et.ID] = et.Volume.ValueOrZero()
		}
	}
	for _, pp := range cp.Pins {
		p, ok := newPin(pp, cp.LastUpdate)
		if !ok {
			continue
		}
		s.pins[p.id] = p
		for _, c := range pp.Contents {
			addVolume(c.Type)
		}
		if v, ok := pp.ExtractorProductType.Value(); ok {
			addVolume(v)
		}
	}
	s.pinIDs = slices.Sorted(maps.Keys(s.pins))
	for _, r := range cp.Routes {
		if s.pins[r.SourcePinID] == nil || s.pins[r.DestinationPinID] == nil || r.ContentType == nil {
			continue
		}
		s.routes = append(s.routes, route{
			id:            r.RouteID,
			sourceID:      r.SourcePinID,
			destinationID: r.DestinationPinID,
			typeID:        r.ContentType.ID,
			quantity:      r.Quantity,
		})
		addVolume(r.ContentType)
	}
	slices.SortFunc(s.routes, func(a, b route) int {
		return cmp.Compare(a.id, b.id)
	})
	return s
}

// runUntil advances the simulation to t.
// It reports whether the simulation completed normally.
func (s *simulation) runUntil(t time.Time) bool {
	if !t.After(s.simTime) {
		return true
	}
	_, r := s.run(t, false)
	return r != runAborted
}

// runUntilWorkEnds advances the simulation until the colony stops working
// and returns the time when that happened.
// It returns [runCompleted] when the colony is still working at the horizon.
func (s *simulation) runUntilWorkEnds(horizon time.Time) (time.Time, runResult) {
	return s.run(horizon, true)
}

// forecast returns the state of the colony at the current simulation time.
func (s *simulation) forecast() *app.ColonyForecast {
	status, pinStatuses := s.colonyStatus(s.simTime)
	f := &app.ColonyForecast{
		Pins:   make(map[int64]*app.PinForecast, len(s.pins)),
		Status: status,
		Time:   s.simTime,
	}
	for _, id := range s.pinIDs {
		p := s.pins[id]
		pf := &app.PinForecast{
			Contents: maps.Clone(p.contents),
			IsActive: p.isActiveNow(),
			Status:   pinStatuses[id],
		}
		if !p.lastRunTime.IsZero() {
			pf.LastRunTime = optional.New(p.lastRunTime)
		}
		if p.isStorage() {
			pf.Capacity = optional.New(p.capacity)
			pf.CapacityUsed = s.usedVolume(p)
		}
		if p.kind == kindExtractor && p.productTypeID != 0 && p.baseValue > 0 {
			pf.ExtractorOutputs = extractorProgram(p.baseValue, p.installTime, p.expiryTime, p.cycleTime)
		}
		if p.isFactory() && !p.lastCycleStartTime.IsZero() {
			pf.LastCycleStart = optional.New(p.lastCycleStartTime)
		}
		if p.schematic != nil {
			pf.Demands = maps.Clone(p.demands)
			pf.OutputQuantity = p.schematic.OutputQuantity
			pf.OutputTypeID = p.schematic.OutputTypeID
		}
		f.Pins[id] = pf
	}
	return f
}

// nextChange returns the earliest time after the current simulation time when the state can change,
// or false when it never changes. The time is conservative: it can be earlier, but never later.
func (s *simulation) nextChange() (time.Time, bool) {
	var next time.Time
	update := func(t time.Time) {
		if next.IsZero() || t.Before(next) {
			next = t
		}
	}
	for _, id := range s.pinIDs {
		p := s.pins[id]
		// pinStatus compares the expiry with the time of the forecast
		if p.kind == kindExtractor && p.isExtractorSetup() && p.expiryTime.After(s.simTime) {
			update(p.expiryTime)
		}
		if s.atRest || !p.isRunnable() {
			continue
		}
		t, ok := p.nextRunTime()
		if !ok || !t.After(s.simTime) {
			t = s.simTime // runs right away, see schedulePin
		}
		update(t)
	}
	return next, !next.IsZero()
}

// run runs the simulation until the given time or until the colony stops working.
// It returns the simulation time at the end and why it stopped.
func (s *simulation) run(until time.Time, untilWorkEnds bool) (time.Time, runResult) {
	if s.atRest {
		if untilWorkEnds {
			return s.simTime, runWorkEnded
		}
		s.simTime = until
		return until, runCompleted
	}
	s.queue = eventQueue{}
	s.scheduled = make(map[int64]event)
	for _, id := range s.pinIDs {
		if p := s.pins[id]; p.canRun(until) {
			s.schedulePin(p)
		}
	}
	for events := 0; s.queue.Len() > 0; events++ {
		if events > maxEvents {
			return s.simTime, runAborted
		}
		e := heap.Pop(&s.queue).(event)
		if current, ok := s.scheduled[e.pinID]; !ok || current.seq != e.seq {
			continue // superseded
		}
		delete(s.scheduled, e.pinID)
		// check once all events at the current time are done
		if e.time.After(s.simTime) {
			if untilWorkEnds {
				if status, _ := s.colonyStatus(s.simTime); !status.IsWorking() {
					return s.simTime, runWorkEnded
				}
			}
			// unlike RIFT, idle factories at rest do not run anymore, which keeps their last run time
			if s.isAtRest() {
				s.atRest = true
				s.simTime = until
				return until, runCompleted
			}
		}
		if e.time.After(until) {
			s.simTime = until
			return until, runCompleted
		}
		s.simTime = e.time
		if p := s.pins[e.pinID]; p.canRun(until) {
			s.evaluatePin(p)
		}
	}
	if untilWorkEnds {
		return s.simTime, runWorkEnded
	}
	s.simTime = until
	return until, runCompleted
}

// isAtRest reports whether the colony cannot change anymore:
// No producer is active and no idle factory will start or pull inputs on its next run.
// Storages only change through producers and pulls, so then every further run changes nothing.
func (s *simulation) isAtRest() bool {
	for _, p := range s.pins {
		switch p.kind {
		case kindExtractor:
			if p.isActive {
				return false
			}
		case kindFactory:
			if p.schematic == nil {
				continue // can never run
			}
			if p.isActive || p.hasReceivedInputs || p.receivedInputsLastCycle {
				return false
			}
		}
	}
	// checking routes is more expensive, so it comes last
	for _, p := range s.pins {
		if p.kind == kindFactory && p.schematic != nil && s.canPull(p) {
			return false
		}
	}
	return true
}

// canPull reports whether a factory would pull inputs from storages routed to it, see routeInput.
func (s *simulation) canPull(destination *pin) bool {
	for _, r := range s.routes {
		if r.destinationID != destination.id {
			continue
		}
		source := s.pins[r.sourceID]
		if !source.isStorage() {
			continue
		}
		if s.canAccept(destination, r.typeID, min(source.contents[r.typeID], r.quantity)) > 0 {
			return true
		}
	}
	return false
}

func (s *simulation) schedulePin(p *pin) {
	next, ok := p.nextRunTime()
	if current, found := s.scheduled[p.id]; found {
		if ok && !next.Before(current.time) {
			return // already scheduled earlier
		}
	}
	t := s.simTime
	if ok && next.After(s.simTime) {
		t = next
	}
	s.seq++
	e := event{time: t, pinID: p.id, seq: s.seq}
	heap.Push(&s.queue, e)
	s.scheduled[p.id] = e
}

func (s *simulation) evaluatePin(p *pin) {
	if !p.canActivate() && !p.isActive {
		return
	}
	products := p.run(s.simTime)
	if p.isFactory() {
		s.routeInput(p)
	}
	if p.isActive || p.canActivate() {
		s.schedulePin(p)
	}
	if len(products) == 0 {
		return
	}
	s.routeOutput(p, products)
}

// routeInput pulls inputs for a factory from storages routed to it.
func (s *simulation) routeInput(destination *pin) {
	for _, r := range s.routes {
		if r.destinationID != destination.id {
			continue
		}
		source := s.pins[r.sourceID]
		if !source.isStorage() || len(source.contents) == 0 {
			continue
		}
		s.transfer(source, r, source.contents, -1)
	}
}

// routeOutput distributes commodities from a pin along its routes.
// Factories are served first, then storages.
func (s *simulation) routeOutput(source *pin, commodities map[int64]int64) {
	received := make(map[int64]map[int64]int64)
	var factoryRoutes, storageRoutes []sortedRoute
	for _, r := range s.routes {
		if r.sourceID != source.id {
			continue
		}
		if _, ok := commodities[r.typeID]; !ok {
			continue
		}
		destination := s.pins[r.destinationID]
		if destination.isFactory() {
			factoryRoutes = append(factoryRoutes, sortedRoute{route: r, key: destination.inputBufferState()})
		} else {
			storageRoutes = append(storageRoutes, sortedRoute{route: r, key: s.freeSpace(destination)})
		}
	}
	sortRoutes := func(a, b sortedRoute) int {
		return cmp.Or(cmp.Compare(a.key, b.key), cmp.Compare(a.id, b.id))
	}
	slices.SortStableFunc(factoryRoutes, sortRoutes)
	slices.SortStableFunc(storageRoutes, sortRoutes)
outer:
	for i, routes := range [][]sortedRoute{factoryRoutes, storageRoutes} {
		isStorageRoutes := i == 1
		for j, r := range routes {
			maxAmount := int64(-1)
			if isStorageRoutes {
				remaining := len(routes) - j
				maxAmount = int64(math.Ceil(float64(commodities[r.typeID]) / float64(remaining)))
			}
			moved := s.transfer(source, r.route, commodities, maxAmount)
			if moved > 0 {
				commodities[r.typeID] -= moved
				if commodities[r.typeID] <= 0 {
					delete(commodities, r.typeID)
				}
				if received[r.destinationID] == nil {
					received[r.destinationID] = make(map[int64]int64)
				}
				received[r.destinationID][r.typeID] += moved
			}
			if len(commodities) == 0 {
				break outer
			}
		}
	}
	for _, id := range slices.Sorted(maps.Keys(received)) {
		destination := s.pins[id]
		if destination.isFactory() {
			s.schedulePin(destination)
		}
		// storages forward what they received from producers one hop
		if !source.isStorage() && destination.isStorage() {
			s.routeOutput(destination, received[id])
		}
	}
}

// transfer moves commodities along a route and returns the amount moved.
// A negative maxAmount means no limit.
func (s *simulation) transfer(source *pin, r route, commodities map[int64]int64, maxAmount int64) int64 {
	available, ok := commodities[r.typeID]
	if !ok {
		return 0
	}
	amount := min(available, r.quantity)
	if maxAmount >= 0 {
		amount = min(amount, maxAmount)
	}
	if amount <= 0 {
		return 0
	}
	moved := s.addCommodity(s.pins[r.destinationID], r.typeID, amount)
	if source.isStorage() {
		source.removeCommodity(r.typeID, moved)
	}
	return moved
}

func (s *simulation) addCommodity(p *pin, typeID, quantity int64) int64 {
	if p.kind == kindExtractor {
		return 0
	}
	n := s.canAccept(p, typeID, quantity)
	if n < 1 {
		return 0
	}
	p.contents[typeID] += n
	if p.isFactory() {
		p.hasReceivedInputs = true
	}
	return n
}

func (s *simulation) canAccept(p *pin, typeID, quantity int64) int64 {
	switch {
	case p.isFactory():
		demand, ok := p.demands[typeID]
		if !ok {
			return 0
		}
		return min(demand-p.contents[typeID], quantity)
	case p.isStorage():
		volume := s.volumes[typeID]
		if volume <= 0 {
			return quantity
		}
		remaining := s.freeSpace(p)
		if volume*float64(quantity) > remaining {
			return int64(max(0, remaining) / volume)
		}
		return quantity
	}
	return 0
}

func (s *simulation) usedVolume(p *pin) float64 {
	var v float64
	// sorted, because the sum of floats depends on the order
	for _, typeID := range slices.Sorted(maps.Keys(p.contents)) {
		v += s.volumes[typeID] * float64(p.contents[typeID])
	}
	return v
}

func (s *simulation) freeSpace(p *pin) float64 {
	return p.capacity - s.usedVolume(p)
}

// colonyStatus returns the status of the colony and of all its pins at now.
func (s *simulation) colonyStatus(now time.Time) (app.ColonyStatus, map[int64]app.PinStatus) {
	pins := make(map[int64]app.PinStatus, len(s.pins))
	var hasNotSetup, needsAttention, isExtracting, isProducing bool
	for id, p := range s.pins {
		ps := s.pinStatus(p, now)
		pins[id] = ps
		switch ps {
		case app.PinNotSetup, app.PinInputNotRouted, app.PinOutputNotRouted:
			hasNotSetup = true
		case app.PinExtractorExpired, app.PinExtractorInactive, app.PinStorageFull:
			needsAttention = true
		case app.PinExtracting:
			isExtracting = true
		case app.PinProducing:
			isProducing = true
		}
	}
	var status app.ColonyStatus
	switch {
	case hasNotSetup:
		status = app.ColonyNotSetup
	case needsAttention:
		status = app.ColonyNeedsAttention
	case isExtracting:
		status = app.ColonyExtracting
	case isProducing:
		status = app.ColonyProducing
	default:
		status = app.ColonyIdle
	}
	return status, pins
}

func (s *simulation) pinStatus(p *pin, now time.Time) app.PinStatus {
	switch p.kind {
	case kindExtractor:
		if !p.isExtractorSetup() {
			return app.PinNotSetup
		}
		if !p.expiryTime.After(now) {
			return app.PinExtractorExpired
		}
		if x := s.routingStatus(p); x != app.PinStatusUndefined {
			return x
		}
		if p.isActive {
			return app.PinExtracting
		}
		return app.PinExtractorInactive
	case kindFactory:
		if p.schematic == nil {
			return app.PinNotSetup
		}
		if x := s.routingStatus(p); x != app.PinStatusUndefined {
			return x
		}
		if p.isActive {
			return app.PinProducing
		}
		return app.PinFactoryIdle
	}
	free := max(0, s.freeSpace(p))
	for _, r := range s.routes {
		if r.destinationID == p.id && s.volumes[r.typeID]*float64(r.quantity) > free {
			return app.PinStorageFull
		}
	}
	return app.PinStatic
}

// routingStatus returns a status when inputs or outputs of a pin are not routed.
// Returns undefined when the pin is properly routed.
func (s *simulation) routingStatus(p *pin) app.PinStatus {
	if p.isFactory() {
		incoming := make(map[int64]bool)
		for _, r := range s.routes {
			if r.destinationID == p.id {
				incoming[r.typeID] = true
			}
		}
		for typeID := range p.demands {
			if !incoming[typeID] {
				return app.PinInputNotRouted
			}
		}
	}
	for _, r := range s.routes {
		if r.sourceID == p.id {
			return app.PinStatusUndefined
		}
	}
	return app.PinOutputNotRouted
}

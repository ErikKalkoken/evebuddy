// Package colonysim estimates the current state of a PI colony
// by simulating it forward in time from its last ESI snapshot.
//
// The simulation is a Go re-implementation of the colony simulation
// in RIFT Intel Fusion Tool by Nohus (https://gitlab.com/rift-intel-fusion-tool/rift-intel-fusion-tool),
// used with the author's permission.
package colonysim

import (
	"cmp"
	"container/heap"
	"log/slog"
	"maps"
	"math"
	"slices"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

const maxEvents = 1_000_000 // guard against runaway simulations

// RunResult reports why a simulation run stopped.
type RunResult uint

const (
	RunCompleted RunResult = iota // reached the given time
	RunWorkEnded                  // the colony stopped working
	RunAborted                    // exceeded the event limit
)

type route struct {
	id            int64
	sourceID      int64
	destinationID int64
	typeID        int64
	quantity      int64
}

type event struct {
	time  time.Time
	pinID int64
	seq   uint64 // also breaks ties in insertion order
}

// Simulation is a simulation of a PI colony.
type Simulation struct {
	pins    map[int64]*pin
	pinIDs  []int64 // sorted for deterministic iteration
	routes  []route // sorted by ID
	simTime time.Time
	volumes map[int64]float64 // m3 per unit by type ID

	queue     eventQueue
	scheduled map[int64]event // currently valid event by pin ID
	seq       uint64
}

// New returns a new simulation of a colony, starting at the time of its last snapshot.
func New(cp *app.CharacterPlanet) *Simulation {
	s := &Simulation{
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

// Time returns the current simulation time.
func (s *Simulation) Time() time.Time {
	return s.simTime
}

// RunUntil advances the simulation to t.
// It reports whether the simulation completed normally.
func (s *Simulation) RunUntil(t time.Time) bool {
	if !t.After(s.simTime) {
		return true
	}
	_, r := s.run(t, false)
	return r != RunAborted
}

// RunUntilWorkEnds advances the simulation until the colony stops working
// and returns the time when that happened.
// It returns [RunCompleted] when the colony is still working at the horizon.
func (s *Simulation) RunUntilWorkEnds(horizon time.Time) (time.Time, RunResult) {
	return s.run(horizon, true)
}

// Forecast returns the state of the colony at the current simulation time.
func (s *Simulation) Forecast() *app.ColonyForecast {
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

// Forecast returns the estimated state of a colony at now
// including when it will stop working within [app.ColonyForecastHorizon].
func Forecast(cp *app.CharacterPlanet, now time.Time) *app.ColonyForecast {
	s := New(cp)
	if !s.RunUntil(now) {
		logAborted(cp, s.simTime)
	}
	f := s.Forecast()
	t, r := s.RunUntilWorkEnds(now.Add(app.ColonyForecastHorizon))
	switch r {
	case RunWorkEnded:
		if t.After(now) {
			f.WorkEndsAt = optional.New(t)
		}
	case RunCompleted:
		f.WorksBeyondHorizon = true
	case RunAborted:
		logAborted(cp, t)
	}
	return f
}

// logAborted logs a simulation which exceeded the event limit, which indicates a bug.
func logAborted(cp *app.CharacterPlanet, simTime time.Time) {
	var planetID int64
	if cp.EvePlanet != nil {
		planetID = cp.EvePlanet.ID
	}
	slog.Warn("Colony simulation aborted after exceeding event limit",
		"characterID", cp.CharacterID, "planetID", planetID, "simTime", simTime, "maxEvents", maxEvents)
}

// run runs the simulation until the given time or until the colony stops working.
// It returns the simulation time at the end and why it stopped.
func (s *Simulation) run(until time.Time, untilWorkEnds bool) (time.Time, RunResult) {
	s.queue = eventQueue{}
	s.scheduled = make(map[int64]event)
	for _, id := range s.pinIDs {
		if p := s.pins[id]; p.canRun(until) {
			s.schedulePin(p)
		}
	}
	for events := 0; s.queue.Len() > 0; events++ {
		if events > maxEvents {
			return s.simTime, RunAborted
		}
		e := heap.Pop(&s.queue).(event)
		if current, ok := s.scheduled[e.pinID]; !ok || current.seq != e.seq {
			continue // superseded
		}
		delete(s.scheduled, e.pinID)
		// check once all events at the current time are done
		if untilWorkEnds && e.time.After(s.simTime) {
			if status, _ := s.colonyStatus(s.simTime); !status.IsWorking() {
				return s.simTime, RunWorkEnded
			}
		}
		if e.time.After(until) {
			s.simTime = until
			return until, RunCompleted
		}
		s.simTime = e.time
		if p := s.pins[e.pinID]; p.canRun(until) {
			s.evaluatePin(p)
		}
	}
	if untilWorkEnds {
		return s.simTime, RunWorkEnded
	}
	s.simTime = until
	return until, RunCompleted
}

func (s *Simulation) schedulePin(p *pin) {
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

func (s *Simulation) evaluatePin(p *pin) {
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
func (s *Simulation) routeInput(destination *pin) {
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

type sortedRoute struct {
	route
	key float64
}

// routeOutput distributes commodities from a pin along its routes.
// Factories are served first, then storages.
func (s *Simulation) routeOutput(source *pin, commodities map[int64]int64) {
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
func (s *Simulation) transfer(source *pin, r route, commodities map[int64]int64, maxAmount int64) int64 {
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

func (s *Simulation) addCommodity(p *pin, typeID, quantity int64) int64 {
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

func (s *Simulation) canAccept(p *pin, typeID, quantity int64) int64 {
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

func (s *Simulation) usedVolume(p *pin) float64 {
	var v float64
	for typeID, amount := range p.contents {
		v += s.volumes[typeID] * float64(amount)
	}
	return v
}

func (s *Simulation) freeSpace(p *pin) float64 {
	return p.capacity - s.usedVolume(p)
}

// colonyStatus returns the status of the colony and of all its pins at now.
func (s *Simulation) colonyStatus(now time.Time) (app.ColonyStatus, map[int64]app.PinStatus) {
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

func (s *Simulation) pinStatus(p *pin, now time.Time) app.PinStatus {
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
func (s *Simulation) routingStatus(p *pin) app.PinStatus {
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

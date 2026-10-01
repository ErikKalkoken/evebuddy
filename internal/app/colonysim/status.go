package colonysim

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

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

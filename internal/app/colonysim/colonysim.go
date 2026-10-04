// Package colonysim estimates the current state of a PI colony
// by simulating it forward in time from its last ESI snapshot.
//
// The simulation is a Go re-implementation of the colony simulation
// in RIFT Intel Fusion Tool by Nohus (https://gitlab.com/rift-intel-fusion-tool/rift-intel-fusion-tool),
// used with the author's permission.
package colonysim

import (
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// Forecast returns the estimated state of a colony at now
// including when it will stop working within [app.ColonyForecastHorizon]
// and until when the forecast stays the same.
func Forecast(cp *app.CharacterPlanet, now time.Time) *app.ColonyForecast {
	s := newSimulation(cp)
	ok := s.runUntil(now)
	if !ok {
		logAborted(cp, s.simTime)
	}
	f := s.forecast()
	validUntil, hasChange := s.nextChange()
	if !ok {
		validUntil, hasChange = now, true // incomplete, so never reuse
	}
	t, r := s.runUntilWorkEnds(now.Add(app.ColonyForecastHorizon))
	switch r {
	case runWorkEnded:
		if t.After(now) {
			f.WorkEndsAt = optional.New(t)
			// the work end is only reported while it is in the future
			if !hasChange || t.Before(validUntil) {
				validUntil, hasChange = t, true
			}
		}
	case runCompleted:
		f.WorksBeyondHorizon = true
	case runAborted:
		logAborted(cp, t)
	}
	if hasChange {
		f.ValidUntil = optional.New(validUntil)
	}
	return f
}

// WorkEndsAt returns when the colony stops working after its snapshot,
// if it was working at the snapshot and stops before until.
// Unlike [Forecast] it is not limited by [app.ColonyForecastHorizon].
func WorkEndsAt(cp *app.CharacterPlanet, until time.Time) optional.Optional[time.Time] {
	s := newSimulation(cp)
	t, r := s.runUntilWorkEnds(until)
	switch r {
	case runWorkEnded:
		if t.After(cp.LastUpdate) {
			return optional.New(t)
		}
	case runAborted:
		logAborted(cp, t)
	}
	return optional.Optional[time.Time]{}
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

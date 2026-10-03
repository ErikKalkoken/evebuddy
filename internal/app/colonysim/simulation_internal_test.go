package colonysim

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestSimulation_Run(t *testing.T) {
	cp := &app.CharacterPlanet{
		LastUpdate: t0,
		Pins: []*app.PlanetPin{
			newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour)),
			newStorage(2, app.EveGroupStorageFacilities, 12_000),
		},
		Routes: []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
	}
	t.Run("should start at the snapshot", func(t *testing.T) {
		s := newSimulation(cp)
		assert.Equal(t, t0, s.simTime)
		f := s.forecast()
		assert.Equal(t, t0, f.Time)
		assert.Empty(t, f.Pins[2].Contents)
	})
	t.Run("should advance time when running", func(t *testing.T) {
		s := newSimulation(cp)
		assert.True(t, s.runUntil(t0.Add(65*time.Minute)))
		assert.Equal(t, t0.Add(65*time.Minute), s.simTime)
		assert.Equal(t, t0.Add(65*time.Minute), s.forecast().Time)
	})
	t.Run("should do nothing when running until current time", func(t *testing.T) {
		s := newSimulation(cp)
		s.runUntil(t0.Add(time.Hour))
		assert.True(t, s.runUntil(t0.Add(time.Hour)))
		assert.Equal(t, t0.Add(time.Hour), s.simTime)
	})
	t.Run("should continue from previous run", func(t *testing.T) {
		s1 := newSimulation(cp)
		s1.runUntil(t0.Add(time.Hour))
		s1.runUntil(t0.Add(2 * time.Hour))
		s2 := newSimulation(cp)
		s2.runUntil(t0.Add(2 * time.Hour))
		assert.Equal(t, s2.forecast(), s1.forecast())
	})
	t.Run("should return snapshot not affected by later runs", func(t *testing.T) {
		s := newSimulation(cp)
		s.runUntil(t0.Add(time.Hour))
		f := s.forecast()
		s.runUntil(t0.Add(2 * time.Hour))
		assert.Equal(t, map[int64]int64{typeAqueousLiquids: 2467 + 2086}, f.Pins[2].Contents)
	})
	t.Run("should not go back in time", func(t *testing.T) {
		s := newSimulation(cp)
		s.runUntil(t0.Add(-time.Hour))
		assert.Equal(t, t0, s.simTime)
	})
}

func TestSimulation_RunUntilWorkEnds(t *testing.T) {
	newPlanet := func(expiry time.Time) *app.CharacterPlanet {
		return &app.CharacterPlanet{
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, expiry),
				newStorage(2, app.EveGroupStorageFacilities, 1_000_000),
			},
			Routes: []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
		}
	}
	t.Run("should return completed when still working at the horizon", func(t *testing.T) {
		s := newSimulation(newPlanet(t0.Add(48 * time.Hour)))
		horizon := t0.Add(24 * time.Hour)
		got, r := s.runUntilWorkEnds(horizon)
		assert.Equal(t, runCompleted, r)
		assert.Equal(t, horizon, got)
	})
	t.Run("should return work ended when colony stops before the horizon", func(t *testing.T) {
		s := newSimulation(newPlanet(t0.Add(4 * time.Hour)))
		got, r := s.runUntilWorkEnds(t0.Add(24 * time.Hour))
		assert.Equal(t, runWorkEnded, r)
		assert.Equal(t, t0.Add(4*time.Hour), got)
	})
	t.Run("should return work ended when colony is not working", func(t *testing.T) {
		s := newSimulation(newPlanet(t0.Add(4 * time.Hour)))
		s.runUntil(t0.Add(5 * time.Hour))
		got, r := s.runUntilWorkEnds(t0.Add(24 * time.Hour))
		assert.Equal(t, runWorkEnded, r)
		assert.Equal(t, t0.Add(5*time.Hour), got)
	})
	t.Run("should return work ended when colony stops within one cycle of the horizon", func(t *testing.T) {
		cp := &app.CharacterPlanet{
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newStorage(1, app.EveGroupStorageFacilities, 12_000, &app.PlanetPinContent{Type: aqueousLiquids, Amount: 9000}),
				newFactory(2, schematicWater),
				newStorage(3, app.EveGroupSpaceports, 10_000),
			},
			Routes: []*app.PlanetRoute{
				newRoute(1, 1, 2, aqueousLiquids, 3000),
				newRoute(2, 2, 3, water, 20),
			},
		}
		s := newSimulation(cp)
		s.runUntil(t0.Add(45 * time.Minute))
		got, r := s.runUntilWorkEnds(t0.Add(100 * time.Minute))
		assert.Equal(t, runWorkEnded, r)
		assert.Equal(t, t0.Add(90*time.Minute), got)
	})
}

func TestSimulation_IncompleteExtractorDoesNotAbort(t *testing.T) {
	cases := map[string]func(p *app.PlanetPin){
		"without cycle time": func(p *app.PlanetPin) { p.ExtractorCycleTime = optional.Optional[time.Duration]{} },
		"without quantity":   func(p *app.PlanetPin) { p.ExtractorQtyPerCycle = optional.Optional[int64]{} },
		"without product":    func(p *app.PlanetPin) { p.ExtractorProductType = optional.Optional[*app.EveType]{} },
		"without program": func(p *app.PlanetPin) {
			p.InstallTime = optional.Optional[time.Time]{}
			p.ExpiryTime = optional.Optional[time.Time]{}
		},
	}
	for name, modify := range cases {
		t.Run(name, func(t *testing.T) {
			p := newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour))
			modify(p)
			cp := &app.CharacterPlanet{
				LastUpdate: t0,
				Pins:       []*app.PlanetPin{p, newStorage(2, app.EveGroupStorageFacilities, 12_000)},
				Routes:     []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
			}
			assert.True(t, newSimulation(cp).runUntil(t0.Add(time.Hour)))
		})
	}
}

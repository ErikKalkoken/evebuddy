package colonysim

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestSimulation_PinStatus(t *testing.T) {
	expiry := t0.Add(4 * time.Hour)
	toStorage := newRoute(1, 1, 2, aqueousLiquids, 10_000)
	fromStorage := newRoute(1, 2, 1, aqueousLiquids, 3000)
	toLaunchpad := newRoute(2, 1, 3, water, 20)
	cases := []struct {
		name   string
		pin    func() *app.PlanetPin // pin 1, next to storage 2 and launchpad 3
		routes []*app.PlanetRoute
		want   app.PinStatus
	}{
		{
			name: "extractor without product",
			pin: func() *app.PlanetPin {
				p := newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, expiry)
				p.ExtractorProductType = optional.Optional[*app.EveType]{}
				return p
			},
			routes: []*app.PlanetRoute{toStorage},
			want:   app.PinNotSetup,
		},
		{
			name:   "extractor without cycle time",
			pin:    func() *app.PlanetPin { return newExtractor(1, aqueousLiquids, 1081, 0, t0, expiry) },
			routes: []*app.PlanetRoute{toStorage},
			want:   app.PinNotSetup,
		},
		{
			name:   "extractor without quantity",
			pin:    func() *app.PlanetPin { return newExtractor(1, aqueousLiquids, 0, 30*time.Minute, t0, expiry) },
			routes: []*app.PlanetRoute{toStorage},
			want:   app.PinNotSetup,
		},
		{
			name: "extractor without program",
			pin: func() *app.PlanetPin {
				p := newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, expiry)
				p.InstallTime = optional.Optional[time.Time]{}
				p.ExpiryTime = optional.Optional[time.Time]{}
				return p
			},
			routes: []*app.PlanetRoute{toStorage},
			want:   app.PinNotSetup,
		},
		{
			name:   "extractor with output not routed",
			pin:    func() *app.PlanetPin { return newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, expiry) },
			routes: nil,
			want:   app.PinOutputNotRouted,
		},
		{
			name: "extractor without last cycle start",
			pin: func() *app.PlanetPin {
				p := newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, expiry)
				p.LastCycleStart = optional.Optional[time.Time]{}
				return p
			},
			routes: []*app.PlanetRoute{toStorage},
			want:   app.PinExtractorInactive,
		},
		{
			name: "factory without schematic",
			pin: func() *app.PlanetPin {
				p := newFactory(1, schematicWater)
				p.Schematic = optional.Optional[*app.EveSchematic]{}
				return p
			},
			routes: []*app.PlanetRoute{fromStorage, toLaunchpad},
			want:   app.PinNotSetup,
		},
		{
			name:   "factory with schematic unknown to the SDE",
			pin:    func() *app.PlanetPin { return newFactory(1, 999_999) },
			routes: []*app.PlanetRoute{fromStorage, toLaunchpad},
			want:   app.PinNotSetup,
		},
		{
			name: "factory with schematic from factory fallback",
			pin: func() *app.PlanetPin {
				p := newFactory(1, schematicWater)
				p.Schematic = optional.Optional[*app.EveSchematic]{}
				p.FactorySchematic = optional.New(&app.EveSchematic{ID: schematicWater})
				return p
			},
			routes: []*app.PlanetRoute{fromStorage, toLaunchpad},
			want:   app.PinFactoryIdle,
		},
		{
			name:   "factory with output not routed",
			pin:    func() *app.PlanetPin { return newFactory(1, schematicWater) },
			routes: []*app.PlanetRoute{fromStorage},
			want:   app.PinOutputNotRouted,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cp := &app.CharacterPlanet{
				LastUpdate: t0,
				Pins: []*app.PlanetPin{
					tc.pin(),
					newStorage(2, app.EveGroupStorageFacilities, 12_000),
					newStorage(3, app.EveGroupSpaceports, 10_000),
				},
				Routes: tc.routes,
			}
			assert.True(t, New(cp).RunUntil(t0.Add(time.Minute)), "not aborted")
			f := Forecast(cp, t0.Add(time.Minute))
			assert.Equal(t, tc.want, f.Pins[1].Status)
		})
	}
}

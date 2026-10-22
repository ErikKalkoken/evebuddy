package infoviewer

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func entityItemIDs(w *entityList) []int64 {
	return xslices.Map(w.items, func(x entityItem) int64 {
		return x.id
	})
}

func TestSolarSystemInfo_update(t *testing.T) {
	t.Run("should show solar system", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{SecurityStatus: 0.85})
		factory.GetOrCreateEveEntity(*system.ToEveEntity())
		factory.GetOrCreateEveEntity(*system.Constellation.ToEveEntity())
		station := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityStation})
		structure := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{
			SolarSystemID: optional.New(system.ID),
		})
		planet := factory.CreateEvePlanet(storage.CreateEvePlanetParams{SolarSystemID: system.ID})
		otherSystem := factory.CreateEveSolarSystem()
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/universe/systems/%d", system.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellation_id": system.Constellation.ID,
				"name":             system.Name,
				"planets":          []map[string]any{{"planet_id": planet.ID}},
				"position":         map[string]any{"x": 1, "y": 2, "z": 3},
				"security_status":  0.85,
				"star_id":          40000040,
				"stargates":        []int64{50000342},
				"stations":         []int64{station.ID},
				"system_id":        system.ID,
			}))
		httpmock.RegisterResponder("GET", "https://esi.evetech.net/universe/stargates/50000342",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"destination": map[string]any{"stargate_id": 50000343, "system_id": otherSystem.ID},
				"name":        "Stargate",
				"position":    map[string]any{"x": 1, "y": 2, "z": 3},
				"stargate_id": 50000342,
				"system_id":   system.ID,
				"type_id":     29624,
			}))
		httpmock.RegisterResponder("GET", "https://esi.evetech.net/universe/stars/40000040",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"age":             9398686722,
				"luminosity":      0.06615000218153,
				"name":            "Star",
				"radius":          346600000,
				"solar_system_id": system.ID,
				"spectral_class":  "K2 V",
				"temperature":     3953,
				"type_id":         3800,
			}))
		a := newSolarSystemInfo(New(u), system.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, system.Name, a.name.Text)
		assert.Equal(t, system.Constellation.Name, a.constellation.Text)
		assert.Equal(t, system.Constellation.Region.Name, a.region.Text)
		assert.Equal(t, system.SecurityStatusDisplay(), a.security.Text)
		assert.Equal(t, []int64{station.ID}, entityItemIDs(a.stations))
		assert.Equal(t, []int64{structure.ID}, entityItemIDs(a.structures))
		assert.Equal(t, []int64{otherSystem.ID}, entityItemIDs(a.stargates))
		assert.Equal(t, []int64{planet.ID}, entityItemIDs(a.planets))
	})
	t.Run("should show solar system when star type can not be fetched", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		system := factory.CreateEveSolarSystem()
		factory.GetOrCreateEveEntity(*system.ToEveEntity())
		factory.GetOrCreateEveEntity(*system.Constellation.ToEveEntity())
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/universe/systems/%d", system.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellation_id": system.Constellation.ID,
				"name":             system.Name,
				"position":         map[string]any{"x": 1, "y": 2, "z": 3},
				"security_status":  system.SecurityStatus,
				"star_id":          40000040,
				"system_id":        system.ID,
			}))
		httpmock.RegisterResponder("GET", "https://esi.evetech.net/universe/stars/40000040",
			httpmock.NewStringResponder(404, `{"error":"not found"}`))
		a := newSolarSystemInfo(New(u), system.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, system.Name, a.name.Text)
	})
	t.Run("should report error when solar system can not be fetched", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", `=~^https://esi\.evetech\.net/universe/systems/\d+`,
			httpmock.NewStringResponder(404, `{"error":"not found"}`))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		system := factory.CreateEveSolarSystem()
		a := newSolarSystemInfo(New(u), system.ID)
		// when
		err := a.update(t.Context())
		// then
		assert.Error(t, err)
		assert.Equal(t, "Loading...", a.name.Text)
	})
}

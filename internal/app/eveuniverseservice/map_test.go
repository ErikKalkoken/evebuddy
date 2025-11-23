package eveuniverseservice_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetOrCreateEveRegionESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing region", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveRegion(storage.CreateEveRegionParams{ID: 6})
		// when
		x1, err := s.GetOrCreateRegionESI(ctx, 6)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(6), x1.ID)
		}
	})
	t.Run("should fetch region from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/regions/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellations": []int{20000302, 20000303},
				"description":    "It has long been an established fact of civilization...",
				"name":           "Metropolis",
				"region_id":      10000042,
			}),
		)
		// when
		x1, err := s.GetOrCreateRegionESI(ctx, 10000042)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(10000042), x1.ID)
			assert.Equal(t, "Metropolis", x1.Name)
			x2, err := st.GetEveRegion(ctx, 10000042)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEveConstellationESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing constellation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveConstellation(storage.CreateEveConstellationParams{ID: 25})
		// when
		x1, err := s.GetOrCreateConstellationESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(25), x1.ID)
		}
	})
	t.Run("should fetch constellation from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveRegion(storage.CreateEveRegionParams{ID: 10000001})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/constellations/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellation_id": 20000009,
				"name":             "Mekashtad",
				"position": map[string]any{
					"x": 67796138757472320,
					"y": -70591121348560960,
					"z": -59587016159270070,
				},
				"region_id": 10000001,
				"systems":   []int{20000302, 20000303},
			}),
		)
		// when
		x1, err := s.GetOrCreateConstellationESI(ctx, 20000009)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(20000009), x1.ID)
			assert.Equal(t, "Mekashtad", x1.Name)
			assert.Equal(t, int32(10000001), x1.Region.ID)
			x2, err := st.GetEveConstellation(ctx, 20000009)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEveSolarSystemESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing solar system", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 587})
		// when
		x1, err := s.GetOrCreateSolarSystemESI(ctx, 587)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(587), x1.ID)
		}
	})
	t.Run("should fetch solar system from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveConstellation(storage.CreateEveConstellationParams{ID: 20000001})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/systems/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellation_id": 20000001,
				"name":             "Akpivem",
				"planets": []map[string]any{
					{
						"moons":     []int{40000042},
						"planet_id": 40000041,
					},
					{
						"planet_id": 40000043,
					},
				},
				"position": map[string]any{
					"x": -91174141133075340,
					"y": 43938227486247170,
					"z": -56482824383339900,
				},
				"security_class":  "B",
				"security_status": 0.8462923765182495,
				"star_id":         40000040,
				"stargates":       []int{50000342},
				"system_id":       30000003,
			}),
		)
		// when
		x1, err := s.GetOrCreateSolarSystemESI(ctx, 30000003)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(30000003), x1.ID)
			assert.Equal(t, "Akpivem", x1.Name)
			assert.Equal(t, int32(20000001), x1.Constellation.ID)
			x2, err := st.GetEveSolarSystem(ctx, 30000003)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should fetch solar system from ESI and create it (integration)", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/regions/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellations": []int{
					20000001,
					20000002,
					20000003,
					20000004,
					20000005,
					20000006,
					20000007,
					20000008,
					20000009,
					20000010,
					20000011,
					20000012,
					20000013,
					20000014,
					20000015,
					20000016,
				},
				"description": "The Derelik region, sovereign seat of the Ammatar Mandate, became the shield to the Amarrian flank in the wake of the Minmatar Rebellion. Derelik witnessed many hostile exchanges between the Amarr and rebel forces as the latter tried to push deeper into the territory of their former masters. Having held their ground, thanks in no small part to the Ammatars' military efforts, the Amarr awarded the Ammatar with their own province. However, this portion of space shared borders with the newly forming Minmatar Republic as well as the Empire, and thus came to be situated in a dark recess surrounded by hostiles. \n\nGiven the lack of safe routes elsewhere, the local economies of this region were dependent on trade with the Amarr as their primary means of survival. The Ammatar persevered over many decades of economic stagnation and limited trade partners, and their determination has in recent decades been rewarded with an increase in economic prosperity. This harsh trail is a point of pride for all who call themselves Ammatar, and it has bolstered their faith in the Amarrian way to no end.",
				"name":        "Derelik",
				"region_id":   10000001,
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/constellations/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellation_id": 20000001,
				"name":             "San Matar",
				"position": map[string]any{
					"x": -94046559700991340,
					"y": 49520153153798850,
					"z": -42738731818401970,
				},
				"region_id": 10000001,
				"systems": []int{
					30000001,
					30000002,
					30000003,
					30000004,
					30000005,
					30000006,
					30000007,
					30000008,
				},
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/systems/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellation_id": 20000001,
				"name":             "Akpivem",
				"planets": []map[string]any{
					{
						"moons":     []int{40000042},
						"planet_id": 40000041,
					},
					{
						"planet_id": 40000043,
					},
				},
				"position": map[string]any{
					"x": -91174141133075340,
					"y": 43938227486247170,
					"z": -56482824383339900,
				},
				"security_class":  "B",
				"security_status": 0.8462923765182495,
				"star_id":         40000040,
				"stargates": []int{
					50000342,
				},
				"system_id": 30000003,
			}),
		)
		// when
		x1, err := s.GetOrCreateSolarSystemESI(ctx, 30000003)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(30000003), x1.ID)
			assert.Equal(t, "Akpivem", x1.Name)
			assert.Equal(t, int32(20000001), x1.Constellation.ID)
			x2, err := st.GetEveSolarSystem(ctx, 30000003)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEvePlanetESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing planet", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 25})
		// when
		x1, err := s.GetOrCreatePlanetESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(25), x1.ID)
		}
	})
	t.Run("should fetch planet from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		solarSystem := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000003})
		type_ := factory.CreateEveType(storage.CreateEveTypeParams{ID: 13})
		data := map[string]any{
			"name":      "Akpivem III",
			"planet_id": 40000046,
			"position": map[string]any{
				"x": -189226344497,
				"y": 9901605317,
				"z": -254852632979,
			},
			"system_id": 30000003,
			"type_id":   13,
		}
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/planets/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		x1, err := s.GetOrCreatePlanetESI(ctx, 40000046)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(40000046), x1.ID)
			assert.Equal(t, "Akpivem III", x1.Name)
			assert.Equal(t, solarSystem, x1.SolarSystem)
			assert.Equal(t, type_, x1.Type)
		}
	})
}

func TestGetOrCreateEveMoonESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing moon", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveMoon(storage.CreateEveMoonParams{ID: 25})
		// when
		x1, err := s.GetOrCreateMoonESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(25), x1.ID)
		}
	})
	t.Run("should fetch moon from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		solarSystem := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000003})
		data := map[string]any{
			"moon_id": 40000042,
			"name":    "Akpivem I - Moon 1",
			"position": map[string]any{
				"x": 58605102008,
				"y": -3066616285,
				"z": -55193617920,
			},
			"system_id": 30000003,
		}
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/moons/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		x1, err := s.GetOrCreateMoonESI(ctx, 40000042)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(40000042), x1.ID)
			assert.Equal(t, "Akpivem I - Moon 1", x1.Name)
			assert.Equal(t, solarSystem, x1.SolarSystem)
		}
	})
}

func TestFetchRoute(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return route when valid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		s1 := factory.CreateEveSolarSystem()
		s2 := factory.CreateEveSolarSystem()
		s3 := factory.CreateEveSolarSystem()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/route/\d+/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, []int32{s1.ID, s2.ID, s3.ID}),
		)
		// when
		x, err := s.FetchRoute(ctx, app.EveRouteHeader{
			Destination: s3,
			Origin:      s1,
			Preference:  app.RouteShortest,
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, []*app.EveSolarSystem{s1, s2, s3}, x)
		}
	})
	t.Run("should return short route when origin and dest the same", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		o := factory.CreateEveSolarSystem()
		// when
		x, err := s.FetchRoute(ctx, app.EveRouteHeader{
			Destination: o,
			Origin:      o,
			Preference:  app.RouteShortest,
		})
		// then
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, []*app.EveSolarSystem{o}, x)
			assert.Equal(t, 0, httpmock.GetTotalCallCount())
		}
	})
	t.Run("should return invalid route when origin in WH space", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		orig := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 31000001})
		dest := factory.CreateEveSolarSystem()
		// when
		x, err := s.FetchRoute(ctx, app.EveRouteHeader{
			Destination: dest,
			Origin:      orig,
			Preference:  app.RouteShortest,
		})
		// then
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, []*app.EveSolarSystem{}, x)
		}
	})
	t.Run("should return error when called with invalid systems", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		// when
		_, err := s.FetchRoute(ctx, app.EveRouteHeader{
			Preference: app.RouteShortest,
		})
		// then
		if assert.ErrorIs(t, err, app.ErrInvalid) {
			assert.Equal(t, 0, httpmock.GetTotalCallCount())
		}
	})
	t.Run("should return error when called with invalid systems", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		x := factory.CreateEveSolarSystem()
		// when
		_, err := s.FetchRoute(ctx, app.EveRouteHeader{
			Origin:     x,
			Preference: app.RouteShortest,
		})
		// then
		if assert.ErrorIs(t, err, app.ErrInvalid) {
			assert.Equal(t, 0, httpmock.GetTotalCallCount())
		}
	})
	t.Run("return empty slice when no route found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		s1 := factory.CreateEveSolarSystem()
		s2 := factory.CreateEveSolarSystem()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/route/\d+/\d+/`,
			httpmock.NewJsonResponderOrPanic(404, map[string]string{"error": "no route found"}),
		)
		// when
		got, err := s.FetchRoute(ctx, app.EveRouteHeader{
			Destination: s2,
			Origin:      s1,
			Preference:  app.RouteShortest,
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, []*app.EveSolarSystem{}, got)
		}
	})
}

func TestFetchRoutes(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return routes", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		a1 := factory.CreateEveSolarSystem()
		a2 := factory.CreateEveSolarSystem()
		a3 := factory.CreateEveSolarSystem()
		b1 := factory.CreateEveSolarSystem()
		b2 := factory.CreateEveSolarSystem()
		b3 := factory.CreateEveSolarSystem()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/route/%d/%d/", a1.ID, a3.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{a1.ID, a2.ID, a3.ID}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/route/%d/%d/", b1.ID, b3.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{b1.ID, b2.ID, b3.ID}),
		)
		// when
		r1 := app.EveRouteHeader{
			Destination: a3,
			Origin:      a1,
			Preference:  app.RouteShortest,
		}
		r2 := app.EveRouteHeader{
			Destination: b3,
			Origin:      b1,
			Preference:  app.RouteShortest,
		}
		got, err := s.FetchRoutes(ctx, []app.EveRouteHeader{r1, r2})
		// then
		if assert.NoError(t, err) && assert.Len(t, got, 2) {
			assert.Equal(t, []*app.EveSolarSystem{a1, a2, a3}, got[r1])
			assert.Equal(t, []*app.EveSolarSystem{b1, b2, b3}, got[r2])
		}
	})
}

func TestMembershipHistory(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return corporation membership history", func(t *testing.T) {
		// given
		s.Now = func() time.Time { return time.Date(2016, 7, 30, 20, 0, 0, 0, time.UTC) }
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c1 := factory.CreateEveEntityCorporation(app.EveEntity{ID: 90000001})
		c2 := factory.CreateEveEntityCorporation(app.EveEntity{ID: 90000002})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/corporationhistory/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"corporation_id": 90000001,
					"is_deleted":     true,
					"record_id":      500,
					"start_date":     "2016-06-26T20:00:00Z",
				},
				{
					"corporation_id": 90000002,
					"record_id":      501,
					"start_date":     "2016-07-26T20:00:00Z",
				},
			}))
		// when
		x, err := s.FetchCharacterCorporationHistory(ctx, 42)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, x, 2)
			assert.EqualValues(t, app.MembershipHistoryItem{
				Days:         4,
				IsDeleted:    false,
				Organization: c2,
				RecordID:     501,
				StartDate:    time.Date(2016, 7, 26, 20, 0, 0, 0, time.UTC),
			}, x[0])
			assert.EqualValues(t, app.MembershipHistoryItem{
				EndDate:      time.Date(2016, 7, 26, 20, 0, 0, 0, time.UTC),
				Days:         30,
				IsDeleted:    true,
				IsOldest:     true,
				Organization: c1,
				RecordID:     500,
				StartDate:    time.Date(2016, 6, 26, 20, 0, 0, 0, time.UTC),
			}, x[1])
		}
	})
	t.Run("should return alliance membership history", func(t *testing.T) {
		// given
		s.Now = func() time.Time { return time.Date(2016, 10, 30, 20, 0, 0, 0, time.UTC) }
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c1 := factory.CreateEveEntityAlliance(app.EveEntity{ID: 99000006})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/alliancehistory/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id": 99000006,
					"is_deleted":  true,
					"record_id":   23,
					"start_date":  "2016-10-25T14:46:00Z",
				},
				{
					"record_id":  1,
					"start_date": "2015-07-06T20:56:00Z",
				},
			}))
		// when
		x, err := s.FetchCorporationAllianceHistory(ctx, 42)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, x, 2)
			assert.EqualValues(t, app.MembershipHistoryItem{
				Days:         5,
				IsDeleted:    true,
				Organization: c1,
				RecordID:     23,
				StartDate:    time.Date(2016, 10, 25, 14, 46, 0, 0, time.UTC),
			}, x[0])
			assert.EqualValues(t, app.MembershipHistoryItem{
				EndDate:   time.Date(2016, 10, 25, 14, 46, 0, 0, time.UTC),
				Days:      476,
				IsOldest:  true,
				RecordID:  1,
				StartDate: time.Date(2015, 7, 6, 20, 56, 0, 0, time.UTC),
			}, x[1])
		}
	})
}

func TestGetStarTypeID(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	// given
	testutil.MustTruncateTables(db)
	httpmock.Reset()
	httpmock.RegisterResponder(
		"GET",
		`=~^https://esi\.evetech\.net/v\d+/universe/stars/\d+/`,
		httpmock.NewJsonResponderOrPanic(200, map[string]any{
			"age":             9398686722,
			"luminosity":      0.06615000218153,
			"name":            "BKG-Q2 - Star",
			"radius":          346600000,
			"solar_system_id": 30004333,
			"spectral_class":  "K2 V",
			"temperature":     3953,
			"type_id":         45033,
		}),
	)
	// when
	got, err := s.GetStarTypeID(ctx, 40000046)
	// then
	if assert.NoError(t, err) {
		assert.EqualValues(t, 45033, got)
	}
}

func TestGetSolarSystemInfoESI(t *testing.T) {
	// given
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	system := factory.CreateEveSolarSystem()
	constellation := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityConstellation})
	factory.CreateEveEntity(*system.EveEntity())
	station := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityStation})
	structure := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{
		SolarSystemID: optional.New(system.ID),
	})
	httpmock.RegisterResponder(
		"GET",
		`=~^https://esi\.evetech\.net/v\d+/universe/systems/\d+/`,
		httpmock.NewJsonResponderOrPanic(200, map[string]any{
			"constellation_id": constellation.ID,
			"name":             "Akpivem",
			"planets": []map[string]any{
				{
					"moons":     []int{40000042},
					"planet_id": 40000041,
				},
				{
					"planet_id":      40000043,
					"asteroid_belts": []int{40000051},
				},
			},
			"position": map[string]any{
				"x": -91174141133075340,
				"y": 43938227486247170,
				"z": -56482824383339900,
			},
			"security_class":  "B",
			"security_status": 0.8462923765182495,
			"star_id":         40000040,
			"stargates":       []int{50000342},
			"stations":        []int{int(station.ID)},
			"system_id":       system.ID,
		}),
	)
	// when
	starID, planets, stargateIDs, stations, structures, err := s.GetSolarSystemInfoESI(ctx, system.ID)
	// then
	if assert.NoError(t, err) {
		assert.EqualValues(t, 40000040, starID)
		assert.ElementsMatch(t, []app.EveSolarSystemPlanet{
			{
				PlanetID: int32(40000041),
				MoonIDs:  []int32{40000042},
			},
			{
				PlanetID:        int32(40000043),
				AsteroidBeltIDs: []int32{40000051},
			},
		},
			planets,
		)
		assert.ElementsMatch(t, []int32{50000342}, stargateIDs)
		assert.ElementsMatch(t, []*app.EveEntity{station}, stations)
		assert.ElementsMatch(t, []*app.EveLocation{structure}, structures)
	}
}

func TestGetRegionConstellationsESI(t *testing.T) {
	// given
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	c1 := factory.CreateEveEntity(app.EveEntity{
		ID:       20000302,
		Category: app.EveEntityConstellation,
	})
	c2 := factory.CreateEveEntity(app.EveEntity{
		ID:       20000303,
		Category: app.EveEntityConstellation,
	})
	httpmock.RegisterResponder(
		"GET",
		`=~^https://esi\.evetech\.net/v\d+/universe/regions/\d+/`,
		httpmock.NewJsonResponderOrPanic(200, map[string]any{
			"constellations": []int{20000302, 20000303},
			"description":    "It has long been an established fact of civilization...",
			"name":           "Metropolis",
			"region_id":      10000042,
		}),
	)
	// when
	got, err := s.GetRegionConstellationsESI(ctx, 10000042)
	// then
	if assert.NoError(t, err) {
		want := []*app.EveEntity{c1, c2}
		assert.ElementsMatch(t, want, got)
	}
}

func TestGetConstellationSolarSystemsESI(t *testing.T) {
	// given
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	s1 := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 20000302})
	s2 := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 20000303})
	httpmock.RegisterResponder(
		"GET",
		`=~^https://esi\.evetech\.net/v\d+/universe/constellations/\d+/`,
		httpmock.NewJsonResponderOrPanic(200, map[string]any{
			"constellation_id": 20000009,
			"name":             "Mekashtad",
			"position": map[string]any{
				"x": 67796138757472320,
				"y": -70591121348560960,
				"z": -59587016159270070,
			},
			"region_id": 10000001,
			"systems":   []int{20000302, 20000303},
		}),
	)
	// when
	got, err := s.GetConstellationSolarSystemsESI(ctx, 20000009)
	// then
	if assert.NoError(t, err) {
		want := []*app.EveSolarSystem{s1, s2}
		assert.ElementsMatch(t, want, got)
	}
}

func TestGetStargateSolarSystemsESI(t *testing.T) {
	// given
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000001})
	httpmock.RegisterResponder(
		"GET",
		`=~^https://esi\.evetech\.net/v\d+/universe/stargates/\d+/`,
		httpmock.NewJsonResponderOrPanic(200, map[string]any{
			"destination": map[string]int{
				"stargate_id": 50000056,
				"system_id":   30000001,
			},
			"name": "Stargate (Tanoo)",
			"position": map[string]int{
				"x": -101092761600,
				"y": 5279539200,
				"z": 1550503403520,
			},
			"stargate_id": 50000342,
			"system_id":   30000003,
			"type_id":     29624,
		}),
	)
	// when
	got, err := s.GetStargatesSolarSystemsESI(ctx, []int32{20000009})
	// then
	if assert.NoError(t, err) {
		want := []*app.EveSolarSystem{system}
		assert.ElementsMatch(t, want, got)
	}
}

func TestGetSolarSystemPlanets(t *testing.T) {
	// given
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	p1 := factory.CreateEvePlanet()
	p2 := factory.CreateEvePlanet()
	// when
	got, err := s.GetSolarSystemPlanets(ctx, []app.EveSolarSystemPlanet{
		{PlanetID: p1.ID},
		{PlanetID: p2.ID},
	})
	// then
	if assert.NoError(t, err) {
		want := []*app.EvePlanet{p1, p2}
		assert.ElementsMatch(t, want, got)
	}
}

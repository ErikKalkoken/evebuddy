package eveuniverseservice_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

const (
	stationID   = 60000277
	structureID = 1_000_000_000_009
)

func TestEveLocation(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	xesi.ActivateRateLimiterMock()
	defer xesi.DeactivateRateLimiterMock()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should create location for a station", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		owner := factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000003})
		system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000148})
		myType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 1531})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/universe/stations/%d/", stationID),
			httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
				"max_dockable_ship_volume": 50000000,
				"name":                     "Jakanerva III - Moon 15 - Prompt Delivery Storage",
				"office_rental_cost":       10000,
				"owner":                    1000003,
				"position": map[string]any{
					"x": 165632286720,
					"y": 2771804160,
					"z": -2455331266560,
				},
				"race_id":                    1,
				"reprocessing_efficiency":    0.5,
				"reprocessing_stations_take": 0.05,
				"services": []string{
					"courier-missions",
					"reprocessing-plant",
					"market",
					"repair-facilities",
					"fitting",
					"news",
					"storage",
					"insurance",
					"docking",
					"office-rental",
					"loyalty-point-store",
					"navy-offices",
				},
				"station_id": 60000277,
				"system_id":  30000148,
				"type_id":    1531,
			}),
		)
		// when
		x1, err := s.GetOrCreateLocationESI(ctx, stationID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int64(stationID), x1.ID)
			assert.Equal(t, "Jakanerva III - Moon 15 - Prompt Delivery Storage", x1.Name)
			assert.Equal(t, owner, x1.Owner)
			assert.Equal(t, system, x1.SolarSystem)
			assert.Equal(t, myType, x1.Type)
			x2, err := st.GetLocation(ctx, stationID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should create location for a solar system", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		myType := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeSolarSystem})
		system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000148})
		// when
		x1, err := s.GetOrCreateLocationESI(ctx, int64(system.ID))
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int64(system.ID), x1.ID)
			assert.Equal(t, system.Name, x1.DisplayName())
			assert.Nil(t, x1.Owner)
			assert.Equal(t, system, x1.SolarSystem)
			assert.Equal(t, myType, x1.Type)
			x2, err := st.GetLocation(ctx, int64(system.ID))
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("can create unknown location", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		myType := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeSolarSystem})
		// when
		x1, err := s.GetOrCreateLocationESI(ctx, 888)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, myType, x1.Type)
			x2, err := st.GetLocation(ctx, x1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("can create asset safety location", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		myType := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeAssetSafetyWrap})
		// when
		x1, err := s.GetOrCreateLocationESI(ctx, 2004)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, myType, x1.Type)
			x2, err := st.GetLocation(ctx, x1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestLocationStructures(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	xesi.ActivateRateLimiterMock()
	defer xesi.DeactivateRateLimiterMock()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	t.Run("should return existing structure", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: structureID, Name: "Alpha"})
		// when
		x, err := s.GetOrCreateLocationESI(context.Background(), structureID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "Alpha", x.Name)
		}
	})
	t.Run("should fetch structure from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		owner := factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000142})
		myType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 99})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/structures/\d+/`,
			httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
				"name":            "V-3YG7 VI - The Capital",
				"owner_id":        109299958,
				"solar_system_id": 30000142,
				"type_id":         99,
				"position": map[string]any{
					"x": 1.1,
					"y": 2.2,
					"z": 3.3,
				}}),
		)
		ctx := xesi.NewContextWithAccessToken(context.Background(), "DUMMY")
		ctx = xesi.NewContextWithCharacterID(ctx, int32(42))
		// when
		x1, err := s.GetOrCreateLocationESI(ctx, structureID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int64(structureID), x1.ID)
			assert.Equal(t, "V-3YG7 VI - The Capital", x1.Name)
			assert.Equal(t, owner, x1.Owner)
			assert.Equal(t, system, x1.SolarSystem)
			assert.Equal(t, myType, x1.Type)
			x2, err := st.GetLocation(ctx, structureID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should return error when trying to fetch structure without token", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		// when
		_, err := s.GetOrCreateLocationESI(context.Background(), structureID)
		// then
		assert.Error(t, err)
	})
	t.Run("should create empty structure from ESI when no access", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/structures/\d+/`,
			httpmock.NewJsonResponderOrPanic(http.StatusForbidden, map[string]any{
				"error": "forbidden",
			}),
		)
		ctx := xesi.NewContextWithAccessToken(context.Background(), "DUMMY")
		ctx = xesi.NewContextWithCharacterID(ctx, int32(42))
		// when
		x1, err := s.GetOrCreateLocationESI(ctx, structureID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int64(structureID), x1.ID)
			assert.Equal(t, "", x1.Name)
			assert.Nil(t, x1.Owner)
			assert.Nil(t, x1.SolarSystem)
			assert.Nil(t, x1.Type)
			x2, err := st.GetLocation(ctx, structureID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should return error when other http error occurs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/structures/\d+/`,
			httpmock.NewJsonResponderOrPanic(http.StatusNotFound, map[string]any{
				"error": "xxx",
			}),
		)
		ctx := xesi.NewContextWithAccessToken(context.Background(), "DUMMY")
		ctx = xesi.NewContextWithCharacterID(ctx, int32(42))
		// when
		_, err := s.GetOrCreateLocationESI(ctx, structureID)
		// then
		assert.Error(t, err)
	})
}

func TestGetStationServicesESI(t *testing.T) {
	// given
	xesi.ActivateRateLimiterMock()
	defer xesi.DeactivateRateLimiterMock()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(nil)
	httpmock.RegisterResponder(
		"GET",
		`=~^https://esi\.evetech\.net/v\d+/universe/stations/\d+/`,
		httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
			"max_dockable_ship_volume": 50000000,
			"name":                     "Jakanerva III - Moon 15 - Prompt Delivery Storage",
			"office_rental_cost":       10000,
			"owner":                    1000003,
			"position": map[string]any{
				"x": 165632286720,
				"y": 2771804160,
				"z": -2455331266560,
			},
			"race_id":                    1,
			"reprocessing_efficiency":    0.5,
			"reprocessing_stations_take": 0.05,
			"services": []string{
				"courier-missions",
				"reprocessing-plant",
				"market",
			},
			"station_id": 60000277,
			"system_id":  30000148,
			"type_id":    1531,
		},
		),
	)
	// when
	got, err := s.GetStationServicesESI(context.Background(), 42)
	// then
	if assert.NoError(t, err) {
		want := []string{
			"courier-missions",
			"market",
			"reprocessing-plant",
		}
		assert.Equal(t, want, got)
	}
}

func TestAddMissingLocations(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	xesi.ActivateRateLimiterMock()
	defer xesi.DeactivateRateLimiterMock()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("does nothing when given no ids", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		err := s.AddMissingLocations(ctx, set.Set[int64]{})
		if assert.NoError(t, err) {
			ids, err := st.ListEveLocationIDs(ctx)
			if assert.NoError(t, err) {
				assert.Equal(t, 0, ids.Size())
			}
		}
	})
	t.Run("can create missing location from scratch", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeAssetSafetyWrap})
		err := s.AddMissingLocations(ctx, set.Of[int64](2004))
		if assert.NoError(t, err) {
			got, err := st.ListEveLocationIDs(ctx)
			if assert.NoError(t, err) {
				want := set.Of[int64](2004)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
	t.Run("ignores invalid location IDs", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeAssetSafetyWrap})
		err := s.AddMissingLocations(ctx, set.Of[int64](2004, 0))
		if assert.NoError(t, err) {
			got, err := st.ListEveLocationIDs(ctx)
			if assert.NoError(t, err) {
				want := set.Of[int64](2004)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
	t.Run("can create missing locations only", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeAssetSafetyWrap})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeSolarSystem})
		_, err := s.GetOrCreateLocationESI(ctx, 888)
		if err != nil {
			t.Fatal(err)
		}
		err = s.AddMissingLocations(ctx, set.Of[int64](2004, 888))
		if assert.NoError(t, err) {
			got, err := st.ListEveLocationIDs(ctx)
			if assert.NoError(t, err) {
				want := set.Of[int64](2004, 888)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
}

func TestEntityIDsFromLocationsESI(t *testing.T) {
	xesi.ActivateRateLimiterMock()
	defer xesi.DeactivateRateLimiterMock()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(nil)
	ctx := context.Background()
	t.Run("can return owner from station", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/universe/stations/%d/", stationID),
			httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
				"max_dockable_ship_volume": 50000000,
				"name":                     "Jakanerva III - Moon 15 - Prompt Delivery Storage",
				"office_rental_cost":       10000,
				"owner":                    1000003,
				"position": map[string]any{
					"x": 165632286720,
					"y": 2771804160,
					"z": -2455331266560,
				},
				"race_id":                    1,
				"reprocessing_efficiency":    0.5,
				"reprocessing_stations_take": 0.05,
				"services": []string{
					"courier-missions",
					"reprocessing-plant",
					"market",
					"repair-facilities",
					"fitting",
					"news",
					"storage",
					"insurance",
					"docking",
					"office-rental",
					"loyalty-point-store",
					"navy-offices",
				},
				"station_id": 60000277,
				"system_id":  30000148,
				"type_id":    1531,
			}),
		)
		// when
		got, err := s.EntityIDsFromLocationsESI(ctx, []int64{60000277})
		// then
		if assert.NoError(t, err) {
			want := set.Of[int32](1000003)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can return owner of a structure", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/structures/\d+/`,
			httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
				"name":            "V-3YG7 VI - The Capital",
				"owner_id":        109299958,
				"solar_system_id": 30000142,
				"type_id":         99,
				"position": map[string]any{
					"x": 1.1,
					"y": 2.2,
					"z": 3.3,
				}}),
		)
		ctx := xesi.NewContextWithAccessToken(context.Background(), "DUMMY")
		ctx = xesi.NewContextWithCharacterID(ctx, int32(42))
		// when
		got, err := s.EntityIDsFromLocationsESI(ctx, []int64{structureID})
		// then
		if assert.NoError(t, err) {
			want := set.Of[int32](109299958)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("should return error when structure and no token", func(t *testing.T) {
		// when
		_, err := s.EntityIDsFromLocationsESI(ctx, []int64{structureID})
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("can filter out invalid IDs", func(t *testing.T) {
		cases := []struct {
			ownerID int
		}{
			{0}, {1},
		}
		for _, tc := range cases {
			httpmock.Reset()
			httpmock.RegisterResponder(
				"GET",
				`=~^https://esi\.evetech\.net/v\d+/universe/structures/\d+/`,
				httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
					"name":            "V-3YG7 VI - The Capital",
					"owner_id":        tc.ownerID,
					"solar_system_id": 30000142,
					"type_id":         99,
					"position": map[string]any{
						"x": 1.1,
						"y": 2.2,
						"z": 3.3,
					}}),
			)
			ctx := xesi.NewContextWithAccessToken(context.Background(), "DUMMY")
			ctx = xesi.NewContextWithCharacterID(ctx, int32(42))
			got, err := s.EntityIDsFromLocationsESI(ctx, []int64{structureID})
			if assert.NoError(t, err) {
				assert.Equal(t, 0, got.Size())
			}
		}
	})
	t.Run("should ignore structures with no access", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/structures/\d+/`,
			httpmock.NewJsonResponderOrPanic(http.StatusForbidden, map[string]string{
				"error": "no access",
			}),
		)
		ctx := xesi.NewContextWithAccessToken(context.Background(), "DUMMY")
		ctx = xesi.NewContextWithCharacterID(ctx, int32(42))
		// when
		got, err := s.EntityIDsFromLocationsESI(ctx, []int64{structureID})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 0, got.Size())
		}
	})
	t.Run("should return empty when no IDs given", func(t *testing.T) {
		// when
		got, err := s.EntityIDsFromLocationsESI(ctx, []int64{})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 0, got.Size())
		}
	})
}

package eveuniverseservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestAddMissing(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("should resolve missing entities", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"id": 47, "name": "Erik", "category": "character"},
			}),
		)
		// when
		err := s.AddMissing(ctx, set.Of[int64](47), set.Of[int64](), set.Of[int64]())
		// then
		require.NoError(t, err)
		o, err := st.GetEveEntity(ctx, 47)
		require.NoError(t, err)
		xassert.Equal(t, "Erik", o.Name)
		xassert.Equal(t, app.EveEntityCharacter, o.Category)
	})
	t.Run("should do nothing when all sets are empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		// when
		err := s.AddMissing(ctx, set.Of[int64](), set.Of[int64](), set.Of[int64]())
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
	})
	t.Run("should do nothing when locations already exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		loc := factory.CreateEveLocationStation()
		// when
		err := s.AddMissing(ctx, set.Of[int64](), set.Of(loc.ID), set.Of[int64]())
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
	})
	t.Run("should resolve missing types", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: 25})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi.evetech.net/universe/types/\d+`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"description": "The Rifter is a...",
				"group_id":    25,
				"name":        "Rifter",
				"published":   true,
				"type_id":     587,
			}),
		)
		// when
		err := s.AddMissing(ctx, set.Of[int64](), set.Of[int64](), set.Of[int64](587))
		// then
		require.NoError(t, err)
		o, err := st.GetEveType(ctx, 587)
		require.NoError(t, err)
		xassert.Equal(t, "Rifter", o.Name)
	})
	t.Run("should do nothing when types already exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		et := factory.CreateEveType()
		// when
		err := s.AddMissing(ctx, set.Of[int64](), set.Of[int64](), set.Of(et.ID))
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
	})
	t.Run("should resolve all kinds in one call", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000003})
		factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000148})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 1531})
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: 25})
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"id": 47, "name": "Erik", "category": "character"},
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/universe/stations/60000277",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"max_dockable_ship_volume":   50000000,
				"name":                       "Jakanerva III - Moon 15 - Prompt Delivery Storage",
				"office_rental_cost":         10000,
				"owner":                      1000003,
				"position":                   map[string]any{"x": 1, "y": 2, "z": 3},
				"race_id":                    1,
				"reprocessing_efficiency":    0.5,
				"reprocessing_stations_take": 0.05,
				"services":                   []string{"docking"},
				"station_id":                 60000277,
				"system_id":                  30000148,
				"type_id":                    1531,
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/universe/types/587",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"description": "The Rifter is a...",
				"group_id":    25,
				"name":        "Rifter",
				"published":   true,
				"type_id":     587,
			}),
		)
		// when
		err := s.AddMissing(ctx, set.Of[int64](47), set.Of[int64](60000277), set.Of[int64](587))
		// then
		require.NoError(t, err)
		_, err = st.GetEveEntity(ctx, 47)
		require.NoError(t, err)
		_, err = st.GetLocation(ctx, 60000277)
		require.NoError(t, err)
		_, err = st.GetEveType(ctx, 587)
		require.NoError(t, err)
	})
	t.Run("should return error when one lookup fails and keep the others", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"id": 47, "name": "Erik", "category": "character"},
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/universe/types/587",
			httpmock.NewJsonResponderOrPanic(404, map[string]any{"error": "Type not found"}),
		)
		// when
		err := s.AddMissing(ctx, set.Of[int64](47), set.Of[int64](), set.Of[int64](587))
		// then
		require.Error(t, err)
		_, err = st.GetEveEntity(ctx, 47)
		require.NoError(t, err)
	})
}

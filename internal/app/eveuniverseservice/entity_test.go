package eveuniverseservice_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestAddMissingEveEntities(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	responder := func(req *http.Request) (*http.Response, error) {
		var ids []int64
		if err := json.NewDecoder(req.Body).Decode(&ids); err != nil {
			return httpmock.NewJsonResponse(400, map[string]any{"error": "invalid request"})
		}
		var results []map[string]any
		for _, id := range ids {
			switch id {
			case 47:
				results = append(results, map[string]any{
					"id":       47,
					"name":     "Erik",
					"category": "character",
				})
			default:
				return httpmock.NewJsonResponse(404, map[string]any{"error": "Invalid ID"})
			}
		}
		return httpmock.NewJsonResponse(200, results)
	}
	t.Run("do nothing when all entities already exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("POST", "https://esi.evetech.net/universe/names", responder)
		e1 := factory.CreateEveEntityCharacter()
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of(e1.ID))
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
		xassert.Equal(t, 0, ids.Size())
	})
	t.Run("can resolve missing entities", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("POST", "https://esi.evetech.net/universe/names", responder)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of[int64](47))
		// then
		require.NoError(t, err)
		xassert.Equal(t, 1, httpmock.GetTotalCallCount())
		assert.True(t, set.Of[int64](47).Equal(ids))
		o, err := st.GetEveEntity(ctx, 47)
		require.NoError(t, err)
		xassert.Equal(t, "Erik", o.Name)
		xassert.Equal(t, app.EveEntityCharacter, o.Category)
	})
	t.Run("can report normal error correctly", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewErrorResponder(fmt.Errorf("failed")),
		)
		// when
		_, err := s.AddMissingEntities(ctx, set.Of[int64](47))
		// then
		assert.Error(t, err)
	})
	t.Run("can resolve mix of missing and non-missing entities", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		e1 := factory.CreateEveEntityAlliance()
		httpmock.Reset()
		httpmock.RegisterResponder("POST",
			"https://esi.evetech.net/universe/names",
			responder,
		)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of(47, e1.ID))
		// then
		require.NoError(t, err)
		xassert.Equal(t, 1, httpmock.GetTotalCallCount())
		assert.True(t, set.Of[int64](47).Equal(ids))
	})
	t.Run("can resolve more then 1000 IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const count = 1001
		ids := make([]int64, count)
		data := make([]map[string]any, count)
		for i := range count {
			id := int64(i) + 1000
			ids[i] = id
			obj := map[string]any{
				"id":       id,
				"name":     fmt.Sprintf("Name #%d", id),
				"category": "character",
			}
			data[i] = obj
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		// when
		missing, err := s.AddMissingEntities(ctx, set.Of(ids...))
		// then
		require.NoError(t, err)
		xassert.Equal(t, 2, httpmock.GetTotalCallCount())
		xassert.Equal(t, count, missing.Size())
		ids2, err := st.ListEveEntityIDs(ctx)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(ids...), ids2)
	})
	t.Run("should store unresolvable IDs accordingly", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("POST",
			"https://esi.evetech.net/universe/names",
			responder,
		)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of[int64](666))
		// then
		require.NoError(t, err)
		assert.GreaterOrEqual(t, 1, httpmock.GetTotalCallCount())
		assert.True(t, set.Of[int64](666).Equal(ids))
		o, err := st.GetEveEntity(ctx, 666)
		require.NoError(t, err)
		xassert.Equal(t, "?", o.Name)
		xassert.Equal(t, app.EveEntityUnknown, o.Category)
	})
	t.Run("should not call API with known invalid IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("POST",
			"https://esi.evetech.net/universe/names",
			responder,
		)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of[int64](1))
		// then
		require.NoError(t, err)
		assert.GreaterOrEqual(t, 0, httpmock.GetTotalCallCount())
		xassert.Equal(t, 1, ids.Size())
		o, err := st.GetEveEntity(ctx, 1)
		require.NoError(t, err)
		xassert.Equal(t, "?", o.Name)
		xassert.Equal(t, app.EveEntityUnknown, o.Category)
	})
	t.Run("should return error when trying to resolve large IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("POST", "https://esi.evetech.net/universe/names", responder)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of[int64](1047607396377))
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
		assert.GreaterOrEqual(t, 0, httpmock.GetTotalCallCount())
		xassert.Equal(t, 0, ids.Size())
		ids2, err := st.ListEveEntityIDs(ctx)
		require.NoError(t, err)
		xassert.Equal(t, 0, ids2.Size())
	})
	t.Run("should do nothing with ID 0", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("POST", "https://esi.evetech.net/universe/names", responder)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of[int64](0))
		// then
		require.NoError(t, err)
		assert.GreaterOrEqual(t, 0, httpmock.GetTotalCallCount())
		xassert.Equal(t, 0, ids.Size())
		r := db.QueryRow("SELECT count(*) FROM eve_entities;")
		var c int
		err = r.Scan(&c)
		require.NoError(t, err)
		xassert.Equal(t, 0, c)
	})
	t.Run("can deal with a mix of resolveable and unresolveable IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			responder,
		)
		// when
		_, err := s.AddMissingEntities(ctx, set.Of[int64](47, 666))
		// then
		require.NoError(t, err)
		o1, err := st.GetEveEntity(ctx, 47)
		require.NoError(t, err)
		xassert.Equal(t, "Erik", o1.Name)
		xassert.Equal(t, app.EveEntityCharacter, o1.Category)
		o2, err := st.GetEveEntity(ctx, 666)
		require.NoError(t, err)
		xassert.Equal(t, app.EveEntityUnknown, o2.Category)
	})
	t.Run("can deal with a mix of resolveable and invalid IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			responder,
		)
		// when
		_, err := s.AddMissingEntities(ctx, set.Of[int64](47, 1))
		// then
		require.NoError(t, err)
		o1, err := st.GetEveEntity(ctx, 47)
		require.NoError(t, err)
		xassert.Equal(t, "Erik", o1.Name)
		xassert.Equal(t, app.EveEntityCharacter, o1.Category)
		o2, err := st.GetEveEntity(ctx, 1)
		require.NoError(t, err)
		xassert.Equal(t, app.EveEntityUnknown, o2.Category)
	})
	t.Run("should do nothing when no ids passed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("POST",
			"https://esi.evetech.net/universe/names",
			responder,
		)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of[int64]())
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
		xassert.Equal(t, 0, ids.Size())
		r := db.QueryRow("SELECT count(*) FROM eve_entities;")
		var c int
		err = r.Scan(&c)
		require.NoError(t, err)
		xassert.Equal(t, 0, c)
	})
	t.Run("should report error when ESI response is incomplete", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewJsonResponderOrPanic(http.StatusOK, []map[string]any{{
				"id":       47,
				"name":     "Erik",
				"category": "character",
			}}),
		)
		// when
		_, err := s.AddMissingEntities(ctx, set.Of[int64](47, 12))
		// then
		require.Error(t, err)
	})
}

func TestGetOrCreateEntityESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("return existing entity", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		x1 := factory.CreateEveEntityCharacter()
		// when
		x2, err := s.GetOrCreateEntityESI(ctx, x1.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
		xassert.Equal(t, x2, x1)
	})
	t.Run("create entity from ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"id": 42, "name": "Erik", "category": "character"},
			}),
		)
		// when
		x, err := s.GetOrCreateEntityESI(ctx, 42)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 1, httpmock.GetTotalCallCount())
		xassert.Equal(t, 42, x.ID)
		xassert.Equal(t, "Erik", x.Name)
		xassert.Equal(t, app.EveEntityCharacter, x.Category)
	})
}

func TestToEveEntities(t *testing.T) {
	ctx := context.Background()
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("should resolve normal IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		e1 := factory.CreateEveEntity()
		e2 := factory.CreateEveEntity()
		// when
		oo, err := s.ToEntities(ctx, set.Of(e1.ID, e2.ID))
		// then
		require.NoError(t, err)
		xassert.Equal(t, map[int64]*app.EveEntity{e1.ID: e1, e2.ID: e2}, oo)
	})
	t.Run("should map unknown IDs to empty objects", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		// when
		oo, err := s.ToEntities(ctx, set.Of[int64](0, 1))
		// then
		require.NoError(t, err)
		xassert.Equal(t, &app.EveEntity{ID: 0}, oo[0])
		xassert.Equal(t, &app.EveEntity{ID: 1, Name: "?", Category: app.EveEntityUnknown}, oo[1])
	})
}

func TestUpdateAllEntityESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("should update existing entity", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 42})
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"id": 42, "name": "Erik", "category": "character"},
			}),
		)
		// when
		got, err := s.UpdateAllEntitiesESI(ctx)
		// then
		require.NoError(t, err)
		want := set.Of[int64](42)
		xassert.Equal(t, want, got)
		o2, err := st.GetEveEntity(ctx, 42)
		require.NoError(t, err)
		xassert.Equal(t, 42, o2.ID)
		xassert.Equal(t, "Erik", o2.Name)
		xassert.Equal(t, app.EveEntityCharacter, o2.Category)
	})
	t.Run("should detect when not changed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		factory.CreateEveEntityCharacter(app.EveEntity{
			ID:   42,
			Name: "Erik",
		})
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"id": 42, "name": "Erik", "category": "character"},
			}),
		)
		// when
		got, err := s.UpdateAllEntitiesESI(ctx)
		// then
		require.NoError(t, err)
		want := set.Of[int64]()
		xassert.Equal(t, want, got)
	})
}

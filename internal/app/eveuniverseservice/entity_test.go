package eveuniverseservice_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestAddMissingEveEntities(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	t.Run("do nothing when all entities already exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		e1 := factory.CreateEveEntityCharacter()
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of(e1.ID))
		// then
		assert.Equal(t, 0, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Equal(t, 0, ids.Size())
		}
	})
	t.Run("can resolve missing entities", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"id": 47, "name": "Erik", "category": "character"},
			}),
		)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of[int32](47))
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.True(t, set.Of[int32](47).Equal(ids))
			e, err := st.GetEveEntity(ctx, 47)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "Erik", e.Name)
			assert.Equal(t, app.EveEntityCharacter, e.Category)
		}
	})
	t.Run("can report normal error correctly", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		// when
		_, err := s.AddMissingEntities(ctx, set.Of[int32](47))
		// then
		assert.Error(t, err)
	})
	t.Run("can resolve mix of missing and non-missing entities", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		e1 := factory.CreateEveEntityAlliance()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"id": 47, "name": "Erik", "category": "character"},
			}),
		)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of(47, e1.ID))
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.True(t, set.Of[int32](47).Equal(ids))
		}
	})
	t.Run("can resolve more then 1000 IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		const count = 1001
		ids := make([]int32, count)
		data := make([]map[string]any, count)
		for i := range count {
			id := int32(i) + 1000
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
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		// when
		missing, err := s.AddMissingEntities(ctx, set.Of(ids...))
		// then
		assert.Equal(t, 2, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Equal(t, count, missing.Size())
			ids2, err := st.ListEveEntityIDs(ctx)
			if err != nil {
				t.Fatal(err)
			}
			assert.ElementsMatch(t, ids, ids2.Slice())
		}
	})
	t.Run("should store unresolvable IDs accordingly", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.NewJsonResponderOrPanic(404, map[string]any{"error": "not found"}),
		)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of[int32](666))
		// then
		assert.GreaterOrEqual(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.True(t, set.Of[int32](666).Equal(ids))
			e, err := st.GetEveEntity(ctx, 666)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "?", e.Name)
			assert.Equal(t, app.EveEntityUnknown, e.Category)
		}
	})
	t.Run("should not call API with known invalid IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.NewJsonResponderOrPanic(404, map[string]any{"error": "not found"}),
		)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of[int32](1))
		// then
		assert.GreaterOrEqual(t, 0, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Equal(t, 0, ids.Size())
			e, err := st.GetEveEntity(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "?", e.Name)
			assert.Equal(t, app.EveEntityUnknown, e.Category)
		}
	})
	t.Run("should do nothing with ID 0", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.NewJsonResponderOrPanic(404, map[string]any{"error": "not found"}),
		)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of[int32](0))
		// then
		assert.GreaterOrEqual(t, 0, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Equal(t, 0, ids.Size())
			r := db.QueryRow("SELECT count(*) FROM eve_entities;")
			var c int
			if err := r.Scan(&c); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 0, c)
		}
	})
	t.Run("can deal with a mix of valid and invalid IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"id": 47, "name": "Erik", "category": "character"},
			}),
		)
		httpmock.RegisterMatcherResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.BodyContainsString("666"),
			httpmock.NewJsonResponderOrPanic(404, map[string]any{"error": "Invalid ID"}),
		)
		// when
		_, err := s.AddMissingEntities(ctx, set.Of[int32](47, 666))
		// then
		assert.LessOrEqual(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			e1, err := st.GetEveEntity(ctx, 47)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "Erik", e1.Name)
			assert.Equal(t, app.EveEntityCharacter, e1.Category)
			e2, err := st.GetEveEntity(ctx, 666)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, app.EveEntityUnknown, e2.Category)
		}
	})
	t.Run("should do nothing when no ids passed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.NewJsonResponderOrPanic(404, map[string]any{"error": "not found"}),
		)
		// when
		ids, err := s.AddMissingEntities(ctx, set.Of[int32]())
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 0, httpmock.GetTotalCallCount())
			assert.Equal(t, 0, ids.Size())
			r := db.QueryRow("SELECT count(*) FROM eve_entities;")
			var c int
			if err := r.Scan(&c); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 0, c)
		}
	})
}

func TestGerOrCreateEntityESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	t.Run("return existing entity", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		x1 := factory.CreateEveEntityCharacter()
		// when
		x2, err := s.GetOrCreateEntityESI(ctx, x1.ID)
		// then
		assert.Equal(t, 0, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Equal(t, x2, x1)
		}
	})
	t.Run("create entity from ESI", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"id": 42, "name": "Erik", "category": "character"},
			}),
		)
		// when
		x, err := s.GetOrCreateEntityESI(ctx, 42)
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.EqualValues(t, 42, x.ID)
			assert.Equal(t, "Erik", x.Name)
			assert.Equal(t, app.EveEntityCharacter, x.Category)
		}
	})
}

func TestToEveEntities(t *testing.T) {
	ctx := context.Background()
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	t.Run("should resolve normal IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		e1 := factory.CreateEveEntity()
		e2 := factory.CreateEveEntity()
		// when
		oo, err := s.ToEntities(ctx, set.Of(e1.ID, e2.ID))
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, map[int32]*app.EveEntity{e1.ID: e1, e2.ID: e2}, oo)
		}
	})
	t.Run("should map unknown IDs to empty objects", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		// when
		oo, err := s.ToEntities(ctx, set.Of[int32](0, 1))
		// then
		if assert.NoError(t, err) {
			assert.EqualValues(t, &app.EveEntity{ID: 0}, oo[0])
			assert.EqualValues(t, &app.EveEntity{ID: 1, Name: "?", Category: app.EveEntityUnknown}, oo[1])
		}
	})
}

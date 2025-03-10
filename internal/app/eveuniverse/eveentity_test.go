package eveuniverse_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestAddMissingEveEntities(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(r, client)
	t.Run("do nothing when all entities already exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		e1 := factory.CreateEveEntityCharacter()
		// when
		ids, err := s.AddMissingEveEntities(ctx, []int32{e1.ID})
		// then
		assert.Equal(t, 0, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Len(t, ids, 0)
		}
	})
	t.Run("can resolve missing entities", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		data := []map[string]any{
			{"id": 47, "name": "Erik", "category": "character"},
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/v3/universe/names/",
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, data)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		// when
		ids, err := s.AddMissingEveEntities(ctx, []int32{47})
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Equal(t, int32(47), ids[0])
			e, err := r.GetEveEntity(ctx, 47)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, e.Name, "Erik")
			assert.Equal(t, e.Category, app.EveEntityCharacter)
		}
	})
	t.Run("can report normal error correctly", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		// when
		_, err := s.AddMissingEveEntities(ctx, []int32{47})
		// then
		assert.Error(t, err)
	})
	t.Run("can resolve mix of missing and non-missing entities", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		e1 := factory.CreateEveEntityAlliance()
		data := []map[string]any{
			{"id": 47, "name": "Erik", "category": "character"},
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/v3/universe/names/",
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, data)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		// when
		ids, err := s.AddMissingEveEntities(ctx, []int32{47, e1.ID})
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Equal(t, int32(47), ids[0])
		}
	})
	t.Run("can resolve more then 1000 IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		const count = 1001
		ids := make([]int32, count)
		data := make([]map[string]any, count)
		for i := range count {
			id := int32(i) + 1
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
			"https://esi.evetech.net/v3/universe/names/",
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, data)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		// when
		missing, err := s.AddMissingEveEntities(ctx, ids)
		// then
		assert.Equal(t, 2, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Len(t, missing, count)
			ids2, err := r.ListEveEntityIDs(ctx)
			if err != nil {
				t.Fatal(err)
			}
			assert.ElementsMatch(t, ids, ids2.ToSlice())
		}
	})
	t.Run("should store unresolvable IDs accordingly", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("POST", "https://esi.evetech.net/v3/universe/names/",
			httpmock.NewStringResponder(404, ""))
		// when
		ids, err := s.AddMissingEveEntities(ctx, []int32{666})
		// then
		assert.GreaterOrEqual(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Equal(t, int32(666), ids[0])
			e, err := r.GetEveEntity(ctx, 666)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, e.Name, "?")
			assert.Equal(t, e.Category, app.EveEntityUnknown)
		}
	})
	t.Run("should not call API with known invalid IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("POST", "https://esi.evetech.net/v3/universe/names/",
			httpmock.NewStringResponder(404, ""))
		// when
		ids, err := s.AddMissingEveEntities(ctx, []int32{1})
		// then
		assert.GreaterOrEqual(t, 0, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Len(t, ids, 1)
			assert.Equal(t, int32(1), ids[0])
			e, err := r.GetEveEntity(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, e.Name, "?")
			assert.Equal(t, e.Category, app.EveEntityUnknown)
		}
	})
	t.Run("can deal with a mix of valid and invalid IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		data := []map[string]any{
			{"id": 47, "name": "Erik", "category": "character"},
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST", "https://esi.evetech.net/v3/universe/names/",
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, data)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		httpmock.RegisterMatcherResponder(
			"POST", "https://esi.evetech.net/v3/universe/names/",
			httpmock.BodyContainsString("666"),
			httpmock.NewStringResponder(404, `{"error":"Invalid ID"}`))
		// when
		_, err := s.AddMissingEveEntities(ctx, []int32{47, 666})
		// then
		assert.LessOrEqual(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			e1, err := r.GetEveEntity(ctx, 47)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, e1.Name, "Erik")
			assert.Equal(t, e1.Category, app.EveEntityCharacter)
			e2, err := r.GetEveEntity(ctx, 666)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, e2.Category, app.EveEntityUnknown)
		}
	})
}

func TestResolveEveEntities(t *testing.T) {
	ctx := context.Background()
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(st, client)
	t.Run("should resolve EveEntity", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		e1 := factory.CreateEveEntity()
		// when
		r, err := s.ToEveEntities(ctx, []int32{e1.ID})
		// then
		if assert.NoError(t, err) {
			assert.Len(t, r, 1)
			assert.Equal(t, *e1, *r[e1.ID])
		}
	})
	t.Run("should return zero value for ID 0", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		// when
		r, err := s.ToEveEntities(ctx, []int32{0})
		// then
		if assert.NoError(t, err) {
			assert.Len(t, r, 1)
			assert.Equal(t, app.EveEntity{}, *r[0])
		}
	})
}

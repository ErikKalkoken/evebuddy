package eveuniverse_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/storage/testutil"
)

func TestResolveUncleanEveEntities(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(r, client, nil, nil)
	t.Run("Can resolve existing when it has category", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		e1 := factory.CreateEveEntityCharacter(model.EveEntity{Name: "Erik"})
		e2 := model.EveEntity{Name: "Erik", Category: model.EveEntityCharacter}
		// when
		ee, err := s.ResolveUncleanEveEntities(ctx, []*model.EveEntity{&e2})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, e1, ee[0])
			assert.Len(t, ee, 1)
		}
	})
	t.Run("Can resolve name through ESI", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		data := map[string][]map[string]any{
			"characters": {
				{"id": 47, "name": "Erik"},
			}}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/v1/universe/ids/",
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, data)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		e := model.EveEntity{Name: "Erik", Category: model.EveEntityUndefined}
		// when
		ee, err := s.ResolveUncleanEveEntities(ctx, []*model.EveEntity{&e})
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Equal(t, int32(47), ee[0].ID)
			assert.Equal(t, "Erik", ee[0].Name)
			assert.Equal(t, model.EveEntityCharacter, ee[0].Category)
			assert.Len(t, ee, 1)
		}
	})
	t.Run("Return error when name does not match", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		data := `{}`
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/v1/universe/ids/",
			httpmock.NewStringResponder(200, data),
		)
		e := model.EveEntity{Name: "Erik", Category: model.EveEntityUndefined}
		// when
		_, err := s.ResolveUncleanEveEntities(ctx, []*model.EveEntity{&e})
		// then
		assert.ErrorIs(t, err, eveuniverse.ErrEveEntityNameNoMatch)
	})
	t.Run("Return error when name matches more then once", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		data := `{}`
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/v1/universe/ids/",
			httpmock.NewStringResponder(200, data),
		)
		factory.CreateEveEntityCharacter(model.EveEntity{Name: "Erik"})
		factory.CreateEveEntityCorporation(model.EveEntity{Name: "Erik"})
		e := model.EveEntity{Name: "Erik", Category: model.EveEntityUndefined}
		// when
		_, err := s.ResolveUncleanEveEntities(ctx, []*model.EveEntity{&e})
		// then
		assert.ErrorIs(t, err, eveuniverse.ErrEveEntityNameMultipleMatches)
	})
	t.Run("Return error when name and category matches more then once", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		data := `{}`
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/v1/universe/ids/",
			httpmock.NewStringResponder(200, data),
		)
		factory.CreateEveEntityCharacter(model.EveEntity{Name: "Erik"})
		factory.CreateEveEntityCharacter(model.EveEntity{Name: "Erik"})
		e := model.EveEntity{Name: "Erik", Category: model.EveEntityUndefined}
		// when
		_, err := s.ResolveUncleanEveEntities(ctx, []*model.EveEntity{&e})
		// then
		assert.ErrorIs(t, err, eveuniverse.ErrEveEntityNameMultipleMatches)
	})
}

func TestAddMissingEveEntities(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(r, client, nil, nil)
	t.Run("do noting when not entities are missing", func(t *testing.T) {
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
			assert.Equal(t, e.Category, model.EveEntityCharacter)
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
			assert.Equal(t, ids, ids2)
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
			assert.Equal(t, e.Category, model.EveEntityUnknown)
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
			assert.Equal(t, e1.Category, model.EveEntityCharacter)
			e2, err := r.GetEveEntity(ctx, 666)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, e2.Category, model.EveEntityUnknown)
		}
	})
}

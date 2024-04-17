package service_test

import (
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/service"
	"example/evebuddy/internal/testutil"
)

func TestResolveUncleanEveEntities(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	// ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := service.NewService(r)
	t.Run("Can resolve existing when it has category", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		e1 := factory.CreateEveEntityCharacter(model.EveEntity{Name: "Erik"})
		e2 := model.EveEntity{Name: "Erik", Category: model.EveEntityCharacter}
		// when
		ee, err := s.ResolveUncleanEveEntities([]model.EveEntity{e2})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, e1, ee[0])
			assert.Len(t, ee, 1)
		}
	})
	t.Run("Can resolve name through ESI", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		data := map[string][]map[string]interface{}{
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
		ee, err := s.ResolveUncleanEveEntities([]model.EveEntity{e})
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
		_, err := s.ResolveUncleanEveEntities([]model.EveEntity{e})
		// then
		assert.ErrorIs(t, err, service.ErrEveEntityNameNoMatch)
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
		factory.CreateEveEntityCharacter(model.EveEntity{Name: "Erik"})
		e := model.EveEntity{Name: "Erik", Category: model.EveEntityUndefined}
		// when
		_, err := s.ResolveUncleanEveEntities([]model.EveEntity{e})
		// then
		assert.ErrorIs(t, err, service.ErrEveEntityNameMultipleMatches)
	})
}

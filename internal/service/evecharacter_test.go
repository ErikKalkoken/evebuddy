package service_test

import (
	"context"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/service"
	"example/evebuddy/internal/testutil"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetOrCreateEveCharacterESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := service.NewService(r)
	ctx := context.Background()
	t.Run("should return existing character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateEveCharacter()
		// when
		x1, err := s.GetOrCreateEveCharacterESI(c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c.ID, x1.ID)
		}
	})
	t.Run("should fetch character from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		characterID := int32(95465499)
		factory.CreateEveEntityCharacter(model.EveEntity{ID: characterID})
		factory.CreateEveEntityCorporation(model.EveEntity{ID: 109299958})
		factory.CreateEveRace(model.EveRace{ID: 2})
		httpmock.Reset()
		data := `{
			"birthday": "2015-03-24T11:37:00Z",
			"bloodline_id": 3,
			"corporation_id": 109299958,
			"description": "",
			"gender": "male",
			"name": "CCP Bartender",
			"race_id": 2,
			"title": "All round pretty awesome guy"
		  }`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/", characterID),
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateEveCharacterESI(characterID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, characterID, x1.ID)
			assert.Equal(t, "CCP Bartender", x1.Name)
			x2, err := r.GetEveCharacter(ctx, characterID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1.Name, x2.Name)
			}
		}
	})
}

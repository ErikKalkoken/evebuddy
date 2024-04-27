package service_test

import (
	"context"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/service"
	"example/evebuddy/internal/storage"
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

func TestUpdateAllEveCharactersESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := service.NewService(r)
	ctx := context.Background()
	t.Run("should update character from ESI", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		characterID := int32(95465499)
		factory.CreateEveEntityCharacter(model.EveEntity{ID: characterID})
		factory.CreateEveEntityCorporation(model.EveEntity{ID: 109299958})
		factory.CreateEveEntityAlliance(model.EveEntity{ID: 434243723})
		factory.CreateEveCharacter(storage.CreateEveCharacterParams{ID: characterID})
		httpmock.Reset()
		dataCharacter := `{
			"birthday": "2015-03-24T11:37:00Z",
			"bloodline_id": 3,
			"corporation_id": 109299958,
			"description": "",
			"gender": "male",
			"name": "CCP Bartender",
			"race_id": 2,
			"title": "All round pretty awesome guy",
			"security_status": -9.9
		  }`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/", characterID),
			httpmock.NewStringResponder(200, dataCharacter).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		dataAffiliation := `[
			{
			  "alliance_id": 434243723,
			  "character_id": 95465499,
			  "corporation_id": 109299958
			}
		  ]`
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/v2/characters/affiliation/",
			httpmock.NewStringResponder(200, dataAffiliation).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		err := s.UpdateAllEveCharactersESI()
		// then
		if assert.NoError(t, err) {
			x, err := r.GetEveCharacter(ctx, characterID)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(109299958), x.Corporation.ID)
				assert.Equal(t, int32(434243723), x.Alliance.ID)
				assert.LessOrEqual(t, -9.9, x.SecurityStatus)
			}
		}
	})
}

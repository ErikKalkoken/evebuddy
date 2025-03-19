package eveuniverse_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestGetOrCreateEveCharacterESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(r, client)
	ctx := context.Background()
	t.Run("should return existing character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateEveCharacter()
		// when
		x1, err := s.GetOrCreateCharacterESI(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c.ID, x1.ID)
		}
	})
	t.Run("should fetch character from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		characterID := int32(95465499)
		factory.CreateEveEntityCharacter(app.EveEntity{ID: characterID})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		factory.CreateEveRace(app.EveRace{ID: 2})
		httpmock.Reset()
		data := map[string]any{
			"birthday":        "2015-03-24T11:37:00Z",
			"bloodline_id":    3,
			"corporation_id":  109299958,
			"description":     "bla bla",
			"gender":          "male",
			"name":            "CCP Bartender",
			"race_id":         2,
			"security_status": -9.9,
			"title":           "All round pretty awesome guy",
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/", characterID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		x1, err := s.GetOrCreateCharacterESI(ctx, characterID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, characterID, x1.ID)
			assert.Equal(t, time.Date(2015, 03, 24, 11, 37, 0, 0, time.UTC), x1.Birthday)
			assert.Equal(t, int32(109299958), x1.Corporation.ID)
			assert.Equal(t, "bla bla", x1.Description)
			assert.Equal(t, "male", x1.Gender)
			assert.Equal(t, "CCP Bartender", x1.Name)
			assert.Equal(t, int32(2), x1.Race.ID)
			assert.Equal(t, "All round pretty awesome guy", x1.Title)
			assert.InDelta(t, -9.9, x1.SecurityStatus, 0.01)
			x2, err := r.GetEveCharacter(ctx, characterID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1.Birthday.UTC(), x2.Birthday.UTC())
				assert.Equal(t, x1.Corporation.ID, x2.Corporation.ID)
				assert.Equal(t, x1.Description, x2.Description)
				assert.Equal(t, x1.Gender, x2.Gender)
				assert.Equal(t, x1.Name, x2.Name)
				assert.Equal(t, x1.Race.ID, x2.Race.ID)
				assert.Equal(t, x1.SecurityStatus, x2.SecurityStatus)
				assert.Equal(t, x1.Title, x2.Title)
			}
		}
	})
}

func TestUpdateAllEveCharactersESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(r, client)
	ctx := context.Background()
	t.Run("should update character from ESI", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		characterID := int32(95465499)
		factory.CreateEveEntityCharacter(app.EveEntity{ID: characterID})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		factory.CreateEveEntityAlliance(app.EveEntity{ID: 434243723})
		factory.CreateEveCharacter(storage.CreateEveCharacterParams{ID: characterID})
		httpmock.Reset()
		dataCharacter := map[string]any{
			"birthday":        "2015-03-24T11:37:00Z",
			"bloodline_id":    3,
			"corporation_id":  109299958,
			"description":     "bla bla",
			"gender":          "male",
			"name":            "CCP Bartender",
			"race_id":         2,
			"security_status": -9.9,
			"title":           "All round pretty awesome guy",
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/", characterID),
			httpmock.NewJsonResponderOrPanic(200, dataCharacter))
		dataAffiliation := []map[string]any{
			{
				"alliance_id":    434243723,
				"character_id":   95465499,
				"corporation_id": 109299958,
			}}
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/v2/characters/affiliation/",
			httpmock.NewJsonResponderOrPanic(200, dataAffiliation))
		// when
		err := s.UpdateAllEveCharactersESI(ctx)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetEveCharacter(ctx, characterID)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(434243723), x.Alliance.ID)
				assert.Equal(t, int32(109299958), x.Corporation.ID)
				assert.Equal(t, "bla bla", x.Description)
				assert.InDelta(t, -9.9, x.SecurityStatus, 0.01)
				assert.Equal(t, "All round pretty awesome guy", x.Title)
			}
		}
	})
}

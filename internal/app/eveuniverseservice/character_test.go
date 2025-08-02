package eveuniverseservice_test

import (
	"context"

	"testing"
	"time"

	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestGetOrCreateEveCharacterESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.New(eveuniverseservice.Params{
		Storage:   st,
		ESIClient: goesi.NewAPIClient(nil, ""),
	})
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
	t.Run("should fetch minimal character from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		characterID := int32(95465499)
		factory.CreateEveEntityCharacter(app.EveEntity{ID: characterID})
		corporation := factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		race := factory.CreateEveRace(app.EveRace{ID: 2})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        "2015-03-24T11:37:00Z",
				"bloodline_id":    3,
				"corporation_id":  109299958,
				"gender":          "male",
				"name":            "CCP Bartender",
				"race_id":         2,
				"security_status": -9.9,
			}))
		// when
		x1, err := s.GetOrCreateCharacterESI(ctx, characterID)
		// then
		if assert.NoError(t, err) {
			assert.Nil(t, x1.Alliance)
			assert.Nil(t, x1.Faction)
			assert.Equal(t, characterID, x1.ID)
			assert.Equal(t, time.Date(2015, 03, 24, 11, 37, 0, 0, time.UTC), x1.Birthday)
			assert.Equal(t, corporation, x1.Corporation)
			assert.Empty(t, x1.Description)
			assert.Equal(t, "male", x1.Gender)
			assert.Equal(t, "CCP Bartender", x1.Name)
			assert.Equal(t, race, x1.Race)
			assert.Empty(t, x1.Title)
			assert.InDelta(t, -9.9, x1.SecurityStatus, 0.01)
			x2, err := st.GetEveCharacter(ctx, characterID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should fetch full character from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		characterID := int32(95465499)
		factory.CreateEveEntityCharacter(app.EveEntity{ID: characterID})
		alliance := factory.CreateEveEntityCorporation(app.EveEntity{ID: 434243723})
		corporation := factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		faction := factory.CreateEveEntity(app.EveEntity{ID: 500004, Category: app.EveEntityFaction})
		race := factory.CreateEveRace(app.EveRace{ID: 2})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        "2015-03-24T11:37:00Z",
				"bloodline_id":    3,
				"alliance_id":     434243723,
				"corporation_id":  109299958,
				"faction_id":      500004,
				"description":     "bla bla",
				"gender":          "male",
				"name":            "CCP Bartender",
				"race_id":         2,
				"security_status": -9.9,
				"title":           "All round pretty awesome guy",
			}))
		// when
		x1, err := s.GetOrCreateCharacterESI(ctx, characterID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, characterID, x1.ID)
			assert.Equal(t, time.Date(2015, 03, 24, 11, 37, 0, 0, time.UTC), x1.Birthday)
			assert.Equal(t, alliance, x1.Alliance)
			assert.Equal(t, corporation, x1.Corporation)
			assert.Equal(t, faction, x1.Faction)
			assert.Equal(t, "bla bla", x1.Description)
			assert.Equal(t, "male", x1.Gender)
			assert.Equal(t, "CCP Bartender", x1.Name)
			assert.Equal(t, race, x1.Race)
			assert.Equal(t, "All round pretty awesome guy", x1.Title)
			assert.InDelta(t, -9.9, x1.SecurityStatus, 0.01)
			x2, err := st.GetEveCharacter(ctx, characterID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestUpdateAllEveCharactersESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should update character from ESI", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		const characterID = 95465499
		factory.CreateEveEntityCharacter(app.EveEntity{ID: characterID})
		factory.CreateEveCharacter(storage.CreateEveCharacterParams{ID: characterID})
		alliance := factory.CreateEveEntityAlliance()
		corporation := factory.CreateEveEntityCorporation()
		faction := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityFaction})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        "2015-03-24T11:37:00Z",
				"bloodline_id":    3,
				"corporation_id":  corporation.ID,
				"description":     "bla bla",
				"gender":          "male",
				"name":            "CCP Bartender",
				"race_id":         2,
				"security_status": -9.9,
				"title":           "All round pretty awesome guy",
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/characters/affiliation/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id":    alliance.ID,
					"character_id":   characterID,
					"corporation_id": corporation.ID,
					"faction_id":     faction.ID,
				}}),
		)
		// when
		err := s.UpdateAllCharactersESI(ctx)
		// then
		if assert.NoError(t, err) {
			ec, err := st.GetEveCharacter(ctx, characterID)
			if assert.NoError(t, err) {
				assert.Equal(t, "CCP Bartender", ec.Name)
				assert.Equal(t, alliance, ec.Alliance)
				assert.Equal(t, corporation, ec.Corporation)
				assert.Equal(t, "bla bla", ec.Description)
				assert.InDelta(t, -9.9, ec.SecurityStatus, 0.01)
				assert.Equal(t, "All round pretty awesome guy", ec.Title)
			}
			ee, err := st.GetEveEntity(ctx, characterID)
			if assert.NoError(t, err) {
				assert.Equal(t, "CCP Bartender", ee.Name)
				assert.Equal(t, app.EveEntityCharacter, ee.Category)
			}
		}
	})
}

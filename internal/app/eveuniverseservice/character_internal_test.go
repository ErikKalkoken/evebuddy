package eveuniverseservice

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestUpdateEveCharacterESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewTestService(st)
	ctx := context.Background()
	t.Run("should update from ESI", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		o1 := factory.CreateEveCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: o1.ID})
		alliance := factory.CreateEveEntityAlliance(app.EveEntity{ID: 434243723})
		corporation := factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		faction := factory.CreateEveEntity(app.EveEntity{ID: 500004, Category: app.EveEntityFaction})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        "2015-03-24T11:37:00Z",
				"bloodline_id":    3,
				"corporation_id":  109299958,
				"description":     "bla bla",
				"gender":          "male",
				"name":            "CCP Bartender",
				"race_id":         2,
				"security_status": -9.9,
				"title":           "All round pretty awesome guy",
			}))
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/characters/affiliation/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id":    alliance.ID,
					"character_id":   o1.ID,
					"corporation_id": corporation.ID,
					"faction_id":     faction.ID,
				}}),
		)
		// when
		changed, err := s.updateCharacterESI(ctx, o1.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o2, err := st.GetEveCharacter(ctx, o1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, alliance, o2.Alliance)
				assert.Equal(t, corporation, o2.Corporation)
				assert.Equal(t, faction, o2.Faction)
				assert.Equal(t, "bla bla", o2.Description)
				assert.Equal(t, "CCP Bartender", o2.Name)
				assert.Equal(t, "All round pretty awesome guy", o2.Title)
				assert.InDelta(t, -9.9, o2.SecurityStatus, 0.01)
			}
		}
	})
	t.Run("should report when not changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		alliance := factory.CreateEveEntityAlliance(app.EveEntity{ID: 434243723})
		corporation := factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		faction := factory.CreateEveEntity(app.EveEntity{ID: 500004, Category: app.EveEntityFaction})
		race := factory.CreateEveRace(app.EveRace{ID: 2})
		o1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			AllianceID:     alliance.ID,
			Birthday:       time.Date(2015, 3, 24, 11, 37, 0, 0, time.UTC),
			CorporationID:  corporation.ID,
			Description:    "bla bla",
			FactionID:      faction.ID,
			Gender:         "male",
			Name:           "CCP Bartender",
			RaceID:         race.ID,
			SecurityStatus: -9.9,
			Title:          "All round pretty awesome guy",
		})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: o1.ID})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        "2015-03-24T11:37:00Z",
				"bloodline_id":    3,
				"corporation_id":  109299958,
				"description":     "bla bla",
				"gender":          "male",
				"name":            "CCP Bartender",
				"race_id":         2,
				"security_status": -9.9,
				"title":           "All round pretty awesome guy",
			}))
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/characters/affiliation/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id":    alliance.ID,
					"character_id":   o1.ID,
					"corporation_id": corporation.ID,
					"faction_id":     faction.ID,
				}}),
		)
		// when
		changed, err := s.updateCharacterESI(ctx, o1.ID)
		// then
		if assert.NoError(t, err) {
			assert.False(t, changed)
		}
	})
	t.Run("should remove affiliations", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		o1 := factory.CreateEveCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: o1.ID})
		corporation := factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        "2015-03-24T11:37:00Z",
				"bloodline_id":    3,
				"corporation_id":  109299958,
				"description":     "bla bla",
				"gender":          "male",
				"name":            "CCP Bartender",
				"race_id":         2,
				"security_status": -9.9,
				"title":           "All round pretty awesome guy",
			}))
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/characters/affiliation/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   o1.ID,
					"corporation_id": corporation.ID,
				}}),
		)
		// when
		changed, err := s.updateCharacterESI(ctx, o1.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o2, err := st.GetEveCharacter(ctx, o1.ID)
			if assert.NoError(t, err) {
				assert.Nil(t, o2.Alliance)
				assert.Equal(t, corporation, o2.Corporation)
				assert.Nil(t, o2.Faction)
				assert.Equal(t, "bla bla", o2.Description)
				assert.Equal(t, "CCP Bartender", o2.Name)
				assert.Equal(t, "All round pretty awesome guy", o2.Title)
				assert.InDelta(t, -9.9, o2.SecurityStatus, 0.01)
			}
		}
	})
	t.Run("should delete when no longer exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		o1 := factory.CreateEveCharacter()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(http.StatusNotFound, map[string]any{
				"error": "not found",
			}))
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/characters/affiliation/`,
			httpmock.NewJsonResponderOrPanic(http.StatusNotFound, []map[string]any{
				{
					"error": "not found",
				}}),
		)
		// when
		changed, err := s.updateCharacterESI(ctx, o1.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			_, err := st.GetEveCharacter(ctx, o1.ID)
			assert.ErrorIs(t, err, app.ErrNotFound)
		}
	})
}

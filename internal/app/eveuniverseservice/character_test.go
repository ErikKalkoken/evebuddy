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
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
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
	const invalidID = 666
	t.Run("should return existing character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateEveCharacter()
		// when
		x1, changed, err := s.GetOrCreateCharacterESI(ctx, c.ID)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.False(t, changed)
		assert.Equal(t, c.ID, x1.ID)
	})
	t.Run("should fetch minimal character from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const characterID = 95465499
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
				"corporation_id":  invalidID,
				"gender":          "male",
				"name":            "CCP Bartender",
				"race_id":         2,
				"security_status": -9.9,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/characters/affiliation/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   characterID,
					"corporation_id": 109299958,
				}}),
		)
		// when
		x1, changed, err := s.GetOrCreateCharacterESI(ctx, characterID)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, changed)
		assert.Nil(t, x1.Alliance)
		assert.Nil(t, x1.Faction)
		assert.EqualValues(t, characterID, x1.ID)
		assert.EqualValues(t, time.Date(2015, 03, 24, 11, 37, 0, 0, time.UTC), x1.Birthday)
		assert.EqualValues(t, corporation, x1.Corporation)
		assert.Empty(t, x1.Description)
		assert.EqualValues(t, "male", x1.Gender)
		assert.EqualValues(t, "CCP Bartender", x1.Name)
		assert.EqualValues(t, race, x1.Race)
		assert.Empty(t, x1.Title)
		assert.InDelta(t, -9.9, x1.SecurityStatus, 0.01)
		x2, err := st.GetEveCharacter(ctx, characterID)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.Equal(t, x1, x2)
	})
	t.Run("should fetch full character from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
				"alliance_id":     invalidID,
				"corporation_id":  invalidID,
				"faction_id":      invalidID,
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
		x1, changed, err := s.GetOrCreateCharacterESI(ctx, characterID)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, changed)
		assert.EqualValues(t, characterID, x1.ID)
		assert.EqualValues(t, time.Date(2015, 03, 24, 11, 37, 0, 0, time.UTC), x1.Birthday)
		assert.EqualValues(t, alliance, x1.Alliance)
		assert.EqualValues(t, corporation, x1.Corporation)
		assert.EqualValues(t, faction, x1.Faction)
		assert.EqualValues(t, "bla bla", x1.Description)
		assert.EqualValues(t, "male", x1.Gender)
		assert.EqualValues(t, "CCP Bartender", x1.Name)
		assert.EqualValues(t, race, x1.Race)
		assert.EqualValues(t, "All round pretty awesome guy", x1.Title)
		assert.InDelta(t, -9.9, x1.SecurityStatus, 0.01)
		x2, err := st.GetEveCharacter(ctx, characterID)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.Equal(t, x1, x2)
	})
}

func TestUpdateOrCreateEveCharacterESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.New(eveuniverseservice.Params{
		Storage:   st,
		ESIClient: goesi.NewAPIClient(nil, ""),
	})
	ctx := context.Background()
	const invalidID = 666
	t.Run("should fetch character from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
				"corporation_id":  invalidID,
				"gender":          "male",
				"name":            "CCP Bartender",
				"race_id":         2,
				"security_status": -9.9,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/characters/affiliation/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   characterID,
					"corporation_id": 109299958,
				}}),
		)
		// when
		x1, changed, err := s.UpdateOrCreateCharacterESI(ctx, characterID)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, changed)
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
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.Equal(t, x1, x2)
	})
	t.Run("should update existing character from ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateEveCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: character.ID})
		alliance := factory.CreateEveEntityAlliance()
		corporation2 := factory.CreateEveEntityCorporation()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        character.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":    3,
				"corporation_id":  invalidID,
				"description":     character.Description,
				"gender":          "male",
				"name":            "CCP Bartender",
				"race_id":         character.Race.ID,
				"security_status": -9.9,
				"title":           "super chad",
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/characters/affiliation/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id":    alliance.ID,
					"character_id":   character.ID,
					"corporation_id": corporation2.ID,
				}}),
		)
		// when
		x1, changed, err := s.UpdateOrCreateCharacterESI(ctx, character.ID)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, changed)
		assert.Equal(t, alliance, x1.Alliance)
		assert.Nil(t, x1.Faction)
		assert.Equal(t, corporation2, x1.Corporation)
		assert.Equal(t, "CCP Bartender", x1.Name)
		assert.Equal(t, "super chad", x1.Title)
		assert.InDelta(t, -9.9, x1.SecurityStatus, 0.01)
		x2, err := st.GetEveCharacter(ctx, character.ID)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, x1.Equal(x2), "got %q, wanted %q", x1, x2)
	})
	t.Run("should report when character was not changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			SecurityStatus: -4.12,
		})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: character.ID})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        character.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":    3,
				"corporation_id":  invalidID,
				"description":     character.Description,
				"gender":          character.Gender,
				"name":            character.Name,
				"race_id":         character.Race.ID,
				"security_status": character.SecurityStatus,
				"title":           character.Title,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/characters/affiliation/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   character.ID,
					"corporation_id": character.Corporation.ID,
				}}),
		)
		// when
		_, changed, err := s.UpdateOrCreateCharacterESI(ctx, character.ID)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.False(t, changed)
	})
	t.Run("should report specific error when character does not exist on ESI", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(404, map[string]any{
				"error": "character not found",
			}),
		)
		// when
		_, _, err := s.UpdateOrCreateCharacterESI(ctx, 42)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
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
		testutil.MustTruncateTables(db)
		const characterID = 95465499
		factory.CreateEveEntityCharacter(app.EveEntity{ID: characterID})
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{ID: characterID})
		alliance := factory.CreateEveEntityAlliance()
		corporation := factory.CreateEveEntityCorporation()
		faction := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityFaction})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        ec.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":    3,
				"corporation_id":  corporation.ID,
				"description":     "bla bla",
				"gender":          ec.Gender,
				"name":            "CCP Bartender",
				"race_id":         ec.Race.ID,
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
		got, err := s.UpdateAllCharactersESI(ctx)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		want := set.Of[int32](characterID)
		xassert.EqualSet(t, want, got)
		ec2, err := st.GetEveCharacter(ctx, characterID)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.Equal(t, "CCP Bartender", ec2.Name)
		assert.Equal(t, alliance, ec2.Alliance)
		assert.Equal(t, corporation, ec2.Corporation)
		assert.Equal(t, "bla bla", ec2.Description)
		assert.InDelta(t, -9.9, ec2.SecurityStatus, 0.01)
		assert.Equal(t, "All round pretty awesome guy", ec2.Title)
		ee, err := st.GetEveEntity(ctx, characterID)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.Equal(t, "CCP Bartender", ee.Name)
		assert.Equal(t, app.EveEntityCharacter, ee.Category)
	})
	t.Run("should delete character which no longer exist on ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const characterID = 95465499
		factory.CreateEveEntityCharacter(app.EveEntity{ID: characterID})
		factory.CreateEveCharacter(storage.CreateEveCharacterParams{ID: characterID})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(404, map[string]any{
				"err": "not found",
			}),
		)
		// when
		got, err := s.UpdateAllCharactersESI(ctx)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		want := set.Of[int32](characterID)
		xassert.EqualSet(t, want, got)
		_, err2 := st.GetEveCharacter(ctx, characterID)
		assert.ErrorIs(t, err2, app.ErrNotFound)
	})
}

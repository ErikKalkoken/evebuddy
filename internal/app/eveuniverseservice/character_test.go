package eveuniverseservice_test

import (
	"context"
	"fmt"

	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestGetOrCreateEveCharacterESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
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
		require.NoError(t, err)
		assert.False(t, changed)
		xassert.Equal(t, c.ID, x1.ID)
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
			`=~^https://esi.evetech.net/characters/\d+`,
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
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   characterID,
					"corporation_id": 109299958,
				}}),
		)
		// when
		x1, changed, err := s.GetOrCreateCharacterESI(ctx, characterID)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Empty(t, x1.Alliance)
		xassert.Empty(t, x1.Faction)
		xassert.Equal(t, characterID, x1.ID)
		xassert.Equal(t, time.Date(2015, 03, 24, 11, 37, 0, 0, time.UTC), x1.Birthday)
		xassert.Equal(t, corporation, x1.Corporation)
		assert.Empty(t, x1.Description)
		xassert.Equal(t, "male", x1.Gender)
		xassert.Equal(t, "CCP Bartender", x1.Name)
		xassert.Equal(t, race, x1.Race)
		assert.Empty(t, x1.Title)
		assert.InDelta(t, -9.9, x1.SecurityStatus.ValueOrZero(), 0.01)
		x2, err := st.GetEveCharacter(ctx, characterID)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
	t.Run("should fetch full character from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		characterID := int64(95465499)
		factory.CreateEveEntityCharacter(app.EveEntity{ID: characterID})
		alliance := factory.CreateEveEntityCorporation(app.EveEntity{ID: 434243723})
		corporation := factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		faction := factory.CreateEveEntity(app.EveEntity{ID: 500004, Category: app.EveEntityFaction})
		race := factory.CreateEveRace(app.EveRace{ID: 2})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi.evetech.net/characters/\d+`,
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
			`=~^https://esi.evetech.net/characters/affiliation`,
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
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Equal(t, characterID, x1.ID)
		xassert.Equal(t, time.Date(2015, 03, 24, 11, 37, 0, 0, time.UTC), x1.Birthday)
		xassert.Equal(t, alliance, x1.Alliance.MustValue())
		xassert.Equal(t, corporation, x1.Corporation)
		xassert.Equal(t, faction, x1.Faction.MustValue())
		xassert.Equal(t, "bla bla", x1.Description.ValueOrZero())
		xassert.Equal(t, "male", x1.Gender)
		xassert.Equal(t, "CCP Bartender", x1.Name)
		xassert.Equal(t, race, x1.Race)
		xassert.Equal(t, "All round pretty awesome guy", x1.Title.ValueOrZero())
		assert.InDelta(t, -9.9, x1.SecurityStatus.ValueOrZero(), 0.01)
		x2, err := st.GetEveCharacter(ctx, characterID)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
	t.Run("should return error when called with invalid ID", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		// when
		_, _, err := s.GetOrCreateCharacterESI(ctx, 0)
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}

func TestUpdateOrCreateEveCharacterESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	ctx := context.Background()
	const invalidID = 666
	t.Run("should create new minimal character from ESI with affiliations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := factory.CreateEveEntityCharacter()
		corporation1 := factory.CreateEveEntityCorporation()
		corporation2 := factory.CreateEveEntityCorporation()
		race := factory.CreateEveRace()
		birthday := time.Now().Truncate(time.Second)
		gender := "male"
		name := "CCP Bartender"
		securityStatus := -9.9
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        birthday.Format(time.RFC3339),
				"bloodline_id":    3,
				"corporation_id":  corporation1.ID,
				"gender":          gender,
				"name":            name,
				"race_id":         race.ID,
				"security_status": securityStatus,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   character.ID,
					"corporation_id": corporation2.ID,
				}}),
		)
		// when
		x1, changed, err := s.UpdateOrCreateCharacterESI(ctx, character.ID)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Empty(t, x1.Alliance)
		xassert.Empty(t, x1.Faction)
		xassert.Equal(t, character.ID, x1.ID)
		xassert.Equal(t, birthday, x1.Birthday)
		xassert.Equal(t, corporation2, x1.Corporation)
		assert.Empty(t, x1.Description)
		xassert.Equal(t, gender, x1.Gender)
		xassert.Equal(t, character.ID, x1.ID)
		xassert.Equal(t, name, x1.Name)
		xassert.Equal(t, race, x1.Race)
		assert.Empty(t, x1.Title)
		xassert.Equal(t, securityStatus, x1.SecurityStatus.ValueOrZero())
		x2, err := st.GetEveCharacter(ctx, character.ID)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
	t.Run("should create new full character from ESI with affiliations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := factory.CreateEveEntityCharacter()
		corporation1 := factory.CreateEveEntityCorporation()
		corporation2 := factory.CreateEveEntityCorporation()
		alliance1 := factory.CreateEveEntityAlliance()
		alliance2 := factory.CreateEveEntityAlliance()
		faction1 := factory.CreateEveEntityFaction()
		faction2 := factory.CreateEveEntityFaction()
		race := factory.CreateEveRace()
		birthday := time.Now().Truncate(time.Second)
		gender := "male"
		description := "description"
		name := "CCP Bartender"
		securityStatus := -9.9
		title := "title"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":     alliance1.ID,
				"birthday":        birthday.Format(time.RFC3339),
				"bloodline_id":    3,
				"corporation_id":  corporation1.ID,
				"description":     description,
				"faction_id":      faction1.ID,
				"gender":          gender,
				"name":            name,
				"race_id":         race.ID,
				"security_status": securityStatus,
				"title":           title,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id":    alliance2.ID,
					"character_id":   character.ID,
					"corporation_id": corporation2.ID,
					"faction_id":     faction2.ID,
				}}),
		)
		// when
		x1, changed, err := s.UpdateOrCreateCharacterESI(ctx, character.ID)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Equal(t, alliance2, x1.Alliance.ValueOrZero())
		xassert.Equal(t, birthday, x1.Birthday)
		xassert.Equal(t, corporation2, x1.Corporation)
		xassert.Equal(t, description, x1.Description.ValueOrZero())
		xassert.Equal(t, faction2, x1.Faction.ValueOrZero())
		xassert.Equal(t, gender, x1.Gender)
		xassert.Equal(t, character.ID, x1.ID)
		xassert.Equal(t, name, x1.Name)
		xassert.Equal(t, race, x1.Race)
		xassert.Equal(t, title, x1.Title.ValueOrZero())
		x2, err := st.GetEveCharacter(ctx, character.ID)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
	t.Run("should create new character from ESI and ignore affiliations when they don't match", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := factory.CreateEveEntityCharacter()
		corporation1 := factory.CreateEveEntityCorporation()
		corporation2 := factory.CreateEveEntityCorporation()
		race := factory.CreateEveRace()
		birthday := time.Now().Truncate(time.Second)
		gender := "male"
		name := "CCP Bartender"
		securityStatus := -9.9
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        birthday.Format(time.RFC3339),
				"bloodline_id":    3,
				"corporation_id":  corporation1.ID,
				"gender":          gender,
				"name":            name,
				"race_id":         race.ID,
				"security_status": securityStatus,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   factory.CreateEveEntityCharacter().ID,
					"corporation_id": corporation2.ID,
				}}),
		)
		// when
		x1, changed, err := s.UpdateOrCreateCharacterESI(ctx, character.ID)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Empty(t, x1.Alliance)
		xassert.Empty(t, x1.Faction)
		xassert.Equal(t, character.ID, x1.ID)
		xassert.Equal(t, birthday, x1.Birthday)
		xassert.Equal(t, corporation1, x1.Corporation)
		assert.Empty(t, x1.Description)
		xassert.Equal(t, gender, x1.Gender)
		xassert.Equal(t, character.ID, x1.ID)
		xassert.Equal(t, name, x1.Name)
		xassert.Equal(t, race, x1.Race)
		assert.Empty(t, x1.Title)
		xassert.Equal(t, securityStatus, x1.SecurityStatus.ValueOrZero())
		x2, err := st.GetEveCharacter(ctx, character.ID)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
	t.Run("should create new character from ESI and ignore affiliations when their response is unexpected", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := factory.CreateEveEntityCharacter()
		corporation1 := factory.CreateEveEntityCorporation()
		corporation2 := factory.CreateEveEntityCorporation()
		race := factory.CreateEveRace()
		birthday := time.Now().Truncate(time.Second)
		gender := "male"
		name := "CCP Bartender"
		securityStatus := -9.9
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        birthday.Format(time.RFC3339),
				"bloodline_id":    3,
				"corporation_id":  corporation1.ID,
				"gender":          gender,
				"name":            name,
				"race_id":         race.ID,
				"security_status": securityStatus,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"character_id":   factory.CreateEveEntityCharacter().ID,
				"corporation_id": corporation2.ID,
			}, {
				"character_id":   666,
				"corporation_id": factory.CreateEveEntityCorporation().ID,
			}}),
		)
		// when
		x1, changed, err := s.UpdateOrCreateCharacterESI(ctx, character.ID)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Empty(t, x1.Alliance)
		xassert.Empty(t, x1.Faction)
		xassert.Equal(t, character.ID, x1.ID)
		xassert.Equal(t, birthday, x1.Birthday)
		xassert.Equal(t, corporation1, x1.Corporation)
		assert.Empty(t, x1.Description)
		xassert.Equal(t, gender, x1.Gender)
		xassert.Equal(t, character.ID, x1.ID)
		xassert.Equal(t, name, x1.Name)
		xassert.Equal(t, race, x1.Race)
		assert.Empty(t, x1.Title)
		xassert.Equal(t, securityStatus, x1.SecurityStatus.ValueOrZero())
		x2, err := st.GetEveCharacter(ctx, character.ID)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
	t.Run("should update existing character from ESI with affiliations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := factory.CreateEveCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: character.ID})
		alliance2 := factory.CreateEveEntityAlliance()
		corporation2 := factory.CreateEveEntityCorporation()
		faction2 := factory.CreateEveEntityFaction()
		description := "description"
		name2 := "CCP Bartender"
		gender := "male"
		securityStatus2 := -9.9
		title2 := "super chad"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        character.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":    3,
				"corporation_id":  character.Corporation.ID,
				"description":     description,
				"gender":          gender,
				"name":            name2,
				"race_id":         character.Race.ID,
				"security_status": securityStatus2,
				"title":           title2,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id":    alliance2.ID,
					"character_id":   character.ID,
					"corporation_id": corporation2.ID,
					"faction_id":     faction2.ID,
				}}),
		)
		// when
		x1, changed, err := s.UpdateOrCreateCharacterESI(ctx, character.ID)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Equal(t, alliance2, x1.Alliance.ValueOrZero())
		xassert.Equal(t, corporation2, x1.Corporation)
		xassert.Equal(t, description, x1.Description.ValueOrZero())
		xassert.Equal(t, faction2, x1.Faction.ValueOrZero())
		xassert.Equal(t, name2, x1.Name)
		xassert.Equal(t, securityStatus2, x1.SecurityStatus.ValueOrZero())
		xassert.Equal(t, title2, x1.Title.ValueOrZero())
		x2, err := st.GetEveCharacter(ctx, character.ID)
		require.NoError(t, err)
		assert.True(t, x1.Equal(x2), "got %q, wanted %q", x1, x2)
	})
	t.Run("should update existing character from ESI but keep affiliations when affiliations response is invalid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := factory.CreateEveCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: character.ID})
		alliance2 := factory.CreateEveEntityAlliance()
		corporation2 := factory.CreateEveEntityCorporation()
		faction2 := factory.CreateEveEntityFaction()
		alliance3 := factory.CreateEveEntityAlliance()
		corporation3 := factory.CreateEveEntityCorporation()
		faction3 := factory.CreateEveEntityFaction()
		description := "description"
		name2 := "CCP Bartender"
		gender := "male"
		securityStatus2 := -9.9
		title2 := "super chad"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":     alliance3.ID,
				"birthday":        character.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":    3,
				"corporation_id":  corporation3.ID,
				"description":     description,
				"faction_id":      faction3.ID,
				"gender":          gender,
				"name":            name2,
				"race_id":         character.Race.ID,
				"security_status": securityStatus2,
				"title":           title2,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id":    alliance2.ID,
					"character_id":   factory.CreateEveEntityCharacter().ID,
					"corporation_id": corporation2.ID,
					"faction_id":     faction2.ID,
				}}),
		)
		// when
		x1, changed, err := s.UpdateOrCreateCharacterESI(ctx, character.ID)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Equal(t, character.Alliance, x1.Alliance)
		xassert.Equal(t, character.Corporation, x1.Corporation)
		xassert.Equal(t, description, x1.Description.ValueOrZero())
		xassert.Equal(t, character.Faction, x1.Faction)
		xassert.Equal(t, name2, x1.Name)
		xassert.Equal(t, title2, x1.Title.ValueOrZero())
		xassert.Equal(t, securityStatus2, x1.SecurityStatus.ValueOrZero())
		x2, err := st.GetEveCharacter(ctx, character.ID)
		require.NoError(t, err)
		assert.True(t, x1.Equal(x2), "got %q, wanted %q", x1, x2)
	})
	t.Run("should report when character was not changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := factory.CreateEveCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: character.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
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
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   character.ID,
					"corporation_id": character.Corporation.ID,
				}}),
		)
		// when
		_, changed, err := s.UpdateOrCreateCharacterESI(ctx, character.ID)
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})
	t.Run("should report character as unchanged when falling back to original affiliations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := factory.CreateEveCharacter()
		corporation2 := factory.CreateEveEntityCorporation()
		corporation3 := factory.CreateEveEntityCorporation()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: character.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        character.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":    3,
				"corporation_id":  corporation3.ID,
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
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   factory.CreateEveEntityCharacter().ID,
					"corporation_id": corporation2.ID,
				}}),
		)
		// when
		_, changed, err := s.UpdateOrCreateCharacterESI(ctx, character.ID)
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})
	t.Run("should report specific error when character does not exist on ESI", func(t *testing.T) {
		// given
		httpmock.Reset()
		const characterID = 42
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", characterID),
			httpmock.NewJsonResponderOrPanic(404, map[string]any{
				"error": "character not found",
			}),
		)
		// when
		_, _, err := s.UpdateOrCreateCharacterESI(ctx, characterID)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return error when called with invalid ID", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		// when
		_, _, err := s.UpdateOrCreateCharacterESI(ctx, 0)
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}

func TestUpdateAllEveCharactersESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
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
			`=~^https://esi.evetech.net/characters/\d+`,
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
			`=~^https://esi.evetech.net/characters/affiliation`,
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
		require.NoError(t, err)
		want := set.Of[int64](characterID)
		xassert.Equal(t, want, got)
		ec2, err := st.GetEveCharacter(ctx, characterID)
		require.NoError(t, err)
		xassert.Equal(t, "CCP Bartender", ec2.Name)
		xassert.Equal(t, alliance, ec2.Alliance.MustValue())
		xassert.Equal(t, corporation, ec2.Corporation)
		xassert.Equal(t, "bla bla", ec2.Description.ValueOrZero())
		assert.InDelta(t, -9.9, ec2.SecurityStatus.ValueOrZero(), 0.01)
		xassert.Equal(t, "All round pretty awesome guy", ec2.Title.ValueOrZero())
		ee, err := st.GetEveEntity(ctx, characterID)
		require.NoError(t, err)
		xassert.Equal(t, "CCP Bartender", ee.Name)
		xassert.Equal(t, app.EveEntityCharacter, ee.Category)
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
			`=~^https://esi.evetech.net/characters/\d+`,
			httpmock.NewJsonResponderOrPanic(404, map[string]any{
				"err": "not found",
			}),
		)
		// when
		got, err := s.UpdateAllCharactersESI(ctx)
		// then
		require.NoError(t, err)
		want := set.Of[int64](characterID)
		xassert.Equal(t, want, got)
		_, err2 := st.GetEveCharacter(ctx, characterID)
		assert.ErrorIs(t, err2, app.ErrNotFound)
	})
}

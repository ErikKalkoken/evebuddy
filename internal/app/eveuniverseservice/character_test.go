package eveuniverseservice_test

import (
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
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	const invalidID = 666

	t.Run("should return existing character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c1 := f.CreateEveCharacter()

		// when
		c2, changed, err := s.GetOrCreateCharacterESI(t.Context(), c1.ID)

		// then
		require.NoError(t, err)
		assert.False(t, changed)
		xassert.Equal(t, c1.ID, c2.ID)
	})

	t.Run("should fetch minimal character from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const characterID = 95465499
		f.CreateEveEntityCharacter(app.EveEntity{ID: characterID})
		corporation := f.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		bloodline := f.CreateEveBloodline()
		race := bloodline.Race
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi.evetech.net/characters/\d+`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"achievement_score": 1234,
				"birthday":          "2015-03-24T11:37:00Z",
				"bloodline_id":      bloodline.ID,
				"corporation_id":    invalidID,
				"gender":            "male",
				"name":              "CCP Bartender",
				"race_id":           race.ID,
				"security_status":   -9.9,
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
		x1, changed, err := s.GetOrCreateCharacterESI(t.Context(), characterID)

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
		xassert.Equal(t, bloodline.ID, x1.Bloodline.MustValue().ID)
		xassert.Equal(t, race, x1.Race)
		assert.Empty(t, x1.CorporationTitle)
		assert.InDelta(t, -9.9, x1.SecurityStatus.ValueOrZero(), 0.01)
		x2, err := st.GetEveCharacter(t.Context(), characterID)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})

	t.Run("should fetch full character from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		characterID := int64(95465499)
		f.CreateEveEntityCharacter(app.EveEntity{ID: characterID})
		alliance := f.CreateEveEntityCorporation(app.EveEntity{ID: 434243723})
		corporation := f.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		faction := f.CreateEveEntity(app.EveEntity{ID: 500004, Category: app.EveEntityFaction})
		bloodline := f.CreateEveBloodline()
		race := bloodline.Race
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi.evetech.net/characters/\d+`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"achievement_score": 1234,
				"birthday":          "2015-03-24T11:37:00Z",
				"bloodline_id":      bloodline.ID,
				"alliance_id":       invalidID,
				"corporation_id":    invalidID,
				"faction_id":        invalidID,
				"description":       "bla bla",
				"gender":            "male",
				"name":              "CCP Bartender",
				"race_id":           race.ID,
				"security_status":   -9.9,
				"corporation_title": "All round pretty awesome guy",
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
		x1, changed, err := s.GetOrCreateCharacterESI(t.Context(), characterID)

		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Equal(t, characterID, x1.ID)
		xassert.Equal(t, time.Date(2015, 03, 24, 11, 37, 0, 0, time.UTC), x1.Birthday)
		xassert.EqualOptional(t, alliance, x1.Alliance)
		xassert.Equal(t, corporation, x1.Corporation)
		xassert.EqualOptional(t, faction, x1.Faction)
		xassert.Equal(t, "bla bla", x1.Description.ValueOrZero())
		xassert.Equal(t, "male", x1.Gender)
		xassert.Equal(t, "CCP Bartender", x1.Name)
		xassert.Equal(t, race, x1.Race)
		xassert.Equal(t, bloodline.ID, x1.Bloodline.MustValue().ID)
		xassert.Equal(t, "All round pretty awesome guy", x1.CorporationTitle.ValueOrZero())
		assert.InDelta(t, -9.9, x1.SecurityStatus.ValueOrZero(), 0.01)
		x2, err := st.GetEveCharacter(t.Context(), characterID)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})

	t.Run("should return error when called with invalid ID", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		// when
		_, _, err := s.GetOrCreateCharacterESI(t.Context(), 0)
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}

func TestUpdateOrCreateEveCharacterESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	const invalidID = 666

	t.Run("should create new minimal character from ESI with affiliations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := f.CreateEveEntityCharacter()
		corporation1 := f.CreateEveEntityCorporation()
		corporation2 := f.CreateEveEntityCorporation()
		bloodline := f.CreateEveBloodline()
		race := bloodline.Race
		birthday := time.Now().Truncate(time.Second)
		gender := "male"
		name := "CCP Bartender"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"achievement_score": 1234,
				"birthday":          birthday.Format(time.RFC3339),
				"bloodline_id":      bloodline.ID,
				"corporation_id":    corporation1.ID,
				"gender":            gender,
				"name":              name,
				"race_id":           race.ID,
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
		o, changed, err := s.UpdateOrCreateCharacterESI(t.Context(), character.ID)

		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Empty(t, o.Alliance)
		xassert.Empty(t, o.Faction)
		xassert.Equal(t, character.ID, o.ID)
		xassert.Equal(t, birthday, o.Birthday)
		xassert.Equal(t, corporation2, o.Corporation)
		assert.Empty(t, o.Description)
		xassert.Equal(t, gender, o.Gender)
		xassert.Equal(t, character.ID, o.ID)
		xassert.Equal(t, name, o.Name)
		xassert.Equal(t, race, o.Race)
		xassert.EqualOptional(t, eveBloodlineToEntityShort(bloodline), o.Bloodline)
		assert.Empty(t, o.CorporationTitle)
		// assert.Empty(t, o.SecurityStatus)
		x2, err := st.GetEveCharacter(t.Context(), character.ID)
		require.NoError(t, err)
		xassert.Equal(t, o, x2)
	})

	t.Run("should create new full character from ESI with affiliations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := f.CreateEveEntityCharacter()
		corporation1 := f.CreateEveEntityCorporation()
		corporation2 := f.CreateEveEntityCorporation()
		alliance1 := f.CreateEveEntityAlliance()
		alliance2 := f.CreateEveEntityAlliance()
		faction1 := f.CreateEveEntityFaction()
		faction2 := f.CreateEveEntityFaction()
		bloodline := f.CreateEveBloodline()
		race := bloodline.Race
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
				"achievement_score": 1234,
				"alliance_id":       alliance1.ID,
				"birthday":          birthday.Format(time.RFC3339),
				"bloodline_id":      bloodline.ID,
				"corporation_id":    corporation1.ID,
				"corporation_title": title,
				"description":       description,
				"faction_id":        faction1.ID,
				"gender":            gender,
				"name":              name,
				"race_id":           race.ID,
				"security_status":   securityStatus,
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
		x1, changed, err := s.UpdateOrCreateCharacterESI(t.Context(), character.ID)
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
		xassert.Equal(t, bloodline.ID, x1.Bloodline.MustValue().ID)
		xassert.Equal(t, title, x1.CorporationTitle.ValueOrZero())
		x2, err := st.GetEveCharacter(t.Context(), character.ID)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})

	t.Run("should create new minimal character from ESI with neutral security status", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := f.CreateEveEntityCharacter()
		corporation := f.CreateEveEntityCorporation()
		bloodline := f.CreateEveBloodline()
		race := bloodline.Race
		birthday := time.Now().Truncate(time.Second)
		gender := "male"
		name := "CCP Bartender"
		securityStatus := float64(0)

		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"achievement_score": 1234,
				"birthday":          birthday.Format(time.RFC3339),
				"bloodline_id":      bloodline.ID,
				"corporation_id":    corporation.ID,
				"gender":            gender,
				"name":              name,
				"race_id":           race.ID,
				"security_status":   securityStatus,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   character.ID,
					"corporation_id": corporation.ID,
				}}),
		)

		// when
		o, changed, err := s.UpdateOrCreateCharacterESI(t.Context(), character.ID)

		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Empty(t, o.Alliance)
		xassert.Empty(t, o.Faction)
		xassert.Equal(t, character.ID, o.ID)
		xassert.Equal(t, birthday, o.Birthday)
		xassert.Equal(t, corporation, o.Corporation)
		assert.Empty(t, o.Description)
		xassert.Equal(t, gender, o.Gender)
		xassert.Equal(t, character.ID, o.ID)
		xassert.Equal(t, name, o.Name)
		xassert.Equal(t, race, o.Race)
		xassert.EqualOptional(t, eveBloodlineToEntityShort(bloodline), o.Bloodline)
		assert.Empty(t, o.CorporationTitle)
		if assert.NotEmpty(t, o.SecurityStatus) {
			xassert.EqualOptional(t, securityStatus, o.SecurityStatus)
		}
		x2, err := st.GetEveCharacter(t.Context(), character.ID)
		require.NoError(t, err)
		xassert.Equal(t, o, x2)
	})

	t.Run("should create new character from ESI and ignore affiliations when they don't match", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := f.CreateEveEntityCharacter()
		corporation1 := f.CreateEveEntityCorporation()
		corporation2 := f.CreateEveEntityCorporation()
		bloodline := f.CreateEveBloodline()
		race := bloodline.Race
		birthday := time.Now().Truncate(time.Second)
		gender := "male"
		name := "CCP Bartender"
		securityStatus := -9.9
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"achievement_score": 1234,
				"birthday":          birthday.Format(time.RFC3339),
				"bloodline_id":      bloodline.ID,
				"corporation_id":    corporation1.ID,
				"gender":            gender,
				"name":              name,
				"race_id":           race.ID,
				"security_status":   securityStatus,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   f.CreateEveEntityCharacter().ID,
					"corporation_id": corporation2.ID,
				}}),
		)
		// when
		x1, changed, err := s.UpdateOrCreateCharacterESI(t.Context(), character.ID)
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
		assert.Empty(t, x1.CorporationTitle)
		xassert.Equal(t, securityStatus, x1.SecurityStatus.ValueOrZero())
		x2, err := st.GetEveCharacter(t.Context(), character.ID)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})

	t.Run("should create new character from ESI and ignore affiliations when their response is unexpected", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := f.CreateEveEntityCharacter()
		corporation1 := f.CreateEveEntityCorporation()
		corporation2 := f.CreateEveEntityCorporation()
		bloodline := f.CreateEveBloodline()
		race := bloodline.Race
		birthday := time.Now().Truncate(time.Second)
		gender := "male"
		name := "CCP Bartender"
		securityStatus := -9.9
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"achievement_score": 1234,
				"birthday":          birthday.Format(time.RFC3339),
				"bloodline_id":      bloodline.ID,
				"corporation_id":    corporation1.ID,
				"gender":            gender,
				"name":              name,
				"race_id":           race.ID,
				"security_status":   securityStatus,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"character_id":   f.CreateEveEntityCharacter().ID,
				"corporation_id": corporation2.ID,
			}, {
				"character_id":   666,
				"corporation_id": f.CreateEveEntityCorporation().ID,
			}}),
		)

		// when
		x1, changed, err := s.UpdateOrCreateCharacterESI(t.Context(), character.ID)

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
		assert.Empty(t, x1.CorporationTitle)
		xassert.Equal(t, securityStatus, x1.SecurityStatus.ValueOrZero())
		x2, err := st.GetEveCharacter(t.Context(), character.ID)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})

	t.Run("should update existing character from ESI with affiliations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := f.CreateEveCharacter()
		f.CreateEveEntityCharacter(app.EveEntity{ID: character.ID})
		alliance2 := f.CreateEveEntityAlliance()
		corporation2 := f.CreateEveEntityCorporation()
		faction2 := f.CreateEveEntityFaction()
		description := "description"
		name2 := "CCP Bartender"
		gender := "male"
		securityStatus2 := -9.9
		title2 := "super chad"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"achievement_score": 1234,
				"birthday":          character.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":      character.Bloodline.MustValue().ID,
				"corporation_id":    character.Corporation.ID,
				"corporation_title": title2,
				"description":       description,
				"gender":            gender,
				"name":              name2,
				"race_id":           character.Race.ID,
				"security_status":   securityStatus2,
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
		x1, changed, err := s.UpdateOrCreateCharacterESI(t.Context(), character.ID)

		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Equal(t, alliance2, x1.Alliance.ValueOrZero())
		xassert.Equal(t, corporation2, x1.Corporation)
		xassert.Equal(t, description, x1.Description.ValueOrZero())
		xassert.Equal(t, faction2, x1.Faction.ValueOrZero())
		xassert.Equal(t, name2, x1.Name)
		xassert.Equal(t, securityStatus2, x1.SecurityStatus.ValueOrZero())
		xassert.Equal(t, title2, x1.CorporationTitle.ValueOrZero())
		character2, err := st.GetEveCharacter(t.Context(), character.ID)
		require.NoError(t, err)
		assert.True(t, x1.Equal(character2), "got %q, wanted %q", x1, character2)
	})

	t.Run("should update existing character from ESI but keep affiliations when affiliations response is invalid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := f.CreateEveCharacter()
		f.CreateEveEntityCharacter(app.EveEntity{ID: character.ID})
		alliance2 := f.CreateEveEntityAlliance()
		corporation2 := f.CreateEveEntityCorporation()
		faction2 := f.CreateEveEntityFaction()
		alliance3 := f.CreateEveEntityAlliance()
		corporation3 := f.CreateEveEntityCorporation()
		faction3 := f.CreateEveEntityFaction()
		description := "description"
		name2 := "CCP Bartender"
		gender := "male"
		securityStatus2 := -9.9
		title2 := "super chad"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"achievement_score": 1234,
				"alliance_id":       alliance3.ID,
				"birthday":          character.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":      character.Bloodline.MustValue().ID,
				"corporation_id":    corporation3.ID,
				"corporation_title": title2,
				"description":       description,
				"faction_id":        faction3.ID,
				"gender":            gender,
				"name":              name2,
				"race_id":           character.Race.ID,
				"security_status":   securityStatus2,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id":    alliance2.ID,
					"character_id":   f.CreateEveEntityCharacter().ID,
					"corporation_id": corporation2.ID,
					"faction_id":     faction2.ID,
				}}),
		)

		// when
		got, changed, err := s.UpdateOrCreateCharacterESI(t.Context(), character.ID)

		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Equal(t, character.Alliance, got.Alliance)
		xassert.Equal(t, character.Corporation, got.Corporation)
		xassert.Equal(t, description, got.Description.ValueOrZero())
		xassert.Equal(t, character.Faction, got.Faction)
		xassert.Equal(t, name2, got.Name)
		xassert.Equal(t, title2, got.CorporationTitle.ValueOrZero())
		xassert.Equal(t, securityStatus2, got.SecurityStatus.ValueOrZero())
		character2, err := st.GetEveCharacter(t.Context(), character.ID)
		require.NoError(t, err)
		assert.True(t, got.Equal(character2), "got %q, wanted %q", got, character2)
	})

	t.Run("should report when character was not changed and return unchanged character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := f.CreateEveCharacter()
		f.CreateEveEntityCharacter(app.EveEntity{ID: character.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"achievement_score": 1234,
				"birthday":          character.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":      character.Bloodline.MustValue().ID,
				"corporation_id":    invalidID,
				"corporation_title": character.CorporationTitle,
				"description":       character.Description,
				"gender":            character.Gender,
				"name":              character.Name,
				"race_id":           character.Race.ID,
				"security_status":   character.SecurityStatus,
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
		got, changed, err := s.UpdateOrCreateCharacterESI(t.Context(), character.ID)

		// then
		require.NoError(t, err)
		assert.False(t, changed)
		xassert.Equal(t, character, got)
	})

	t.Run("should report character as unchanged when falling back to original affiliations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		character := f.CreateEveCharacter()
		corporation2 := f.CreateEveEntityCorporation()
		corporation3 := f.CreateEveEntityCorporation()
		f.CreateEveEntityCharacter(app.EveEntity{ID: character.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d", character.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"achievement_score": 1234,
				"birthday":          character.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":      character.Bloodline.MustValue().ID,
				"corporation_id":    corporation3.ID,
				"corporation_title": character.CorporationTitle,
				"description":       character.Description,
				"gender":            character.Gender,
				"name":              character.Name,
				"race_id":           character.Race.ID,
				"security_status":   character.SecurityStatus,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi.evetech.net/characters/affiliation`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   f.CreateEveEntityCharacter().ID,
					"corporation_id": corporation2.ID,
				}}),
		)

		// when
		got, changed, err := s.UpdateOrCreateCharacterESI(t.Context(), character.ID)

		// then
		require.NoError(t, err)
		assert.False(t, changed)
		xassert.Equal(t, character, got)
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
		_, _, err := s.UpdateOrCreateCharacterESI(t.Context(), characterID)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})

	t.Run("should return error when called with invalid ID", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		// when
		_, _, err := s.UpdateOrCreateCharacterESI(t.Context(), 0)
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
				"achievement_score": 1234,
				"birthday":          ec.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":      ec.Bloodline.MustValue().ID,
				"corporation_id":    corporation.ID,
				"corporation_title": "All round pretty awesome guy",
				"description":       "bla bla",
				"gender":            ec.Gender,
				"name":              "CCP Bartender",
				"race_id":           ec.Race.ID,
				"security_status":   -9.9,
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
		got, err := s.UpdateAllCharactersESI(t.Context())

		// then
		require.NoError(t, err)
		want := set.Of[int64](characterID)
		xassert.Equal(t, want, got)
		ec2, err := st.GetEveCharacter(t.Context(), characterID)
		require.NoError(t, err)
		xassert.Equal(t, "CCP Bartender", ec2.Name)
		xassert.EqualOptional(t, alliance, ec2.Alliance)
		xassert.Equal(t, corporation, ec2.Corporation)
		xassert.Equal(t, "bla bla", ec2.Description.ValueOrZero())
		assert.InDelta(t, -9.9, ec2.SecurityStatus.ValueOrZero(), 0.01)
		xassert.Equal(t, "All round pretty awesome guy", ec2.CorporationTitle.ValueOrZero())
		ee, err := st.GetEveEntity(t.Context(), characterID)
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
		got, err := s.UpdateAllCharactersESI(t.Context())

		// then
		require.NoError(t, err)
		want := set.Of[int64](characterID)
		xassert.Equal(t, want, got)
		_, err2 := st.GetEveCharacter(t.Context(), characterID)
		assert.ErrorIs(t, err2, app.ErrNotFound)
	})
}

func eveBloodlineToEntityShort(eb *app.EveBloodline) *app.EntityShort {
	if eb == nil {
		return nil
	}
	o := &app.EntityShort{
		ID:   eb.ID,
		Name: eb.Name,
	}
	return o
}

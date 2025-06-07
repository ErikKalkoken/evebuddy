package eveuniverseservice_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestFetchAlliance(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.New(eveuniverseservice.Params{
		Storage:   st,
		ESIClient: goesi.NewAPIClient(nil, ""),
	})
	const allianceID = 434243723
	factory.CreateEveEntityAlliance(app.EveEntity{ID: allianceID})
	creator := factory.CreateEveEntityCharacter(app.EveEntity{ID: 12345})
	creatorCorp := factory.CreateEveEntityCorporation(app.EveEntity{ID: 45678})
	executor := factory.CreateEveEntityCorporation(app.EveEntity{ID: 98356193})
	ctx := context.Background()
	t.Run("should return complete alliance", func(t *testing.T) {
		// given
		faction := factory.CreateEveEntity(app.EveEntity{ID: 888, Category: app.EveEntityFaction})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/alliances/%d/", allianceID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"creator_corporation_id":  45678,
				"creator_id":              12345,
				"faction_id":              888,
				"date_founded":            "2016-06-26T21:00:00Z",
				"executor_corporation_id": 98356193,
				"name":                    "C C P Alliance",
				"ticker":                  "<C C P>",
			}),
		)
		// when
		x, err := s.FetchAlliance(ctx, allianceID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "C C P Alliance", x.Name)
			assert.Equal(t, "<C C P>", x.Ticker)
			assert.Equal(t, creator, x.Creator)
			assert.Equal(t, creatorCorp, x.CreatorCorporation)
			assert.Equal(t, executor, x.ExecutorCorporation)
			assert.Equal(t, faction, x.Faction)
			assert.Equal(t, time.Date(2016, 6, 26, 21, 0, 0, 0, time.UTC), x.DateFounded)
		}
	})
	t.Run("should return nil for undefined entities", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/alliances/%d/", allianceID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"creator_corporation_id":  45678,
				"creator_id":              12345,
				"date_founded":            "2016-06-26T21:00:00Z",
				"executor_corporation_id": 98356193,
				"name":                    "C C P Alliance",
				"ticker":                  "<C C P>",
			}),
		)
		// when
		x, err := s.FetchAlliance(ctx, allianceID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "C C P Alliance", x.Name)
			assert.Nil(t, x.Faction)
		}
	})
}

func TestFetchAllianceCorporations(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.New(eveuniverseservice.Params{
		Storage:   st,
		ESIClient: goesi.NewAPIClient(nil, ""),
	})
	ctx := context.Background()
	t.Run("should return corporations", func(t *testing.T) {
		// given
		const allianceID = 42
		testutil.TruncateTables(db)
		factory.CreateEveEntityAlliance(app.EveEntity{ID: allianceID})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 101})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 102, Name: "Bravo"})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 103, Name: "Alpha"})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/alliances/%d/corporations/", allianceID),
			httpmock.NewJsonResponderOrPanic(200, []int32{102, 103}),
		)
		// when
		oo, err := s.FetchAllianceCorporations(ctx, allianceID)
		// then
		if assert.NoError(t, err) {
			got := xslices.Map(oo, func(a *app.EveEntity) int32 {
				return a.ID
			})
			want := []int32{103, 102}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return empty list when there are no corporations", func(t *testing.T) {
		// given
		const allianceID = 42
		testutil.TruncateTables(db)
		factory.CreateEveEntityAlliance(app.EveEntity{ID: allianceID})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/alliances/%d/corporations/", allianceID),
			httpmock.NewJsonResponderOrPanic(200, []int32{}),
		)
		// when
		oo, err := s.FetchAllianceCorporations(ctx, allianceID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 0)
		}
	})
}

func TestGetOrCreateEveCharacterESI(t *testing.T) {
	db, st, factory := testutil.New()
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
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/", characterID),
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
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/", characterID),
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
	db, st, factory := testutil.New()
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
			x, err := st.GetEveCharacter(ctx, characterID)
			if assert.NoError(t, err) {
				assert.Equal(t, alliance, x.Alliance)
				assert.Equal(t, corporation, x.Corporation)
				assert.Equal(t, "bla bla", x.Description)
				assert.InDelta(t, -9.9, x.SecurityStatus, 0.01)
				assert.Equal(t, "All round pretty awesome guy", x.Title)
			}
		}
	})
}

func TestGetOrCreateEveCorporationESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should create new corporation", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		alliance := factory.CreateEveEntityAlliance(app.EveEntity{ID: 434243723})
		faction := factory.CreateEveEntity(app.EveEntity{ID: 123, Category: app.EveEntityFaction})
		station := factory.CreateEveEntity(app.EveEntity{ID: 456, Category: app.EveEntityStation})
		ceo := factory.CreateEveEntityCharacter(app.EveEntity{ID: 180548812})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":     434243723,
				"ceo_id":          180548812,
				"creator_id":      180548812,
				"date_founded":    "2004-11-28T16:42:51Z",
				"description":     "This is a corporation description, it's basically just a string",
				"faction_id":      123,
				"home_station_id": 456,
				"member_count":    656,
				"name":            "C C P",
				"tax_rate":        0.256,
				"ticker":          "-CCP-",
				"url":             "http://www.eveonline.com",
			}),
		)
		// when
		o, err := s.GetOrCreateCorporationESI(ctx, 109299958)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, alliance, o.Alliance)
			assert.Equal(t, ceo, o.Creator)
			assert.Equal(t, ceo, o.Ceo)
			assert.Equal(t, time.Date(2004, 11, 28, 16, 42, 51, 0, time.UTC), o.DateFounded.MustValue().UTC())
			assert.Equal(t, "This is a corporation description, it's basically just a string", o.Description)
			assert.Equal(t, faction, o.Faction)
			assert.Equal(t, station, o.HomeStation)
			assert.Equal(t, 656, o.MemberCount)
			assert.Equal(t, "C C P", o.Name)
			assert.Equal(t, float32(0.256), o.TaxRate)
			assert.Equal(t, "-CCP-", o.Ticker)
			assert.Equal(t, "http://www.eveonline.com", o.URL)
		}
	})
	t.Run("can handle no CEO and no creator", func(t *testing.T) {
		// given
		const corporationID = 666
		testutil.TruncateTables(db)
		factory.CreateEveEntityCorporation(app.EveEntity{ID: corporationID})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"ceo_id":       1,
				"creator_id":   1,
				"date_founded": "2004-11-28T16:42:51Z",
				"description":  "This is a corporation description, it's basically just a string",
				"member_count": 656,
				"name":         "C C P",
				"tax_rate":     0.256,
				"ticker":       "-CCP-",
			}),
		)
		// when
		o, err := s.GetOrCreateCorporationESI(ctx, corporationID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, time.Date(2004, 11, 28, 16, 42, 51, 0, time.UTC), o.DateFounded.MustValue().UTC())
			assert.Equal(t, "This is a corporation description, it's basically just a string", o.Description)
			assert.Equal(t, 656, o.MemberCount)
			assert.Equal(t, "C C P", o.Name)
			assert.Equal(t, float32(0.256), o.TaxRate)
			assert.Equal(t, "-CCP-", o.Ticker)
			assert.Nil(t, o.Ceo)
			assert.Nil(t, o.Creator)
			assert.Nil(t, o.Alliance)
			assert.Nil(t, o.Faction)
		}
	})
}
func TestGetOrCreateEveSchematicESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing schematic", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		x1 := factory.CreateEveSchematic()
		// when
		x2, err := s.GetOrCreateSchematicESI(ctx, x1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
	t.Run("should fetch schematic from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/schematics/3/",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"cycle_time":     1800,
				"schematic_name": "Bacteria",
			}))

		// when
		x1, err := s.GetOrCreateSchematicESI(ctx, 3)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(3), x1.ID)
			assert.Equal(t, "Bacteria", x1.Name)
			assert.Equal(t, 1800, x1.CycleTime)
			x2, err := st.GetEveSchematic(ctx, 3)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

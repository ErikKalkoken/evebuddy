package eveuniverseservice_test

import (
	"context"
	"fmt"
	"net/http"
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

func TestGetEveAllianceCorporationsESI(t *testing.T) {
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
		oo, err := s.GetAllianceCorporationsESI(ctx, allianceID)
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
		oo, err := s.GetAllianceCorporationsESI(ctx, allianceID)
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
			x2, err := st.GetEveCharacter(ctx, characterID)
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
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
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
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, dataCharacter),
		)
		dataAffiliation := []map[string]any{
			{
				"alliance_id":    434243723,
				"character_id":   95465499,
				"corporation_id": 109299958,
			}}
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/characters/affiliation/`,
			httpmock.NewJsonResponderOrPanic(200, dataAffiliation),
		)
		// when
		err := s.UpdateAllCharactersESI(ctx)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetEveCharacter(ctx, characterID)
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

func TestGetEveCorporationESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return corporation", func(t *testing.T) {
		// given
		const corporationID = 109299958
		testutil.TruncateTables(db)
		factory.CreateEveEntityCorporation(app.EveEntity{ID: corporationID})
		alliance := factory.CreateEveEntityAlliance(app.EveEntity{ID: 434243723})
		faction := factory.CreateEveEntity(app.EveEntity{ID: 123, Category: app.EveEntityFaction})
		station := factory.CreateEveEntity(app.EveEntity{ID: 456, Category: app.EveEntityStation})
		ceo := factory.CreateEveEntityCharacter(app.EveEntity{ID: 180548812})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/corporations/%d/", corporationID),
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
		o, err := s.GetCorporationESI(ctx, corporationID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, alliance, o.Alliance)
			assert.Equal(t, ceo, o.Creator)
			assert.Equal(t, ceo, o.Ceo)
			assert.Equal(t, time.Date(2004, 11, 28, 16, 42, 51, 0, time.UTC), o.DateFounded.UTC())
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
			fmt.Sprintf("https://esi.evetech.net/v5/corporations/%d/", corporationID),
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
		o, err := s.GetCorporationESI(ctx, corporationID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, time.Date(2004, 11, 28, 16, 42, 51, 0, time.UTC), o.DateFounded.UTC())
			assert.Equal(t, "This is a corporation description, it's basically just a string", o.Description)
			assert.Equal(t, 656, o.MemberCount)
			assert.Equal(t, "C C P", o.Name)
			assert.Equal(t, float32(0.256), o.TaxRate)
			assert.Equal(t, "-CCP-", o.Ticker)
		}
	})
}

func TestGetOrCreateEveDogmaAttributeESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing object", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		x1 := factory.CreateEveDogmaAttribute()
		// when
		x2, err := s.GetOrCreateDogmaAttributeESI(ctx, x1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x2, x1)
		}
	})
	t.Run("should create new object from ESI when it does not exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/dogma/attributes/20/",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"attribute_id":  20,
				"default_value": 1,
				"description":   "Factor by which top speed increases.",
				"display_name":  "Maximum Velocity Bonus",
				"high_is_good":  true,
				"icon_id":       1389,
				"name":          "speedFactor",
				"published":     true,
				"unit_id":       124,
			}))
		// when
		x1, err := s.GetOrCreateDogmaAttributeESI(ctx, 20)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(20), x1.ID)
			assert.Equal(t, float32(1), x1.DefaultValue)
			assert.Equal(t, "Factor by which top speed increases.", x1.Description)
			assert.Equal(t, "Maximum Velocity Bonus", x1.DisplayName)
			assert.Equal(t, int32(1389), x1.IconID)
			assert.Equal(t, "speedFactor", x1.Name)
			assert.True(t, x1.IsHighGood)
			assert.True(t, x1.IsPublished)
			assert.False(t, x1.IsStackable)
			assert.Equal(t, app.EveUnitID(124), x1.Unit)
			x2, err := st.GetEveDogmaAttribute(ctx, 20)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestAddMissingEveEntities(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	t.Run("do nothing when all entities already exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		e1 := factory.CreateEveEntityCharacter()
		// when
		ids, err := s.AddMissingEntities(ctx, []int32{e1.ID})
		// then
		assert.Equal(t, 0, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Len(t, ids, 0)
		}
	})
	t.Run("can resolve missing entities", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		data := []map[string]any{
			{"id": 47, "name": "Erik", "category": "character"},
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, data)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		// when
		ids, err := s.AddMissingEntities(ctx, []int32{47})
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Equal(t, int32(47), ids[0])
			e, err := st.GetEveEntity(ctx, 47)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, e.Name, "Erik")
			assert.Equal(t, e.Category, app.EveEntityCharacter)
		}
	})
	t.Run("can report normal error correctly", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		// when
		_, err := s.AddMissingEntities(ctx, []int32{47})
		// then
		assert.Error(t, err)
	})
	t.Run("can resolve mix of missing and non-missing entities", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		e1 := factory.CreateEveEntityAlliance()
		data := []map[string]any{
			{"id": 47, "name": "Erik", "category": "character"},
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, data)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		// when
		ids, err := s.AddMissingEntities(ctx, []int32{47, e1.ID})
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Equal(t, int32(47), ids[0])
		}
	})
	t.Run("can resolve more then 1000 IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		const count = 1001
		ids := make([]int32, count)
		data := make([]map[string]any, count)
		for i := range count {
			id := int32(i) + 1000
			ids[i] = id
			obj := map[string]any{
				"id":       id,
				"name":     fmt.Sprintf("Name #%d", id),
				"category": "character",
			}
			data[i] = obj
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, data)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		// when
		missing, err := s.AddMissingEntities(ctx, ids)
		// then
		assert.Equal(t, 2, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Len(t, missing, count)
			ids2, err := st.ListEveEntityIDs(ctx)
			if err != nil {
				t.Fatal(err)
			}
			assert.ElementsMatch(t, ids, ids2.ToSlice())
		}
	})
	t.Run("should store unresolvable IDs accordingly", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.NewStringResponder(404, ""),
		)
		// when
		ids, err := s.AddMissingEntities(ctx, []int32{666})
		// then
		assert.GreaterOrEqual(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Equal(t, int32(666), ids[0])
			e, err := st.GetEveEntity(ctx, 666)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, e.Name, "?")
			assert.Equal(t, e.Category, app.EveEntityUnknown)
		}
	})
	t.Run("should not call API with known invalid IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.NewStringResponder(404, ""),
		)
		// when
		ids, err := s.AddMissingEntities(ctx, []int32{1})
		// then
		assert.GreaterOrEqual(t, 0, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Len(t, ids, 0)
			e, err := st.GetEveEntity(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, e.Name, "?")
			assert.Equal(t, e.Category, app.EveEntityUnknown)
		}
	})
	t.Run("should do nothing with ID 0", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.NewStringResponder(404, ""),
		)
		// when
		ids, err := s.AddMissingEntities(ctx, []int32{0})
		// then
		assert.GreaterOrEqual(t, 0, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.Len(t, ids, 0)
			r := db.QueryRow("SELECT count(*) FROM eve_entities;")
			var c int
			if err := r.Scan(&c); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 0, c)
		}
	})
	t.Run("can deal with a mix of valid and invalid IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		data := []map[string]any{
			{"id": 47, "name": "Erik", "category": "character"},
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, data)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		httpmock.RegisterMatcherResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/universe/names/`,
			httpmock.BodyContainsString("666"),
			httpmock.NewStringResponder(404, `{"error":"Invalid ID"}`),
		)
		// when
		_, err := s.AddMissingEntities(ctx, []int32{47, 666})
		// then
		assert.LessOrEqual(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			e1, err := st.GetEveEntity(ctx, 47)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, e1.Name, "Erik")
			assert.Equal(t, e1.Category, app.EveEntityCharacter)
			e2, err := st.GetEveEntity(ctx, 666)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, e2.Category, app.EveEntityUnknown)
		}
	})
}

func TestToEveEntities(t *testing.T) {
	ctx := context.Background()
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	t.Run("should resolve normal IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		e1 := factory.CreateEveEntity()
		e2 := factory.CreateEveEntity()
		// when
		oo, err := s.ToEntities(ctx, []int32{e1.ID, e2.ID})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, map[int32]*app.EveEntity{e1.ID: e1, e2.ID: e2}, oo)
		}
	})
	t.Run("should map unknown IDs to empty objects", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		// when
		oo, err := s.ToEntities(ctx, []int32{0, 1})
		// then
		if assert.NoError(t, err) {
			assert.EqualValues(t, &app.EveEntity{ID: 0}, oo[0])
			assert.EqualValues(t, &app.EveEntity{ID: 1, Name: "?", Category: app.EveEntityUnknown}, oo[1])
		}
	})
}

func TestGetOrCreateEveCategoryESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing category", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: 6})
		// when
		x1, err := s.GetOrCreateCategoryESI(ctx, 6)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(6), x1.ID)
		}
	})
	t.Run("should fetch category from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		data := `{
			"category_id": 6,
			"groups": [
			  25,
			  26,
			  27
			],
			"name": "Ship",
			"published": true
		  }`
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/categories/\d+/`,
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateCategoryESI(ctx, 6)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(6), x1.ID)
			assert.Equal(t, "Ship", x1.Name)
			assert.Equal(t, true, x1.IsPublished)
			x2, err := st.GetEveCategory(ctx, 6)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEveGroupESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing group", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: 25})
		// when
		x1, err := s.GetOrCreateGroupESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(25), x1.ID)
		}
	})
	t.Run("should fetch group from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: 6})
		data := `{
			"category_id": 6,
			"group_id": 25,
			"name": "Frigate",
			"published": true,
			"types": [
			  587,
			  586,
			  585
			]
		  }`
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/groups/\d+/`,
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateGroupESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(25), x1.ID)
			assert.Equal(t, "Frigate", x1.Name)
			assert.Equal(t, int32(6), x1.Category.ID)
			assert.Equal(t, true, x1.IsPublished)
			x2, err := st.GetEveGroup(ctx, 25)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEveTypeESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing type", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})
		// when
		x1, err := s.GetOrCreateTypeESI(ctx, 587)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(587), x1.ID)
		}
	})
	t.Run("should fetch type from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: 25})
		factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{ID: 161})
		factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{ID: 162})
		data := `{
			"description": "The Rifter is a...",
			"dogma_attributes": [
				{
					"attribute_id": 161,
					"value": 11
					},
				{
					"attribute_id": 162,
					"value": 12
				}
			],
			"dogma_effects": [
				{
					"effect_id": 111,
					"is_default": true
					},
				{
					"effect_id": 112,
					"is_default": false
				}
			],
			"group_id": 25,
			"name": "Rifter",
			"published": true,
			"type_id": 587
		  }`
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/types/\d+/`,
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateTypeESI(ctx, 587)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(587), x1.ID)
			assert.Equal(t, "Rifter", x1.Name)
			assert.Equal(t, int32(25), x1.Group.ID)
			assert.Equal(t, true, x1.IsPublished)
			x2, err := st.GetEveType(ctx, 587)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
			y, err := st.GetEveTypeDogmaAttribute(ctx, 587, 161)
			if assert.NoError(t, err) {
				assert.Equal(t, float32(11), y)
			}
			z, err := st.GetEveTypeDogmaEffect(ctx, 587, 111)
			if assert.NoError(t, err) {
				assert.True(t, z)
			}

		}
	})
	t.Run("should fetch group from ESI and create it (integration)", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()

		data1 := `{
			"category_id": 6,
			"groups": [
			  25,
			  26,
			  27
			],
			"name": "Ship",
			"published": true
		  }`
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/categories/\d+/`,
			httpmock.NewStringResponder(200, data1).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		data2 := `{
			"category_id": 6,
			"group_id": 25,
			"name": "Frigate",
			"published": true,
			"types": [
			  587,
			  586,
			  585
			]
		  }`
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/groups/\d+/`,
			httpmock.NewStringResponder(200, data2).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		data3 := `{
			"description": "The Rifter is a...",
			"group_id": 25,
			"name": "Rifter",
			"published": true,
			"type_id": 587
		  }`
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/types/\d+/`,
			httpmock.NewStringResponder(200, data3).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateTypeESI(ctx, 587)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(587), x1.ID)
			assert.Equal(t, "Rifter", x1.Name)
			assert.Equal(t, int32(25), x1.Group.ID)
			assert.Equal(t, true, x1.IsPublished)
			x2, err := st.GetEveType(ctx, 587)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

const (
	stationID   = 60000277
	structureID = 1_000_000_000_009
)

func TestEveLocationOther(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(r)
	ctx := context.Background()
	t.Run("should create location for a station", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		owner := factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000003})
		system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000148})
		myType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 1531})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/universe/stations/%d/", stationID),
			httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
				"max_dockable_ship_volume": 50000000,
				"name":                     "Jakanerva III - Moon 15 - Prompt Delivery Storage",
				"office_rental_cost":       10000,
				"owner":                    1000003,
				"position": map[string]any{
					"x": 165632286720,
					"y": 2771804160,
					"z": -2455331266560,
				},
				"race_id":                    1,
				"reprocessing_efficiency":    0.5,
				"reprocessing_stations_take": 0.05,
				"services": []string{
					"courier-missions",
					"reprocessing-plant",
					"market",
					"repair-facilities",
					"fitting",
					"news",
					"storage",
					"insurance",
					"docking",
					"office-rental",
					"loyalty-point-store",
					"navy-offices",
				},
				"station_id": 60000277,
				"system_id":  30000148,
				"type_id":    1531,
			}),
		)
		// when
		x1, err := s.GetOrCreateLocationESI(ctx, stationID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int64(stationID), x1.ID)
			assert.Equal(t, "Jakanerva III - Moon 15 - Prompt Delivery Storage", x1.Name)
			assert.Equal(t, owner, x1.Owner)
			assert.Equal(t, system, x1.SolarSystem)
			assert.Equal(t, myType, x1.Type)
			x2, err := r.GetLocation(ctx, stationID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should create location for a solar system", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		myType := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeSolarSystem})
		system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000148})
		// when
		x1, err := s.GetOrCreateLocationESI(ctx, int64(system.ID))
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int64(system.ID), x1.ID)
			assert.Equal(t, system.Name, x1.DisplayName())
			assert.Nil(t, x1.Owner)
			assert.Equal(t, system, x1.SolarSystem)
			assert.Equal(t, myType, x1.Type)
			x2, err := r.GetLocation(ctx, int64(system.ID))
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestLocationStructures(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(r)
	t.Run("should return existing structure", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: structureID, Name: "Alpha"})
		// when
		x, err := s.GetOrCreateLocationESI(context.Background(), structureID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "Alpha", x.Name)
		}
	})
	t.Run("should fetch structure from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		owner := factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000142})
		myType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 99})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/structures/\d+/`,
			httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
				"name":            "V-3YG7 VI - The Capital",
				"owner_id":        109299958,
				"solar_system_id": 30000142,
				"type_id":         99,
				"position": map[string]any{
					"x": 1.1,
					"y": 2.2,
					"z": 3.3,
				}}),
		)
		ctx := context.WithValue(context.Background(), goesi.ContextAccessToken, "DUMMY")
		// when
		x1, err := s.GetOrCreateLocationESI(ctx, structureID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int64(structureID), x1.ID)
			assert.Equal(t, "V-3YG7 VI - The Capital", x1.Name)
			assert.Equal(t, owner, x1.Owner)
			assert.Equal(t, system, x1.SolarSystem)
			assert.Equal(t, myType, x1.Type)
			x2, err := r.GetLocation(ctx, structureID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should return error when trying to fetch structure without token", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		// when
		_, err := s.GetOrCreateLocationESI(context.Background(), structureID)
		// then
		assert.Error(t, err)
	})
	t.Run("should create empty structure from ESI when no access", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/structures/\d+/`,
			httpmock.NewJsonResponderOrPanic(http.StatusForbidden, map[string]any{
				"error": "forbidden",
			}),
		)
		ctx := context.WithValue(context.Background(), goesi.ContextAccessToken, "DUMMY")
		// when
		x1, err := s.GetOrCreateLocationESI(ctx, structureID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int64(structureID), x1.ID)
			assert.Equal(t, "", x1.Name)
			assert.Nil(t, x1.Owner)
			assert.Nil(t, x1.SolarSystem)
			assert.Nil(t, x1.Type)
			x2, err := r.GetLocation(ctx, structureID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should return error when other http error occurs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/structures/\d+/`,
			httpmock.NewJsonResponderOrPanic(http.StatusNotFound, map[string]any{
				"error": "xxx",
			}),
		)
		ctx := context.WithValue(context.Background(), goesi.ContextAccessToken, "DUMMY")
		// when
		_, err := s.GetOrCreateLocationESI(ctx, structureID)
		// then
		assert.Error(t, err)
	})
}

func TestGetOrCreateEveRegionESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing region", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveRegion(storage.CreateEveRegionParams{ID: 6})
		// when
		x1, err := s.GetOrCreateRegionESI(ctx, 6)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(6), x1.ID)
		}
	})
	t.Run("should fetch region from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		data := `{
			"constellations": [
			  20000302,
			  20000303
			],
			"description": "It has long been an established fact of civilization...",
			"name": "Metropolis",
			"region_id": 10000042
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/regions/10000042/",
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateRegionESI(ctx, 10000042)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(10000042), x1.ID)
			assert.Equal(t, "Metropolis", x1.Name)
			x2, err := st.GetEveRegion(ctx, 10000042)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEveConstellationESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing constellation", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveConstellation(storage.CreateEveConstellationParams{ID: 25})
		// when
		x1, err := s.GetOrCreateConstellationESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(25), x1.ID)
		}
	})
	t.Run("should fetch constellation from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveRegion(storage.CreateEveRegionParams{ID: 10000001})
		data := `{
			"constellation_id": 20000009,
			"name": "Mekashtad",
			"position": {
			  "x": 67796138757472320,
			  "y": -70591121348560960,
			  "z": -59587016159270070
			},
			"region_id": 10000001,
			"systems": [
			  20000302,
			  20000303
			]
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/constellations/20000009/",
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateConstellationESI(ctx, 20000009)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(20000009), x1.ID)
			assert.Equal(t, "Mekashtad", x1.Name)
			assert.Equal(t, int32(10000001), x1.Region.ID)
			x2, err := st.GetEveConstellation(ctx, 20000009)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEveSolarSystemESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing solar system", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 587})
		// when
		x1, err := s.GetOrCreateSolarSystemESI(ctx, 587)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(587), x1.ID)
		}
	})
	t.Run("should fetch solar system from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveConstellation(storage.CreateEveConstellationParams{ID: 20000001})
		data := `{
			"constellation_id": 20000001,
			"name": "Akpivem",
			"planets": [
			  {
				"moons": [
				  40000042
				],
				"planet_id": 40000041
			  },
			  {
				"planet_id": 40000043
			  }
			],
			"position": {
			  "x": -91174141133075340,
			  "y": 43938227486247170,
			  "z": -56482824383339900
			},
			"security_class": "B",
			"security_status": 0.8462923765182495,
			"star_id": 40000040,
			"stargates": [
			  50000342
			],
			"system_id": 30000003
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v4/universe/systems/30000003/",
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateSolarSystemESI(ctx, 30000003)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(30000003), x1.ID)
			assert.Equal(t, "Akpivem", x1.Name)
			assert.Equal(t, int32(20000001), x1.Constellation.ID)
			x2, err := st.GetEveSolarSystem(ctx, 30000003)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should fetch solar system from ESI and create it (integration)", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()

		data1 := `{
			"constellations": [
			  20000001,
			  20000002,
			  20000003,
			  20000004,
			  20000005,
			  20000006,
			  20000007,
			  20000008,
			  20000009,
			  20000010,
			  20000011,
			  20000012,
			  20000013,
			  20000014,
			  20000015,
			  20000016
			],
			"description": "The Derelik region, sovereign seat of the Ammatar Mandate, became the shield to the Amarrian flank in the wake of the Minmatar Rebellion. Derelik witnessed many hostile exchanges between the Amarr and rebel forces as the latter tried to push deeper into the territory of their former masters. Having held their ground, thanks in no small part to the Ammatars' military efforts, the Amarr awarded the Ammatar with their own province. However, this portion of space shared borders with the newly forming Minmatar Republic as well as the Empire, and thus came to be situated in a dark recess surrounded by hostiles. \n\nGiven the lack of safe routes elsewhere, the local economies of this region were dependent on trade with the Amarr as their primary means of survival. The Ammatar persevered over many decades of economic stagnation and limited trade partners, and their determination has in recent decades been rewarded with an increase in economic prosperity. This harsh trail is a point of pride for all who call themselves Ammatar, and it has bolstered their faith in the Amarrian way to no end.",
			"name": "Derelik",
			"region_id": 10000001
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/regions/10000001/",
			httpmock.NewStringResponder(200, data1).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		data2 := `{
			"constellation_id": 20000001,
			"name": "San Matar",
			"position": {
			  "x": -94046559700991340,
			  "y": 49520153153798850,
			  "z": -42738731818401970
			},
			"region_id": 10000001,
			"systems": [
			  30000001,
			  30000002,
			  30000003,
			  30000004,
			  30000005,
			  30000006,
			  30000007,
			  30000008
			]
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/constellations/20000001/",
			httpmock.NewStringResponder(200, data2).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		data3 := `{
			"constellation_id": 20000001,
			"name": "Akpivem",
			"planets": [
			  {
				"moons": [
				  40000042
				],
				"planet_id": 40000041
			  },
			  {
				"planet_id": 40000043
			  }
			],
			"position": {
			  "x": -91174141133075340,
			  "y": 43938227486247170,
			  "z": -56482824383339900
			},
			"security_class": "B",
			"security_status": 0.8462923765182495,
			"star_id": 40000040,
			"stargates": [
			  50000342
			],
			"system_id": 30000003
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v4/universe/systems/30000003/",
			httpmock.NewStringResponder(200, data3).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateSolarSystemESI(ctx, 30000003)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(30000003), x1.ID)
			assert.Equal(t, "Akpivem", x1.Name)
			assert.Equal(t, int32(20000001), x1.Constellation.ID)
			x2, err := st.GetEveSolarSystem(ctx, 30000003)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEvePlanetESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing planet", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 25})
		// when
		x1, err := s.GetOrCreatePlanetESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(25), x1.ID)
		}
	})
	t.Run("should fetch planet from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		solarSystem := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000003})
		type_ := factory.CreateEveType(storage.CreateEveTypeParams{ID: 13})
		data := map[string]any{
			"name":      "Akpivem III",
			"planet_id": 40000046,
			"position": map[string]any{
				"x": -189226344497,
				"y": 9901605317,
				"z": -254852632979,
			},
			"system_id": 30000003,
			"type_id":   13,
		}
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/planets/40000046/",
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		x1, err := s.GetOrCreatePlanetESI(ctx, 40000046)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(40000046), x1.ID)
			assert.Equal(t, "Akpivem III", x1.Name)
			assert.Equal(t, solarSystem, x1.SolarSystem)
			assert.Equal(t, type_, x1.Type)
		}
	})
}

func TestGetOrCreateEveMoonESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing moon", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveMoon(storage.CreateEveMoonParams{ID: 25})
		// when
		x1, err := s.GetOrCreateMoonESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(25), x1.ID)
		}
	})
	t.Run("should fetch moon from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		solarSystem := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000003})
		data := map[string]any{
			"moon_id": 40000042,
			"name":    "Akpivem I - Moon 1",
			"position": map[string]any{
				"x": 58605102008,
				"y": -3066616285,
				"z": -55193617920,
			},
			"system_id": 30000003,
		}
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/moons/40000042/",
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		x1, err := s.GetOrCreateMoonESI(ctx, 40000042)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(40000042), x1.ID)
			assert.Equal(t, "Akpivem I - Moon 1", x1.Name)
			assert.Equal(t, solarSystem, x1.SolarSystem)
		}
	})
}

func TestGetRouteESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return short route when origin and dest the same", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		o := factory.CreateEveSolarSystem()
		// when
		x, err := s.GetRouteESI(ctx, o, o, app.RouteShortest)
		// then
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, []*app.EveSolarSystem{o}, x)
			assert.Equal(t, 0, httpmock.GetTotalCallCount())
		}
	})
	t.Run("should return invalid route when origin in WH space", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		orig := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 31000001})
		dest := factory.CreateEveSolarSystem()
		// when
		x, err := s.GetRouteESI(ctx, dest, orig, app.RouteShortest)
		// then
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, []*app.EveSolarSystem{}, x)
		}
	})
	t.Run("should return error when caled with invalid preference", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		orig := factory.CreateEveSolarSystem()
		dest := factory.CreateEveSolarSystem()
		// when
		_, err := s.GetRouteESI(ctx, dest, orig, app.RoutePreference("invalid"))
		// then
		if assert.Error(t, err) {
			assert.Equal(t, 0, httpmock.GetTotalCallCount())
		}
	})
	t.Run("should return invalid route when dest in WH space", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		orig := factory.CreateEveSolarSystem()
		dest := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 31000001})
		// when
		x, err := s.GetRouteESI(ctx, dest, orig, app.RouteShortest)
		// then
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, []*app.EveSolarSystem{}, x)
		}
	})
	// FIXME
	// t.Run("should return route when valid", func(t *testing.T) {
	// 	// given
	// 	testutil.TruncateTables(db)
	// 	httpmock.Reset()
	// 	s1 := factory.CreateEveSolarSystem()
	// 	s2 := factory.CreateEveSolarSystem()
	// 	s3 := factory.CreateEveSolarSystem()
	// 	httpmock.RegisterResponder(
	// 		"GET",
	// 		fmt.Sprintf("https://esi.evetech.net/v1/route/%d/%d/", s1.ID, s3.ID),
	// 		httpmock.NewJsonResponderOrPanic(200, []int32{s1.ID, s2.ID, s3.ID}))
	// 	// when
	// 	x, err := s.GetRouteESI(ctx, s3, s1, app.RouteShortest)
	// 	// then
	// 	if assert.NoError(t, err) {
	// 		assert.Equal(t, []*app.EveSolarSystem{s1, s2, s3}, x)
	// 	}
	// })
}

func TestMembershipHistory(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return corporation membership history", func(t *testing.T) {
		// given
		s.Now = func() time.Time { return time.Date(2016, 7, 30, 20, 0, 0, 0, time.UTC) }
		testutil.TruncateTables(db)
		httpmock.Reset()
		c1 := factory.CreateEveEntityCorporation(app.EveEntity{ID: 90000001})
		c2 := factory.CreateEveEntityCorporation(app.EveEntity{ID: 90000002})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/characters/%d/corporationhistory/", 42),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"corporation_id": 90000001,
					"is_deleted":     true,
					"record_id":      500,
					"start_date":     "2016-06-26T20:00:00Z",
				},
				{
					"corporation_id": 90000002,
					"record_id":      501,
					"start_date":     "2016-07-26T20:00:00Z",
				},
			}))
		// when
		x, err := s.GetCharacterCorporationHistory(ctx, 42)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, x, 2)
			assert.EqualValues(t, app.MembershipHistoryItem{
				Days:         4,
				IsDeleted:    false,
				Organization: c2,
				RecordID:     501,
				StartDate:    time.Date(2016, 7, 26, 20, 0, 0, 0, time.UTC),
			}, x[0])
			assert.EqualValues(t, app.MembershipHistoryItem{
				EndDate:      time.Date(2016, 7, 26, 20, 0, 0, 0, time.UTC),
				Days:         30,
				IsDeleted:    true,
				IsOldest:     true,
				Organization: c1,
				RecordID:     500,
				StartDate:    time.Date(2016, 6, 26, 20, 0, 0, 0, time.UTC),
			}, x[1])
		}
	})
	t.Run("should return alliance membership history", func(t *testing.T) {
		// given
		s.Now = func() time.Time { return time.Date(2016, 10, 30, 20, 0, 0, 0, time.UTC) }
		testutil.TruncateTables(db)
		httpmock.Reset()
		c1 := factory.CreateEveEntityAlliance(app.EveEntity{ID: 99000006})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/corporations/%d/alliancehistory/", 42),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id": 99000006,
					"is_deleted":  true,
					"record_id":   23,
					"start_date":  "2016-10-25T14:46:00Z",
				},
				{
					"record_id":  1,
					"start_date": "2015-07-06T20:56:00Z",
				},
			}))
		// when
		x, err := s.GetCorporationAllianceHistory(ctx, 42)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, x, 2)
			assert.EqualValues(t, app.MembershipHistoryItem{
				Days:         5,
				IsDeleted:    true,
				Organization: c1,
				RecordID:     23,
				StartDate:    time.Date(2016, 10, 25, 14, 46, 0, 0, time.UTC),
			}, x[0])
			assert.EqualValues(t, app.MembershipHistoryItem{
				EndDate:   time.Date(2016, 10, 25, 14, 46, 0, 0, time.UTC),
				Days:      476,
				IsOldest:  true,
				RecordID:  1,
				StartDate: time.Date(2015, 7, 6, 20, 56, 0, 0, time.UTC),
			}, x[1])
		}
	})
}

func TestGetOrCreateEveRaceESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing race", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		x1 := factory.CreateEveRace(app.EveRace{ID: 7})
		// when
		x2, err := s.GetOrCreateRaceESI(ctx, 7)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
	t.Run("should create race from ESI when it does not exit in DB", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/races/",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id": 500001,
					"description": "Founded on the tenets of patriotism and hard work...",
					"name":        "Caldari",
					"race_id":     7,
				},
			}))

		// when
		x1, err := s.GetOrCreateRaceESI(ctx, 7)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "Caldari", x1.Name)
			assert.Equal(t, "Founded on the tenets of patriotism and hard work...", x1.Description)
			x2, err := st.GetEveRace(ctx, 7)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should return specific error when race ID is invalid", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/races/",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id": 500001,
					"description": "Founded on the tenets of patriotism and hard work...",
					"name":        "Caldari",
					"race_id":     7,
				},
			}))

		// when
		_, err := s.GetOrCreateRaceESI(ctx, 42)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
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

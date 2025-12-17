package eveuniverseservice_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ErikKalkoken/kx/set"
	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestFetchAlliance(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
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
	db, st, factory := testutil.NewDBInMemory()
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
		testutil.MustTruncateTables(db)
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
		testutil.MustTruncateTables(db)
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
func TestGetOrCreateEveCorporationESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should create new corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		alliance := factory.CreateEveEntityAlliance(app.EveEntity{ID: 434243723})
		faction := factory.CreateEveEntity(app.EveEntity{ID: 123, Category: app.EveEntityFaction})
		station := factory.CreateEveEntity(app.EveEntity{ID: 456, Category: app.EveEntityStation})
		ceo := factory.CreateEveEntityCharacter(app.EveEntity{ID: 180548812})
		creator := factory.CreateEveEntityCharacter()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":     434243723,
				"ceo_id":          180548812,
				"creator_id":      creator.ID,
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
			assert.Equal(t, creator, o.Creator)
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
		testutil.MustTruncateTables(db)
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

func TestUpdateOrCreateEveCorporationESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should create new corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		alliance := factory.CreateEveEntityAlliance(app.EveEntity{ID: 434243723})
		faction := factory.CreateEveEntity(app.EveEntity{ID: 123, Category: app.EveEntityFaction})
		station := factory.CreateEveEntity(app.EveEntity{ID: 456, Category: app.EveEntityStation})
		ceo := factory.CreateEveEntityCharacter(app.EveEntity{ID: 180548812})
		creator := factory.CreateEveEntityCharacter()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":     434243723,
				"ceo_id":          180548812,
				"creator_id":      creator.ID,
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
		o, err := s.UpdateOrCreateCorporationFromESI(ctx, 109299958)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, alliance, o.Alliance)
			assert.Equal(t, creator, o.Creator)
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
	t.Run("should update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		orig := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{ID: 109299958})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		alliance := factory.CreateEveEntityAlliance(app.EveEntity{ID: 434243723})
		faction := factory.CreateEveEntity(app.EveEntity{ID: 123, Category: app.EveEntityFaction})
		station := factory.CreateEveEntity(app.EveEntity{ID: 456, Category: app.EveEntityStation})
		ceo := factory.CreateEveEntityCharacter(app.EveEntity{ID: 180548812})
		creator := factory.CreateEveEntityCharacter()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":     434243723,
				"ceo_id":          180548812,
				"creator_id":      creator.ID,
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
		o, err := s.UpdateOrCreateCorporationFromESI(ctx, 109299958)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, alliance, o.Alliance)
			assert.Equal(t, orig.Creator, o.Creator)
			assert.Equal(t, ceo, o.Ceo)
			assert.Equal(t, orig.DateFounded.MustValue(), o.DateFounded.MustValue())
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
}

func TestUpdateAllEveCorporationESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("can update from ESI and report changed IDs", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		orig := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{ID: 109299958})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		alliance := factory.CreateEveEntityAlliance(app.EveEntity{ID: 434243723})
		faction := factory.CreateEveEntity(app.EveEntity{ID: 123, Category: app.EveEntityFaction})
		station := factory.CreateEveEntity(app.EveEntity{ID: 456, Category: app.EveEntityStation})
		ceo := factory.CreateEveEntityCharacter(app.EveEntity{ID: 180548812})
		creator := factory.CreateEveEntityCharacter()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":     434243723,
				"ceo_id":          180548812,
				"creator_id":      creator.ID,
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
		got, err := s.UpdateAllCorporationsESI(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of[int32](109299958)
			xassert.EqualSet(t, want, got)
			ec, err := st.GetEveCorporation(ctx, 109299958)
			if assert.NoError(t, err) {
				assert.Equal(t, alliance, ec.Alliance)
				assert.Equal(t, orig.Creator, ec.Creator)
				assert.Equal(t, ceo, ec.Ceo)
				assert.Equal(t, orig.DateFounded.MustValue(), ec.DateFounded.MustValue())
				assert.Equal(t, "This is a corporation description, it's basically just a string", ec.Description)
				assert.Equal(t, faction, ec.Faction)
				assert.Equal(t, station, ec.HomeStation)
				assert.Equal(t, 656, ec.MemberCount)
				assert.Equal(t, "C C P", ec.Name)
				assert.Equal(t, float32(0.256), ec.TaxRate)
				assert.Equal(t, "-CCP-", ec.Ticker)
				assert.Equal(t, "http://www.eveonline.com", ec.URL)
			}
			ee, err := st.GetEveEntity(ctx, 109299958)
			if assert.NoError(t, err) {
				assert.Equal(t, ec.Name, ee.Name)
				assert.Equal(t, app.EveEntityCorporation, ee.Category)
			}
		}
	})
	t.Run("can report when not changed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 109299958})
		alliance := factory.CreateEveEntityAlliance(app.EveEntity{ID: 434243723})
		faction := factory.CreateEveEntity(app.EveEntity{ID: 123, Category: app.EveEntityFaction})
		station := factory.CreateEveEntity(app.EveEntity{ID: 456, Category: app.EveEntityStation})
		ceo := factory.CreateEveEntityCharacter(app.EveEntity{ID: 180548812})
		creator := factory.CreateEveEntityCharacter()
		factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			ID:            109299958,
			AllianceID:    optional.New(alliance.ID),
			CeoID:         optional.New(ceo.ID),
			CreatorID:     optional.New(creator.ID),
			DateFounded:   optional.New(time.Date(2004, 11, 28, 16, 42, 51, 0, time.UTC)),
			Description:   "This is a corporation description, it's basically just a string",
			FactionID:     optional.New(faction.ID),
			HomeStationID: optional.New(station.ID),
			MemberCount:   656,
			Name:          "C C P",
			TaxRate:       0.256,
			Ticker:        "-CCP-",
			URL:           "http://www.eveonline.com",
			WarEligible:   false,
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":     434243723,
				"ceo_id":          180548812,
				"creator_id":      creator.ID,
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
		got, err := s.UpdateAllCorporationsESI(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of[int32]()
			xassert.EqualSet(t, want, got)
		}
	})
}

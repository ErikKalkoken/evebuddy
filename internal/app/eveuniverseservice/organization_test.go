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
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestFetchAlliance(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
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
			fmt.Sprintf("https://esi.evetech.net/alliances/%d", allianceID),
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
		require.NoError(t, err)
		xassert.Equal(t, "C C P Alliance", x.Name)
		xassert.Equal(t, "<C C P>", x.Ticker)
		xassert.Equal(t, creator, x.Creator)
		xassert.Equal(t, creatorCorp, x.CreatorCorporation)
		xassert.Equal(t, executor, x.ExecutorCorporation.MustValue())
		xassert.Equal(t, faction, x.Faction.MustValue())
		xassert.Equal(t, time.Date(2016, 6, 26, 21, 0, 0, 0, time.UTC), x.DateFounded)
	})
	t.Run("should return nil for undefined entities", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/alliances/%d", allianceID),
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
		require.NoError(t, err)
		xassert.Equal(t, "C C P Alliance", x.Name)
		xassert.Empty(t, x.Faction)
	})
}

func TestFetchAllianceCorporations(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
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
			fmt.Sprintf("https://esi.evetech.net/alliances/%d/corporations", allianceID),
			httpmock.NewJsonResponderOrPanic(200, []int64{102, 103}),
		)
		// when
		oo, err := s.FetchAllianceCorporations(ctx, allianceID)
		// then
		require.NoError(t, err)
		got := xslices.Map(oo, func(a *app.EveEntity) int64 {
			return a.ID
		})
		want := []int64{103, 102}
		xassert.Equal(t, want, got)
	})
	t.Run("should return empty list when there are no corporations", func(t *testing.T) {
		// given
		const allianceID = 42
		testutil.MustTruncateTables(db)
		factory.CreateEveEntityAlliance(app.EveEntity{ID: allianceID})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/alliances/%d/corporations", allianceID),
			httpmock.NewJsonResponderOrPanic(200, []int64{}),
		)
		// when
		oo, err := s.FetchAllianceCorporations(ctx, allianceID)
		// then
		require.NoError(t, err)
		assert.Len(t, oo, 0)
	})
}
func TestGetOrCreateEveCorporationESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("should create new corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			description = "This is a corporation description, it's basically just a string"
			memberCount = 656
			name        = "C C P"
			taxRate     = 0.256
			ticker      = "-CCP-"
			url         = "http://www.eveonline.com"
		)
		corporation := factory.CreateEveEntityCorporation()
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityFaction})
		station := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityStation})
		ceo := factory.CreateEveEntityCharacter()
		creator := factory.CreateEveEntityCharacter()
		dateFounded := time.Now().Add(-time.Hour * 1000).Round(time.Second)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d", corporation.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":         alliance.ID,
				"ceo_id":              ceo.ID,
				"creator_id":          creator.ID,
				"date_founded":        dateFounded.Format(time.RFC3339),
				"description":         description,
				"enlisted_faction_id": faction.ID,
				"home_station_id":     station.ID,
				"member_count":        memberCount,
				"name":                name,
				"ticker":              ticker,
				"url":                 url,
				"war_eligible":        false,
				"shares":              100000000,
				"state":               "active",
				"friendly_fire":       "legal",
				"tax_rates": map[string]float64{
					"isk":           taxRate,
					"loyalty_point": 0,
				},
				"type": "player_owned",
			}),
		)
		// when
		o, err := s.GetOrCreateCorporationESI(t.Context(), corporation.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, alliance, o.Alliance.MustValue())
		xassert.Equal(t, creator, o.Creator.MustValue())
		xassert.Equal(t, ceo, o.Ceo.MustValue())
		xassert.Equal(t, dateFounded, o.DateFounded.MustValue())
		xassert.Equal(t, description, o.Description)
		xassert.Equal(t, faction, o.Faction.MustValue())
		xassert.Equal(t, station, o.HomeStation.MustValue())
		xassert.Equal(t, memberCount, o.MemberCount)
		xassert.Equal(t, name, o.Name)
		xassert.Equal(t, taxRate, o.TaxRate)
		xassert.Equal(t, ticker, o.Ticker)
		xassert.Equal(t, url, o.URL.ValueOrZero())
	})
	t.Run("can handle no CEO and no creator", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateEveEntityCorporation()
		station := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityStation})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d", corporation.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"date_founded":    "2004-11-28T16:42:51Z",
				"description":     "This is a corporation description, it's basically just a string",
				"member_count":    656,
				"name":            "C C P",
				"ticker":          "-CCP-",
				"home_station_id": station.ID,
				"war_eligible":    false,
				"shares":          100000000,
				"state":           "active",
				"friendly_fire":   "legal",
				"tax_rates": map[string]float64{
					"isk":           0.256,
					"loyalty_point": 0,
				},
				"type": "player_owned"}),
		)
		// when
		o, err := s.GetOrCreateCorporationESI(t.Context(), corporation.ID)
		// then
		require.NoError(t, err)
		xassert.Empty(t, o.Ceo)
		xassert.Empty(t, o.Creator)
	})
}

func TestUpdateOrCreateEveCorporationESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("should create new minimal corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			description = "This is a corporation description, it's basically just a string"
			memberCount = 656
			name        = "C C P"
			taxRate     = 0.256
			ticker      = "-CCP-"
			url         = "http://www.eveonline.com"
		)
		corporation := factory.CreateEveEntityCorporation()
		station := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityStation})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d", corporation.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"member_count":    memberCount,
				"name":            name,
				"ticker":          ticker,
				"description":     description,
				"home_station_id": station.ID,
				"war_eligible":    false,
				"shares":          100000000,
				"state":           "active",
				"friendly_fire":   "legal",
				"tax_rates": map[string]float64{
					"isk":           taxRate,
					"loyalty_point": 0,
				},
				"type": "player_owned",
			}),
		)
		// when
		o, err := s.UpdateOrCreateCorporationFromESI(t.Context(), corporation.ID)
		// then
		require.NoError(t, err)
		xassert.Empty(t, o.Alliance)
		xassert.Empty(t, o.Ceo)
		xassert.Empty(t, o.Creator)
		xassert.Empty(t, o.DateFounded)
		xassert.Equal(t, description, o.Description)
		xassert.Empty(t, o.Faction)
		xassert.Equal(t, station, o.HomeStation.MustValue())
		xassert.Equal(t, memberCount, o.MemberCount)
		xassert.Equal(t, name, o.Name)
		xassert.Equal(t, taxRate, o.TaxRate)
		xassert.Equal(t, ticker, o.Ticker)
		xassert.Empty(t, o.URL)
	})
	t.Run("should create new full corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			description = "This is a corporation description, it's basically just a string"
			memberCount = 656
			name        = "C C P"
			taxRate     = 0.256
			ticker      = "-CCP-"
			url         = "http://www.eveonline.com"
		)
		corporation := factory.CreateEveEntityCorporation()
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityFaction})
		station := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityStation})
		ceo := factory.CreateEveEntityCharacter()
		creator := factory.CreateEveEntityCharacter()
		dateFounded := time.Now().Add(-time.Hour * 1000).Round(time.Second)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d", corporation.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":         alliance.ID,
				"ceo_id":              ceo.ID,
				"creator_id":          creator.ID,
				"date_founded":        dateFounded.Format(time.RFC3339),
				"description":         description,
				"enlisted_faction_id": faction.ID,
				"home_station_id":     station.ID,
				"member_count":        memberCount,
				"name":                name,
				"ticker":              ticker,
				"url":                 url,
				"war_eligible":        false,
				"shares":              100000000,
				"state":               "active",
				"friendly_fire":       "legal",
				"tax_rates": map[string]float64{
					"isk":           taxRate,
					"loyalty_point": 0,
				},
				"type": "player_owned",
			}),
		)
		// when
		o, err := s.UpdateOrCreateCorporationFromESI(t.Context(), corporation.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, alliance, o.Alliance.MustValue())
		xassert.Equal(t, creator, o.Creator.MustValue())
		xassert.Equal(t, ceo, o.Ceo.MustValue())
		xassert.Equal(t, dateFounded, o.DateFounded.MustValue())
		xassert.Equal(t, description, o.Description)
		xassert.Equal(t, faction, o.Faction.MustValue())
		xassert.Equal(t, station, o.HomeStation.MustValue())
		xassert.Equal(t, memberCount, o.MemberCount)
		xassert.Equal(t, name, o.Name)
		xassert.Equal(t, taxRate, o.TaxRate)
		xassert.Equal(t, ticker, o.Ticker)
		xassert.Equal(t, url, o.URL.ValueOrZero())
	})
	t.Run("should update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateEveCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c1.ID})

		const (
			description = "This is a corporation description, it's basically just a string"
			memberCount = 656
			name        = "C C P"
			taxRate     = 0.256
			ticker      = "-CCP-"
			url         = "http://www.eveonline.com"
		)
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityFaction})
		station := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityStation})
		ceo := factory.CreateEveEntityCharacter()
		creator := factory.CreateEveEntityCharacter()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":         alliance.ID,
				"ceo_id":              ceo.ID,
				"creator_id":          creator.ID,
				"date_founded":        c1.DateFounded.MustValue().Format(time.RFC3339),
				"description":         description,
				"enlisted_faction_id": faction.ID,
				"home_station_id":     station.ID,
				"member_count":        memberCount,
				"name":                name,
				"ticker":              ticker,
				"url":                 url,
				"war_eligible":        false,
				"shares":              100000000,
				"state":               "active",
				"friendly_fire":       "legal",
				"tax_rates": map[string]float64{
					"isk":           taxRate,
					"loyalty_point": 0,
				},
				"type": "player_owned",
			}),
		)
		// when
		c2, err := s.UpdateOrCreateCorporationFromESI(t.Context(), c1.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, alliance, c2.Alliance.MustValue())
		xassert.Equal(t, c1.Creator, c2.Creator)
		xassert.Equal(t, ceo, c2.Ceo.MustValue())
		xassert.Equal(t, c1.DateFounded.MustValue(), c2.DateFounded.MustValue())
		xassert.Equal(t, description, c2.Description)
		xassert.Equal(t, faction, c2.Faction.MustValue())
		xassert.Equal(t, station, c2.HomeStation.MustValue())
		xassert.Equal(t, memberCount, c2.MemberCount)
		xassert.Equal(t, name, c2.Name)
		xassert.Equal(t, taxRate, c2.TaxRate)
		xassert.Equal(t, ticker, c2.Ticker)
		xassert.Equal(t, url, c2.URL.ValueOrZero())
	})
}

func TestUpdateAllEveCorporationESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("can update from ESI and report changed IDs", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c1 := factory.CreateEveCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c1.ID})

		const (
			description = "This is a corporation description, it's basically just a string"
			memberCount = 656
			name        = "C C P"
			taxRate     = 0.256
			ticker      = "-CCP-"
			url         = "http://www.eveonline.com"
		)
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityFaction})
		station := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityStation})
		ceo := factory.CreateEveEntityCharacter()
		creator := factory.CreateEveEntityCharacter()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":         alliance.ID,
				"ceo_id":              ceo.ID,
				"creator_id":          creator.ID,
				"date_founded":        c1.DateFounded.MustValue().Format(time.RFC3339),
				"description":         description,
				"enlisted_faction_id": faction.ID,
				"home_station_id":     station.ID,
				"member_count":        memberCount,
				"name":                name,
				"ticker":              ticker,
				"url":                 url,
				"war_eligible":        false,
				"shares":              100000000,
				"state":               "active",
				"friendly_fire":       "legal",
				"tax_rates": map[string]float64{
					"isk":           0.256,
					"loyalty_point": 0,
				},
				"type": "player_owned",
			}),
		)
		// when
		got, err := s.UpdateAllCorporationsESI(t.Context())
		// then
		require.NoError(t, err)
		want := set.Of(c1.ID)
		xassert.Equal(t, want, got)
		c2, err := st.GetEveCorporation(t.Context(), c1.ID)
		require.NoError(t, err)
		xassert.Equal(t, alliance, c2.Alliance.MustValue())
		xassert.Equal(t, c1.Creator, c2.Creator)
		xassert.Equal(t, ceo, c2.Ceo.MustValue())
		xassert.Equal(t, c1.DateFounded.MustValue(), c2.DateFounded.MustValue())
		xassert.Equal(t, description, c2.Description)
		xassert.Equal(t, faction, c2.Faction.MustValue())
		xassert.Equal(t, station, c2.HomeStation.MustValue())
		xassert.Equal(t, memberCount, c2.MemberCount)
		xassert.Equal(t, name, c2.Name)
		xassert.Equal(t, taxRate, c2.TaxRate)
		xassert.Equal(t, ticker, c2.Ticker)
		xassert.Equal(t, url, c2.URL.ValueOrZero())
		ee, err := st.GetEveEntity(t.Context(), c1.ID)
		require.NoError(t, err)
		xassert.Equal(t, c2.Name, ee.Name)
		xassert.Equal(t, app.EveEntityCorporation, ee.Category)
	})
	t.Run("can report when not changed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		corporation := factory.CreateEveEntityCorporation()
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityFaction})
		station := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityStation})
		ceo := factory.CreateEveEntityCharacter()
		creator := factory.CreateEveEntityCharacter()
		c1 := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			ID:            corporation.ID,
			AllianceID:    optional.New(alliance.ID),
			CeoID:         optional.New(ceo.ID),
			CreatorID:     optional.New(creator.ID),
			DateFounded:   optional.New(time.Date(2004, 11, 28, 16, 42, 51, 0, time.UTC)),
			Description:   optional.New("This is a corporation description, it's basically just a string"),
			FactionID:     optional.New(faction.ID),
			HomeStationID: optional.New(station.ID),
			MemberCount:   656,
			Name:          "C C P",
			Shares:        optional.New[int64](1000),
			TaxRate:       0.256,
			Ticker:        "-CCP-",
			URL:           optional.New("http://www.eveonline.com"),
			WarEligible:   optional.New(false),
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"alliance_id":         c1.Alliance.MustValue().ID,
				"ceo_id":              c1.Ceo.MustValue().ID,
				"creator_id":          c1.Creator.MustValue().ID,
				"date_founded":        c1.DateFounded.MustValue().Format(time.RFC3339),
				"description":         c1.Description,
				"enlisted_faction_id": c1.Faction.MustValue().ID,
				"home_station_id":     c1.HomeStation.MustValue().ID,
				"member_count":        c1.MemberCount,
				"name":                c1.Name,
				"ticker":              c1.Ticker,
				"url":                 c1.URL.MustValue(),
				"war_eligible":        false,
				"shares":              c1.Shares.MustValue(),
				"state":               "active",
				"friendly_fire":       "legal",
				"tax_rates": map[string]float64{
					"isk":           c1.TaxRate,
					"loyalty_point": 0,
				},
				"type": "player_owned",
			}),
		)
		// when
		got, err := s.UpdateAllCorporationsESI(t.Context())
		// then
		require.NoError(t, err)
		c2, err := st.GetEveCorporation(t.Context(), c1.ID)
		require.NoError(t, err)
		assert.Equal(t, c2, c1)
		want := set.Of[int64]()
		xassert.Equal(t, want, got)
	})
}

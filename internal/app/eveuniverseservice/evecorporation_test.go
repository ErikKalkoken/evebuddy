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
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestGetEveCorporationESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverseservice.New(r, client)
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

package eveuniverseservice_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestMembershipHistory(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	eu := eveuniverseservice.New(r, client)
	ctx := context.Background()
	t.Run("should return corporation membership history", func(t *testing.T) {
		// given
		eu.Now = func() time.Time { return time.Date(2016, 7, 30, 20, 0, 0, 0, time.UTC) }
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
		x, err := eu.GetCharacterCorporationHistory(ctx, 42)
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
		eu.Now = func() time.Time { return time.Date(2016, 10, 30, 20, 0, 0, 0, time.UTC) }
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
		x, err := eu.GetCorporationAllianceHistory(ctx, 42)
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

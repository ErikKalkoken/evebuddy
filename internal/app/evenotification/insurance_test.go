package evenotification_test

import (
	"testing"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestInsurance_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("InsuranceExpirationMsg", func(t *testing.T) {
		text := "endDate: 132202241630000000\nshipID: 587\nshipName: My Precious\nstartDate: 132202242630000000\n"
		title, body, err := en.RenderESI(t.Context(), app.InsuranceExpirationMsg, optional.New(text), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "My Precious")
		assert.Contains(t, body, "My Precious")
	})

	t.Run("InsuranceFirstShipMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		shipType := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"isHouseWarmingGift": 1,
			"shipTypeID":         shipType.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.InsuranceFirstShipMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, shipType.Name)
		assert.Contains(t, body, shipType.Name)
		assert.Contains(t, body, "house warming gift")
	})

	t.Run("InsuranceInvalidatedMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		owner := f.CreateEveEntityCharacter()
		shipType := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"endDate":   132202243630000000,
			"ownerID":   owner.ID,
			"reason":    1,
			"shipID":    670,
			"startDate": 132202244630000000,
			"typeID":    shipType.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.InsuranceInvalidatedMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, shipType.Name)
		assert.Contains(t, body, owner.Name)
		assert.Contains(t, body, shipType.Name)
	})

	t.Run("InsuranceIssuedMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		shipType := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"endDate":   132202245630000000,
			"itemID":    1040000000005,
			"level":     0.5,
			"numWeeks":  1,
			"shipName":  "My Precious",
			"startDate": 132202246630000000,
			"typeID":    shipType.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.InsuranceIssuedMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "My Precious")
		assert.Contains(t, body, shipType.Name)
		assert.Contains(t, body, "50")
	})

	t.Run("InsurancePayoutMsg", func(t *testing.T) {
		text := "amount: 15000000.0\nitemID: 1040000000006\npayout: 1\n"
		title, body, err := en.RenderESI(t.Context(), app.InsurancePayoutMsg, optional.New(text), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "15,000,000")
		assert.Contains(t, body, "15,000,000")
	})
}

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

func TestBounty_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("BountyClaimMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		claimer := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"amount": 15_000_000.0,
			"charID": claimer.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.BountyClaimMsg, optional.New(string(text)), time.Now())

		require.NoError(t, err)
		assert.Equal(t, "Bounty claimed by "+claimer.Name, title)
		assert.Contains(t, body, claimer.Name)
		assert.Contains(t, body, "15,000,000")
	})

	t.Run("BountyESSShared", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"charID":   char.ID,
			"myIsk":    1_000_000.0,
			"totalIsk": 5_000_000.0,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.BountyESSShared, optional.New(string(text)), time.Now())

		require.NoError(t, err)
		assert.Equal(t, "ESS bank shared", title)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, "1,000,000")
		assert.Contains(t, body, "5,000,000")
	})

	t.Run("BountyESSTaken", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"charID":   char.ID,
			"myIsk":    1_000_000.0,
			"totalIsk": 5_000_000.0,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.BountyESSTaken, optional.New(string(text)), time.Now())

		require.NoError(t, err)
		assert.Equal(t, "ESS bank taken", title)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, "1,000,000")
		assert.Contains(t, body, "5,000,000")
	})

	t.Run("BountyPlacedAlliance", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		placer := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"bounty":         15_000_000.0,
			"bountyPlacerID": placer.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.BountyPlacedAlliance, optional.New(string(text)), time.Now())

		require.NoError(t, err)
		assert.Equal(t, "Bounty of 15,000,000 ISK placed on your alliance", title)
		assert.Contains(t, body, placer.Name)
	})

	t.Run("BountyPlacedChar", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		placer := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"bounty":         15_000_000.0,
			"bountyPlacerID": placer.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.BountyPlacedChar, optional.New(string(text)), time.Now())

		require.NoError(t, err)
		assert.Equal(t, "Bounty of 15,000,000 ISK placed on you", title)
		assert.Contains(t, body, placer.Name)
	})

	t.Run("BountyPlacedCorp", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		placer := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"bounty":         15_000_000.0,
			"bountyPlacerID": placer.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.BountyPlacedCorp, optional.New(string(text)), time.Now())

		require.NoError(t, err)
		assert.Equal(t, "Bounty of 15,000,000 ISK placed on your corporation", title)
		assert.Contains(t, body, placer.Name)
	})

	t.Run("BountyYourBountyClaimed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		victim := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"bounty":   15_000_000.0,
			"victimID": victim.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.BountyYourBountyClaimed, optional.New(string(text)), time.Now())

		require.NoError(t, err)
		assert.Equal(t, "Bounty of 15,000,000 ISK claimed", title)
		assert.Contains(t, body, victim.Name)
	})
}

func TestBounty_EntityIDs(t *testing.T) {
	en := evenotification.New(nil)

	t.Run("BountyClaimMsg returns charID", func(t *testing.T) {
		text := "amount: 15000000.0\ncharID: 95100002\n"
		ids, err := en.EntityIDs(app.BountyClaimMsg, optional.New(text))
		require.NoError(t, err)
		assert.True(t, ids.Contains(95100002))
	})

	t.Run("BountyPlacedCorp returns bountyPlacerID", func(t *testing.T) {
		text := "bounty: 15000000.0\nbountyPlacerID: 98100011\n"
		ids, err := en.EntityIDs(app.BountyPlacedCorp, optional.New(text))
		require.NoError(t, err)
		assert.True(t, ids.Contains(98100011))
	})
}

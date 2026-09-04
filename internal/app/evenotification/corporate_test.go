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

func TestCorporate2_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("BuddyConnectContactAdd", func(t *testing.T) {
		text := "level: 3\nmessage: 'Welcome!'\n"
		title, body, err := en.RenderESI(t.Context(), app.BuddyConnectContactAdd, optional.New(text), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Contact added", title)
		assert.Contains(t, body, "Welcome!")
	})

	t.Run("CharMedalMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		corp := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"corpID":  corp.ID,
			"medalID": 501,
			"reason":  "For outstanding service.",
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CharMedalMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Medal awarded", title)
		assert.Contains(t, body, corp.Name)
		assert.Contains(t, body, "outstanding service")
	})

	t.Run("CharTerminationMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		corp := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"charID":      char.ID,
			"corpID":      corp.ID,
			"roleName":    "Director",
			"roleNameIDs": []int64{3, 4},
			"security":    0.5,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CharTerminationMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, char.Name)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, corp.Name)
		assert.Contains(t, body, "Director")
	})

	appCases := []struct {
		name  string
		notif app.EveNotificationType
	}{
		{"CorpAppAcceptMsg", app.CorpAppAcceptMsg},
		{"CorpAppRejectMsg", app.CorpAppRejectMsg},
	}
	for _, tc := range appCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			httpmock.Reset()
			char := f.CreateEveEntityCharacter()
			corp := f.CreateEveEntityCorporation()
			text, err := yaml.Marshal(map[string]any{
				"applicationText": "Hi, this is Bruce Wayne.",
				"charID":          char.ID,
				"corpID":          corp.ID,
			})
			require.NoError(t, err)

			title, body, err := en.RenderESI(t.Context(), tc.notif, optional.New(string(text)), time.Now())
			require.NoError(t, err)
			assert.Contains(t, title, corp.Name)
			assert.Contains(t, body, char.Name)
			assert.Contains(t, body, corp.Name)
			assert.Contains(t, body, "Bruce Wayne")
		})
	}

	t.Run("CorpKicked", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		corp := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{"corpID": corp.ID})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CorpKicked, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, corp.Name)
		assert.Contains(t, body, corp.Name)
	})

	t.Run("CorpNewCEOMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		corp := f.CreateEveEntityCorporation()
		newCeo := f.CreateEveEntityCharacter()
		oldCeo := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"corpID":   corp.ID,
			"newCeoID": newCeo.ID,
			"oldCeoID": oldCeo.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CorpNewCEOMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, corp.Name)
		assert.Contains(t, body, newCeo.Name)
		assert.Contains(t, body, oldCeo.Name)
	})

	t.Run("CorpVoteMsg", func(t *testing.T) {
		text := "body: 'Please read this important corp update.'\nsubject: 'Corp Announcement'\n"
		title, body, err := en.RenderESI(t.Context(), app.CorpVoteMsg, optional.New(text), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Corp Announcement", title)
		assert.Equal(t, "Please read this important corp update.", body)
	})

	t.Run("CorpDividendMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		corp := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"amount":    15_000_000.0,
			"corpID":    corp.ID,
			"isMembers": false,
			"payout":    15_000_000.0,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CorpDividendMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, corp.Name)
		assert.Contains(t, body, "15,000,000")
	})

	t.Run("CorpLiquidationMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		corp := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"amount": 15_000_000.0,
			"corpID": corp.ID,
			"payout": 15_000_000.0,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CorpLiquidationMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, corp.Name)
		assert.Contains(t, body, "15,000,000")
	})

	t.Run("CorpNewsMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		corp := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"body":      "Please read this important corp update.",
			"corpID":    corp.ID,
			"inEffect":  1,
			"parameter": 1,
			"voteType":  1,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CorpNewsMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, corp.Name)
		assert.Contains(t, body, "important corp update")
	})

	t.Run("CorpTaxChangeMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		corp := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"corpID":     corp.ID,
			"newTaxRate": 0.1,
			"oldTaxRate": 0.05,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CorpTaxChangeMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, corp.Name)
		assert.Contains(t, body, "10.0")
		assert.Contains(t, body, "5.0")
	})

	friendlyFireCompletedCases := []struct {
		name  string
		notif app.EveNotificationType
	}{
		{"CorpFriendlyFireDisableTimerCompleted", app.CorpFriendlyFireDisableTimerCompleted},
		{"CorpFriendlyFireEnableTimerCompleted", app.CorpFriendlyFireEnableTimerCompleted},
	}
	for _, tc := range friendlyFireCompletedCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			httpmock.Reset()
			corp := f.CreateEveEntityCorporation()
			text, err := yaml.Marshal(map[string]any{"corpID": corp.ID})
			require.NoError(t, err)

			title, body, err := en.RenderESI(t.Context(), tc.notif, optional.New(string(text)), time.Now())
			require.NoError(t, err)
			assert.NotEmpty(t, title)
			assert.Contains(t, body, corp.Name)
		})
	}

	friendlyFireStartedCases := []struct {
		name  string
		notif app.EveNotificationType
	}{
		{"CorpFriendlyFireDisableTimerStarted", app.CorpFriendlyFireDisableTimerStarted},
		{"CorpFriendlyFireEnableTimerStarted", app.CorpFriendlyFireEnableTimerStarted},
	}
	for _, tc := range friendlyFireStartedCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			httpmock.Reset()
			char := f.CreateEveEntityCharacter()
			corp := f.CreateEveEntityCorporation()
			text, err := yaml.Marshal(map[string]any{
				"charID":       char.ID,
				"corpID":       corp.ID,
				"timeFinished": 132202239630000000,
			})
			require.NoError(t, err)

			title, body, err := en.RenderESI(t.Context(), tc.notif, optional.New(string(text)), time.Now())
			require.NoError(t, err)
			assert.NotEmpty(t, title)
			assert.Contains(t, body, char.Name)
			assert.Contains(t, body, corp.Name)
		})
	}

	t.Run("GiftReceived", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		sender := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"message":      "Enjoy!",
			"offerID":      702,
			"quantity":     5,
			"senderCharID": sender.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.GiftReceived, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, sender.Name)
		assert.Contains(t, body, "Enjoy!")
	})

	t.Run("CorpBecameWarEligible", func(t *testing.T) {
		title, body, err := en.RenderESI(t.Context(), app.CorpBecameWarEligible, optional.New("{}\n"), time.Now())
		require.NoError(t, err)
		assert.NotEmpty(t, title)
		assert.NotEmpty(t, body)
	})

	t.Run("CorpNoLongerWarEligible", func(t *testing.T) {
		title, body, err := en.RenderESI(t.Context(), app.CorpNoLongerWarEligible, optional.New("{}\n"), time.Now())
		require.NoError(t, err)
		assert.NotEmpty(t, title)
		assert.NotEmpty(t, body)
	})
}

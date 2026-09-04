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

func TestFactionWarfare_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	corpFactionCases := []struct {
		name  string
		notif app.EveNotificationType
	}{
		{"FacWarCorpJoinRequestMsg", app.FacWarCorpJoinRequestMsg},
		{"FacWarCorpJoinWithdrawMsg", app.FacWarCorpJoinWithdrawMsg},
		{"FacWarCorpLeaveRequestMsg", app.FacWarCorpLeaveRequestMsg},
		{"FacWarCorpLeaveWithdrawMsg", app.FacWarCorpLeaveWithdrawMsg},
		{"FWCorpJoinMsg", app.FWCorpJoinMsg},
		{"FWCorpLeaveMsg", app.FWCorpLeaveMsg},
	}
	for _, tc := range corpFactionCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			httpmock.Reset()
			corp := f.CreateEveEntityCorporation()
			faction := f.CreateEveEntityFaction()
			text, err := yaml.Marshal(map[string]any{
				"corpID":    corp.ID,
				"factionID": faction.ID,
			})
			require.NoError(t, err)

			title, body, err := en.RenderESI(t.Context(), tc.notif, optional.New(string(text)), time.Now())
			require.NoError(t, err)
			assert.Contains(t, title, corp.Name)
			assert.Contains(t, body, corp.Name)
			assert.Contains(t, body, faction.Name)
		})
	}

	t.Run("FacWarLPDisqualifiedEvent", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		corp := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"amount":               4,
			"charRefID":            char.ID,
			"corpID":               corp.ID,
			"disqualificationType": 1,
			"event":                1,
			"itemRefID":            90000003,
			"locationID":           1040000000001,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.FacWarLPDisqualifiedEvent, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Loyalty points disqualified", title)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, corp.Name)
		assert.Contains(t, body, "4")
	})

	t.Run("FacWarLPPayoutEvent", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		corp := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"amount":     4,
			"charRefID":  char.ID,
			"corpID":     corp.ID,
			"event":      1,
			"itemRefID":  90000007,
			"locationID": 1040000000003,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.FacWarLPPayoutEvent, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "4")
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, corp.Name)
	})

	t.Run("FWAllianceWarningMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		alliance := f.CreateEveEntityAlliance()
		faction := f.CreateEveEntityFaction()
		text, err := yaml.Marshal(map[string]any{
			"allianceID":       alliance.ID,
			"corpList":         "Aideron Robotics, Ivy League",
			"factionID":        faction.ID,
			"requiredStanding": 5.0,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.FWAllianceWarningMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, alliance.Name)
		assert.Contains(t, body, alliance.Name)
		assert.Contains(t, body, faction.Name)
		assert.Contains(t, body, "Aideron Robotics")
	})

	t.Run("FWCharRankGainMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		faction := f.CreateEveEntityFaction()
		text, err := yaml.Marshal(map[string]any{
			"factionID": faction.ID,
			"newRank":   3,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.FWCharRankGainMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "3")
		assert.Contains(t, body, faction.Name)
	})

	t.Run("FWCharRankLossMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		faction := f.CreateEveEntityFaction()
		text, err := yaml.Marshal(map[string]any{
			"factionID": faction.ID,
			"newRank":   1,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.FWCharRankLossMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "1")
		assert.Contains(t, body, faction.Name)
	})

	corpWarningCases := []struct {
		name  string
		notif app.EveNotificationType
	}{
		{"FWCorpKickMsg", app.FWCorpKickMsg},
		{"FWCorpWarningMsg", app.FWCorpWarningMsg},
	}
	for _, tc := range corpWarningCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			httpmock.Reset()
			corp := f.CreateEveEntityCorporation()
			faction := f.CreateEveEntityFaction()
			text, err := yaml.Marshal(map[string]any{
				"corpID":           corp.ID,
				"currentStanding":  1.0,
				"factionID":        faction.ID,
				"requiredStanding": 5.0,
			})
			require.NoError(t, err)

			title, body, err := en.RenderESI(t.Context(), tc.notif, optional.New(string(text)), time.Now())
			require.NoError(t, err)
			assert.Contains(t, title, corp.Name)
			assert.Contains(t, body, corp.Name)
			assert.Contains(t, body, faction.Name)
		})
	}
}

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

func TestWar2_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("AcceptedAlly", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		ally := f.CreateEveEntityCorporation()
		char := f.CreateEveEntityCharacter()
		enemy := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"allyID":   ally.ID,
			"charID":   char.ID,
			"enemyID":  enemy.ID,
			"iskValue": 3_375_000_000.0,
			"time":     132202249630000000,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.AcceptedAlly, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, ally.Name)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, enemy.Name)
	})

	t.Run("AcceptedSurrender", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		entity := f.CreateEveEntityCorporation()
		offering := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"charID":     char.ID,
			"entityID":   entity.ID,
			"iskValue":   3_375_000_000.0,
			"offeringID": offering.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.AcceptedSurrender, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, offering.Name)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, entity.Name)
	})

	t.Run("AllianceCapitalChanged", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		alliance := f.CreateEveEntityAlliance()
		solarSystem := f.CreateEveSolarSystem()
		text, err := yaml.Marshal(map[string]any{
			"allianceID":    alliance.ID,
			"solarSystemID": solarSystem.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.AllianceCapitalChanged, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, alliance.Name)
		assert.Contains(t, body, solarSystem.Name)
	})

	t.Run("AllWarCorpJoinedAllianceMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		alliance := f.CreateEveEntityAlliance()
		corp := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"allianceID": alliance.ID,
			"corpID":     corp.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.AllWarCorpJoinedAllianceMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, corp.Name)
		assert.Contains(t, body, alliance.Name)
	})

	warChangedCases := []struct {
		name  string
		notif app.EveNotificationType
	}{
		{"AllWarDeclaredMsg", app.AllWarDeclaredMsg},
		{"AllWarInvalidatedMsg", app.AllWarInvalidatedMsg},
		{"AllWarRetractedMsg", app.AllWarRetractedMsg},
		{"CorpWarDeclaredMsg", app.CorpWarDeclaredMsg},
		{"CorpWarInvalidatedMsg", app.CorpWarInvalidatedMsg},
		{"CorpWarRetractedMsg", app.CorpWarRetractedMsg},
	}
	for _, tc := range warChangedCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			httpmock.Reset()
			against := f.CreateEveEntityCorporation()
			declaredBy := f.CreateEveEntityCorporation()
			text, err := yaml.Marshal(map[string]any{
				"againstID":    against.ID,
				"cost":         15_000_000.0,
				"declaredByID": declaredBy.ID,
				"delayHours":   24,
				"hostileState": 0,
			})
			require.NoError(t, err)

			title, body, err := en.RenderESI(t.Context(), tc.notif, optional.New(string(text)), time.Now())
			require.NoError(t, err)
			assert.Contains(t, title, declaredBy.Name)
			assert.Contains(t, title, against.Name)
			assert.Contains(t, body, "15,000,000")
		})
	}

	t.Run("CorpWarFightingLegalMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		against := f.CreateEveEntityCorporation()
		declaredBy := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"againstID":    against.ID,
			"cost":         15_000_000.0,
			"declaredByID": declaredBy.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CorpWarFightingLegalMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, declaredBy.Name)
		assert.Contains(t, body, against.Name)
	})

	t.Run("AllyContractCancelled", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressor := f.CreateEveEntityCorporation()
		defender := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"aggressorID":  aggressor.ID,
			"defenderID":   defender.ID,
			"timeFinished": 132202236630000000,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.AllyContractCancelled, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Ally contract cancelled", title)
		assert.Contains(t, body, aggressor.Name)
		assert.Contains(t, body, defender.Name)
	})

	t.Run("AllyJoinedWarAggressorMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		ally := f.CreateEveEntityCorporation()
		defender := f.CreateEveEntityAlliance()
		text, err := yaml.Marshal(map[string]any{
			"allyID":     ally.ID,
			"defenderID": defender.ID,
			"startTime":  133207204770000000,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.AllyJoinedWarAggressorMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, ally.Name)
		assert.Contains(t, body, defender.Name)
	})

	t.Run("AllyJoinedWarAllyMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressor := f.CreateEveEntityAlliance()
		ally := f.CreateEveEntityAlliance()
		defender := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"aggressorID": aggressor.ID,
			"allyID":      ally.ID,
			"defenderID":  defender.ID,
			"startTime":   133215175400000000,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.AllyJoinedWarAllyMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, ally.Name)
		assert.Contains(t, body, aggressor.Name)
		assert.Contains(t, body, defender.Name)
	})

	t.Run("AllyJoinedWarDefenderMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressor := f.CreateEveEntityAlliance()
		ally := f.CreateEveEntityAlliance()
		text, err := yaml.Marshal(map[string]any{
			"aggressorID": aggressor.ID,
			"allyID":      ally.ID,
			"startTime":   132202250630000000,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.AllyJoinedWarDefenderMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, ally.Name)
		assert.Contains(t, body, aggressor.Name)
	})

	t.Run("MadeWarMutual", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		enemy := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"charID":  char.ID,
			"enemyID": enemy.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.MadeWarMutual, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, enemy.Name)
		assert.Contains(t, body, char.Name)
	})

	t.Run("OfferedSurrender", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		entity := f.CreateEveEntityCorporation()
		offered := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"charID":    char.ID,
			"entityID":  entity.ID,
			"iskValue":  0.0,
			"offeredID": offered.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.OfferedSurrender, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, offered.Name)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, entity.Name)
	})

	t.Run("OfferedToAlly", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		defender := f.CreateEveEntityCorporation()
		enemy := f.CreateEveEntityAlliance()
		text, err := yaml.Marshal(map[string]any{
			"charID":     char.ID,
			"defenderID": defender.ID,
			"enemyID":    enemy.ID,
			"iskValue":   0.0,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.OfferedToAlly, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, char.Name)
		assert.Contains(t, body, defender.Name)
		assert.Contains(t, body, enemy.Name)
	})

	t.Run("RetractsWar", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		enemy := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"charID":  char.ID,
			"enemyID": enemy.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.RetractsWar, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, enemy.Name)
		assert.Contains(t, body, char.Name)
	})

	t.Run("WarAllyOfferDeclinedMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressor := f.CreateEveEntityCorporation()
		ally := f.CreateEveEntityCorporation()
		char := f.CreateEveEntityCharacter()
		defender := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"aggressorID": aggressor.ID,
			"allyID":      ally.ID,
			"charID":      char.ID,
			"defenderID":  defender.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.WarAllyOfferDeclinedMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, ally.Name)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, aggressor.Name)
		assert.Contains(t, body, defender.Name)
	})

	t.Run("WarSurrenderDeclinedMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		owner := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"iskValue": 0.87542,
			"ownerID":  owner.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.WarSurrenderDeclinedMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, owner.Name)
		assert.Contains(t, body, owner.Name)
	})

	t.Run("WarSurrenderOfferMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		owner1 := f.CreateEveEntityCorporation()
		owner2 := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"iskValue":         3_375_000_000.0,
			"ownerID1":         owner1.ID,
			"ownerID2":         owner2.ID,
			"warNegotiationID": 470828,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.WarSurrenderOfferMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, owner1.Name)
		assert.Contains(t, title, owner2.Name)
		assert.Contains(t, body, "3,375,000,000")
	})

	t.Run("MercOfferedNegotiationMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressor := f.CreateEveEntityCorporation()
		defender := f.CreateEveEntityCorporation()
		merc := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"aggressorID": aggressor.ID,
			"defenderID":  defender.ID,
			"iskValue":    80_000_000.0,
			"mercID":      merc.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.MercOfferedNegotiationMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, merc.Name)
		assert.Contains(t, body, defender.Name)
		assert.Contains(t, body, aggressor.Name)
	})

	t.Run("MutualWarInviteSent", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		against := f.CreateEveEntityCorporation()
		declaredBy := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"againstID":       against.ID,
			"declaredByID":    declaredBy.ID,
			"expireTimeStamp": 132202234630922771,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.MutualWarInviteSent, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, against.Name)
		assert.Contains(t, body, declaredBy.Name)
		assert.Contains(t, body, against.Name)
	})

	t.Run("MercOfferRetractedMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressor := f.CreateEveEntityCorporation()
		defender := f.CreateEveEntityCorporation()
		merc := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"aggressorID": aggressor.ID,
			"defenderID":  defender.ID,
			"mercID":      merc.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.MercOfferRetractedMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, merc.Name)
		assert.Contains(t, body, defender.Name)
		assert.Contains(t, body, aggressor.Name)
	})
}

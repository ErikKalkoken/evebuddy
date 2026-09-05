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

func TestMisc_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("IncursionCompletedMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		solarSystem := f.CreateEveSolarSystem()
		pilot := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"emailMessageId": 1.0,
			"solarSystemID":  solarSystem.ID,
			"taleID":         90000010,
			"topTen":         [][]int64{{pilot.ID, 1_500_000_000}},
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.IncursionCompletedMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, solarSystem.Name)
		assert.Contains(t, body, solarSystem.Name)
		assert.Contains(t, body, pilot.Name)
	})

	t.Run("IndustryTeamAuctionLost", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		solarSystem := f.CreateEveSolarSystem()
		text, err := yaml.Marshal(map[string]any{
			"solarSystemID": solarSystem.ID,
			"systemBids":    map[int64]float64{30100006: 550_000_000.0},
			"teamNameInfo":  []any{},
			"totalIsk":      1_250_000_000.0,
			"yourAmount":    50_000_000.0,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.IndustryTeamAuctionLost, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Industry team auction lost", title)
		assert.Contains(t, body, solarSystem.Name)
		assert.Contains(t, body, "1,250,000,000")
	})

	t.Run("LocateCharMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		solarSystem := f.CreateEveSolarSystem()
		text, err := yaml.Marshal(map[string]any{
			"agentLocation": map[string]any{"3": 10000031, "4": 20000321, "5": 30100007, "15": 60100008},
			"characterID":   char.ID,
			"messageIndex":  1,
			"targetLocation": map[string]any{
				"3": 10000032, "4": 20000322, "5": solarSystem.ID, "15": 60100009,
			},
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.LocateCharMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, char.Name)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, solarSystem.Name)
	})

	t.Run("MissionOfferExpirationMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		rewardType := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"body":   []string{"This mission offer is about to expire."},
			"header": []string{"Mission Offer Expiring"},
			"missionKeywords": map[string]any{
				"objectiveDestinationID":       1040000000007,
				"objectiveDestinationSystemID": 30100009,
				"objectiveLocationID":          1040000000008,
				"objectiveLocationSystemID":    30100010,
				"objectiveQuantity":            1,
				"objectiveTypeID":              12235,
				"rewardQuantity":               1,
				"rewardTypeID":                 rewardType.ID,
			},
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.MissionOfferExpirationMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Mission offer expiring", title)
		assert.Contains(t, body, "about to expire")
		assert.Contains(t, body, rewardType.Name)
	})

	t.Run("OldLscMessages", func(t *testing.T) {
		text := "body: 'Please read this important corp update.'\nsubject: 'Corp Announcement'\n"
		title, body, err := en.RenderESI(t.Context(), app.OldLscMessages, optional.New(text), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Corp Announcement", title)
		assert.Equal(t, "Please read this important corp update.", body)
	})

	t.Run("OperationFinished", func(t *testing.T) {
		text := "operationID: 4000001\nrewards: \n  isk: 5000000\n"
		title, body, err := en.RenderESI(t.Context(), app.OperationFinished, optional.New(text), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Operation finished", title)
		assert.Contains(t, body, "5,000,000")
	})

	t.Run("ReimbursementMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		shipType := f.CreateEveType()
		solarSystem := f.CreateEveSolarSystem()
		text, err := yaml.Marshal(map[string]any{
			"addCloneInfo":  1,
			"shipTypeID":    shipType.ID,
			"solarSystemID": solarSystem.ID,
			"stationID":     60100010,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.ReimbursementMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, shipType.Name)
		assert.Contains(t, body, shipType.Name)
		assert.Contains(t, body, solarSystem.Name)
	})

	t.Run("ResearchMissionAvailableMsg", func(t *testing.T) {
		title, body, err := en.RenderESI(t.Context(), app.ResearchMissionAvailableMsg, optional.New("{}"), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Research mission available", title)
		assert.NotEmpty(t, body)
	})

	t.Run("SeasonalChallengeCompleted", func(t *testing.T) {
		text := "challenge_name_id: 90000011\nmax_progress: 100\npoints_awarded: 50\nprogress_text: 100\n"
		title, body, err := en.RenderESI(t.Context(), app.SeasonalChallengeCompleted, optional.New(text), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Seasonal challenge completed", title)
		assert.Contains(t, body, "50")
	})
}

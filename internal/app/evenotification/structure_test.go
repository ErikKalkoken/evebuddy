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

func TestStructure_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("SkyhookLostShields", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		solarSystem := f.CreateEveSolarSystem()
		text, err := yaml.Marshal(map[string]any{
			"itemID":              1040000000009,
			"planetID":            40200001,
			"planetShowInfoData":  []any{"showinfo", 2016, 40200001},
			"skyhookShowInfoData": []any{},
			"solarsystemID":       solarSystem.ID,
			"timeLeft":            3600000000,
			"timestamp":           133215175400000000,
			"typeID":              81826,
			"vulnerableTime":      7200000000,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.SkyhookLostShields, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, solarSystem.Name)
		assert.Contains(t, body, solarSystem.Name)
		assert.Contains(t, body, "Orbital Skyhook")
	})

	t.Run("SkyhookUnderAttack", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		solarSystem := f.CreateEveSolarSystem()
		text, err := yaml.Marshal(map[string]any{
			"allianceID":            99100004,
			"allianceLinkData":      []any{"showinfo", 16159, 99100004},
			"allianceName":          "C C P Alliance",
			"armorPercentage":       0.65,
			"charID":                char.ID,
			"corpLinkData":          []any{"showinfo", 2, 98100052},
			"corpName":              "Aideron Robotics",
			"hullPercentage":        1.0,
			"isActive":              true,
			"itemID":                "1040000000010",
			"planetID":              40200002,
			"planetShowInfoData":    []any{"showinfo", 2016, 40200002},
			"shieldPercentage":      0.0,
			"skyhookShowInfoData":   []any{"showinfo", 81826, 1040000000010},
			"solarsystemID":         solarSystem.ID,
			"structureID":           1040000000010,
			"structureShowInfoData": []any{"showinfo", 81826, 1040000000010},
			"typeID":                81826,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.SkyhookUnderAttack, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, solarSystem.Name)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, "Aideron Robotics")
		assert.Contains(t, body, "C C P Alliance")
	})

	stationServiceCases := []struct {
		name  string
		notif app.EveNotificationType
	}{
		{"StationServiceDisabled", app.StationServiceDisabled},
		{"StationServiceEnabled", app.StationServiceEnabled},
	}
	for _, tc := range stationServiceCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			httpmock.Reset()
			solarSystem := f.CreateEveSolarSystem()
			structureType := f.CreateEveEntityWithCategory(app.EveEntityInventoryType)
			text, err := yaml.Marshal(map[string]any{
				"solarSystemID":   solarSystem.ID,
				"structureTypeID": structureType.ID,
			})
			require.NoError(t, err)

			title, body, err := en.RenderESI(t.Context(), tc.notif, optional.New(string(text)), time.Now())
			require.NoError(t, err)
			assert.Contains(t, title, solarSystem.Name)
			assert.Contains(t, body, structureType.Name)
		})
	}

	t.Run("SkyhookDeployed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		solarSystem := f.CreateEveSolarSystem()
		text, err := yaml.Marshal(map[string]any{
			"itemID":              1040000000009,
			"ownerCorpLinkData":   []any{"showinfo", 2, 98100052},
			"ownerCorpName":       "Aideron Robotics",
			"planetID":            40200001,
			"planetShowInfoData":  []any{"showinfo", 2016, 40200001},
			"skyhookShowInfoData": []any{"showinfo", 81080, 1040000000009},
			"solarsystemID":       solarSystem.ID,
			"timeLeft":            18000000000,
			"typeID":              81080,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.SkyhookDeployed, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, solarSystem.Name)
		assert.Contains(t, body, "Aideron Robotics")
		assert.Contains(t, body, "Orbital Skyhook")
	})

	t.Run("SkyhookDestroyed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		solarSystem := f.CreateEveSolarSystem()
		text, err := yaml.Marshal(map[string]any{
			"itemID":              1040000000009,
			"planetID":            40200001,
			"planetShowInfoData":  []any{"showinfo", 2016, 40200001},
			"skyhookShowInfoData": []any{"showinfo", 81080, 1040000000009},
			"solarsystemID":       solarSystem.ID,
			"typeID":              81080,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.SkyhookDestroyed, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, solarSystem.Name)
		assert.Contains(t, body, "Orbital Skyhook")
	})

	t.Run("SkyhookOnline", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		solarSystem := f.CreateEveSolarSystem()
		text, err := yaml.Marshal(map[string]any{
			"itemID":              1040000000009,
			"planetID":            40200001,
			"planetShowInfoData":  []any{"showinfo", 2016, 40200001},
			"skyhookShowInfoData": []any{"showinfo", 81080, 1040000000009},
			"solarsystemID":       solarSystem.ID,
			"typeID":              81080,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.SkyhookOnline, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, solarSystem.Name)
		assert.Contains(t, body, "Orbital Skyhook")
	})

	reagentsAlertCases := []struct {
		name  string
		notif app.EveNotificationType
	}{
		{"StructureLowReagentsAlert", app.StructureLowReagentsAlert},
		{"StructureNoReagentsAlert", app.StructureNoReagentsAlert},
	}
	for _, tc := range reagentsAlertCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			httpmock.Reset()
			solarSystem := f.CreateEveSolarSystem()
			text, err := yaml.Marshal(map[string]any{
				"solarsystemID":         solarSystem.ID,
				"structureID":           1000000000001,
				"structureShowInfoData": []any{"showinfo", 81826, 1000000000001},
				"structureTypeID":       81826,
			})
			require.NoError(t, err)

			title, body, err := en.RenderESI(t.Context(), tc.notif, optional.New(string(text)), time.Now())
			require.NoError(t, err)
			assert.Contains(t, title, solarSystem.Name)
			assert.Contains(t, body, "Structure")
		})
	}
}

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
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestSov_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	damageCases := []struct {
		name  string
		notif app.EveNotificationType
	}{
		{"SovereigntyIHDamageMsg", app.SovereigntyIHDamageMsg},
		{"SovereigntySBUDamageMsg", app.SovereigntySBUDamageMsg},
		{"SovereigntyTCUDamageMsg", app.SovereigntyTCUDamageMsg},
	}
	for _, tc := range damageCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			httpmock.Reset()
			alliance := f.CreateEveEntityAlliance()
			corp := f.CreateEveEntityCorporation()
			aggressor := f.CreateEveEntityCharacter()
			solarSystem := f.CreateEveSolarSystem()
			text, err := yaml.Marshal(map[string]any{
				"aggressorAllianceID": alliance.ID,
				"aggressorCorpID":     corp.ID,
				"aggressorID":         aggressor.ID,
				"armorValue":          0.8,
				"hullValue":           0.9,
				"shieldValue":         0.7,
				"solarSystemID":       solarSystem.ID,
			})
			require.NoError(t, err)

			title, body, err := en.RenderESI(t.Context(), tc.notif, optional.New(string(text)), time.Now())
			require.NoError(t, err)
			assert.Contains(t, title, solarSystem.Name)
			assert.Contains(t, body, alliance.Name)
			assert.Contains(t, body, corp.Name)
			assert.Contains(t, body, aggressor.Name)
		})
	}

	t.Run("SovStationEnteredFreeport", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		solarSystem := f.CreateEveSolarSystem()
		structureType := f.CreateEveEntityWithCategory(app.EveEntityInventoryType)
		text, err := yaml.Marshal(map[string]any{
			"freeportexittime": 132202247630000000,
			"solarSystemID":    solarSystem.ID,
			"structureTypeID":  structureType.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.SovStationEnteredFreeport, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, solarSystem.Name)
		assert.Contains(t, body, structureType.Name)
	})

	t.Run("SovStructureSelfDestructCancel", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		solarSystem := f.CreateEveSolarSystem()
		structureType := f.CreateEveEntityWithCategory(app.EveEntityInventoryType)
		text, err := yaml.Marshal(map[string]any{
			"charID":          char.ID,
			"solarSystemID":   solarSystem.ID,
			"structureTypeID": structureType.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.SovStructureSelfDestructCancel, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, solarSystem.Name)
		assert.Contains(t, body, char.Name)
	})

	t.Run("SovStructureSelfDestructFinished", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		solarSystem := f.CreateEveSolarSystem()
		structureType := f.CreateEveEntityWithCategory(app.EveEntityInventoryType)
		text, err := yaml.Marshal(map[string]any{
			"solarSystemID":   solarSystem.ID,
			"structureTypeID": structureType.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.SovStructureSelfDestructFinished, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, solarSystem.Name)
		assert.Contains(t, body, structureType.Name)
	})

	t.Run("SovStructureSelfDestructRequested", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		solarSystem := f.CreateEveSolarSystem()
		structureType := f.CreateEveEntityWithCategory(app.EveEntityInventoryType)
		text, err := yaml.Marshal(map[string]any{
			"charID":          char.ID,
			"corpName":        "Aideron Robotics",
			"destructTime":    132202248630000000,
			"solarSystemID":   solarSystem.ID,
			"structureTypeID": structureType.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.SovStructureSelfDestructRequested, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, solarSystem.Name)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, "Aideron Robotics")
	})

	t.Run("AllAnchoringMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		alliance := f.CreateEveEntityAlliance()
		corp := f.CreateEveEntityCorporation()
		solarSystem := f.CreateEveSolarSystem()
		moon := f.CreateEveMoon(storage.CreateEveMoonParams{SolarSystemID: solarSystem.ID})
		typ := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"allianceID": alliance.ID,
			"corpID":     corp.ID,
			"corpsPresent": []map[string]any{
				{"allianceID": alliance.ID, "corpID": corp.ID},
			},
			"moonID":        moon.ID,
			"solarSystemID": solarSystem.ID,
			"towers": []map[string]any{
				{"moonID": moon.ID, "typeID": typ.ID},
			},
			"typeID": typ.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.AllAnchoringMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, solarSystem.Name)
		assert.Contains(t, body, corp.Name)
		assert.Contains(t, body, moon.Name)
	})
}

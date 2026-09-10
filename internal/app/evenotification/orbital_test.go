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

func TestOrbital_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("OrbitalAttacked full data", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressor := f.CreateEveEntityCharacter()
		aggressorCorp := f.CreateEveEntityCorporation()
		aggressorAlliance := f.CreateEveEntityAlliance()
		planet := f.CreateEvePlanet()
		structureType := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"aggressorAllianceID": aggressorAlliance.ID,
			"aggressorCorpID":     aggressorCorp.ID,
			"aggressorID":         aggressor.ID,
			"planetID":            planet.ID,
			"planetTypeID":        11,
			"shieldLevel":         0.5,
			"solarSystemID":       30000142,
			"typeID":              structureType.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.OrbitalAttacked, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "is under attack")
		assert.Contains(t, title, structureType.Name)
		assert.Contains(t, title, planet.Name)
		assert.Contains(t, body, aggressor.Name)
		assert.Contains(t, body, aggressorCorp.Name)
		assert.Contains(t, body, aggressorAlliance.Name)
	})

	t.Run("OrbitalAttacked without alliance", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressor := f.CreateEveEntityCharacter()
		aggressorCorp := f.CreateEveEntityCorporation()
		planet := f.CreateEvePlanet()
		structureType := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"aggressorAllianceID": 0,
			"aggressorCorpID":     aggressorCorp.ID,
			"aggressorID":         aggressor.ID,
			"planetID":            planet.ID,
			"planetTypeID":        11,
			"shieldLevel":         0.5,
			"solarSystemID":       30000142,
			"typeID":              structureType.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.OrbitalAttacked, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "is under attack")
		assert.Contains(t, body, aggressor.Name)
		assert.Contains(t, body, aggressorCorp.Name)
		assert.NotContains(t, body, "Alliance:")
	})

	t.Run("OrbitalReinforced full data", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressor := f.CreateEveEntityCharacter()
		aggressorCorp := f.CreateEveEntityCorporation()
		aggressorAlliance := f.CreateEveEntityAlliance()
		planet := f.CreateEvePlanet()
		structureType := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"aggressorAllianceID": aggressorAlliance.ID,
			"aggressorCorpID":     aggressorCorp.ID,
			"aggressorID":         aggressor.ID,
			"planetID":            planet.ID,
			"planetTypeID":        11,
			"reinforceExitTime":   132202238630000000,
			"solarSystemID":       30000142,
			"typeID":              structureType.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.OrbitalReinforced, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "has been reinforced")
		assert.Contains(t, title, structureType.Name)
		assert.Contains(t, title, planet.Name)
		assert.Contains(t, body, aggressor.Name)
		assert.Contains(t, body, aggressorCorp.Name)
		assert.Contains(t, body, aggressorAlliance.Name)
	})

	t.Run("OrbitalReinforced without alliance", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressor := f.CreateEveEntityCharacter()
		aggressorCorp := f.CreateEveEntityCorporation()
		planet := f.CreateEvePlanet()
		structureType := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"aggressorAllianceID": 0,
			"aggressorCorpID":     aggressorCorp.ID,
			"aggressorID":         aggressor.ID,
			"planetID":            planet.ID,
			"planetTypeID":        11,
			"reinforceExitTime":   132202238630000000,
			"solarSystemID":       30000142,
			"typeID":              structureType.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.OrbitalReinforced, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "has been reinforced")
		assert.Contains(t, body, aggressor.Name)
		assert.Contains(t, body, aggressorCorp.Name)
		assert.NotContains(t, body, "Alliance:")
	})
}

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

func TestCustoms_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("ContainerPasswordMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		station := f.CreateEveEntityWithCategory(app.EveEntityStation)
		solarSystem := f.CreateEveSolarSystem()
		containerType := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"charID":        char.ID,
			"password":      "hunter2",
			"passwordType":  "Cargo",
			"solarSystemID": solarSystem.ID,
			"stationID":     station.ID,
			"typeID":        containerType.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.ContainerPasswordMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, containerType.Name)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, station.Name)
		assert.Contains(t, body, solarSystem.Name)
		assert.Contains(t, body, "hunter2")
	})

	t.Run("CustomsMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		faction := f.CreateEveEntityFaction()
		solarSystem := f.CreateEveSolarSystem()
		lostType := f.CreateEveEntityWithCategory(app.EveEntityInventoryType)
		text, err := yaml.Marshal(map[string]any{
			"factionID": faction.ID,
			"lostList": []map[string]any{
				{"fine": 1_050_000.0, "penalty": 500_000.0, "quantity": 12, "typeID": lostType.ID},
			},
			"securityLevel":    0.8,
			"shouldAttack":     0,
			"shouldConfiscate": 1,
			"solarSystemID":    solarSystem.ID,
			"standingDivision": 1.0,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CustomsMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, solarSystem.Name)
		assert.Contains(t, body, faction.Name)
		assert.Contains(t, body, lostType.Name)
	})
}

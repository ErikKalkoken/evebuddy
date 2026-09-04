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

func TestKillRight_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("KillReportFinalBlow", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		victim := f.CreateEveEntityCharacter()
		victimShip := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"killMailHash":     "abc123",
			"killMailID":       93670303,
			"victimID":         victim.ID,
			"victimShipTypeID": victimShip.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.KillReportFinalBlow, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, victim.Name)
		assert.Contains(t, body, victim.Name)
		assert.Contains(t, body, victimShip.Name)
	})

	t.Run("KillReportVictim", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		victimShip := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"killMailHash":     "def456",
			"killMailID":       84685597,
			"victimShipTypeID": victimShip.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.KillReportVictim, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, victimShip.Name)
		assert.Contains(t, body, victimShip.Name)
	})

	t.Run("KillRightAvailable", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		toEntity := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"charID":     char.ID,
			"price":      15_000_000.0,
			"toEntityID": toEntity.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.KillRightAvailable, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, char.Name)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, toEntity.Name)
		assert.Contains(t, body, "15,000,000")
	})

	t.Run("KillRightAvailableOpen", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"charID": char.ID,
			"price":  15_000_000.0,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.KillRightAvailableOpen, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, char.Name)
		assert.Contains(t, body, char.Name)
		assert.Contains(t, body, "anyone")
	})

	t.Run("KillRightEarned", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{"charID": char.ID})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.KillRightEarned, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, char.Name)
		assert.Contains(t, body, char.Name)
	})

	t.Run("KillRightUnavailable", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		toEntity := f.CreateEveEntityCorporation()
		text, err := yaml.Marshal(map[string]any{
			"charID":     char.ID,
			"toEntityID": toEntity.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.KillRightUnavailable, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, char.Name)
		assert.Contains(t, body, toEntity.Name)
	})

	t.Run("KillRightUnavailableOpen", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{"charID": char.ID})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.KillRightUnavailableOpen, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, char.Name)
		assert.Contains(t, body, char.Name)
	})

	t.Run("KillRightUsed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		char := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{"charID": char.ID})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.KillRightUsed, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, char.Name)
		assert.Contains(t, body, char.Name)
	})
}

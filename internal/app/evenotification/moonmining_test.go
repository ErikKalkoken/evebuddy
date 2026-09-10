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

func TestMoonmining_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("MoonminingAutomaticFracture", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		moon := f.CreateEveMoon()
		ore := f.CreateEveEntityWithCategory(app.EveEntityInventoryType)
		text, err := yaml.Marshal(map[string]any{
			"moonID":          moon.ID,
			"oreVolumeByType": map[int64]float64{ore.ID: 1234.5},
			"structureID":     1000000000001,
			"structureName":   "Munin - Mining Outpost",
			"structureTypeID": 35835,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.MoonminingAutomaticFracture, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "Munin - Mining Outpost")
		assert.Contains(t, body, moon.Name)
		assert.Contains(t, body, ore.Name)
	})

	t.Run("MoonminingExtractionStarted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		moon := f.CreateEveMoon()
		ore := f.CreateEveEntityWithCategory(app.EveEntityInventoryType)
		text, err := yaml.Marshal(map[string]any{
			"autoTime":        132202238630000000,
			"moonID":          moon.ID,
			"oreVolumeByType": map[int64]float64{ore.ID: 500},
			"readyTime":       132202237630000000,
			"structureID":     1000000000001,
			"structureName":   "Munin - Mining Outpost",
			"structureTypeID": 35835,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.MoonminingExtractionStarted, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "Munin - Mining Outpost")
		assert.Contains(t, body, moon.Name)
		assert.Contains(t, body, ore.Name)
	})

	t.Run("MoonminingExtractionFinished", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		moon := f.CreateEveMoon()
		ore := f.CreateEveEntityWithCategory(app.EveEntityInventoryType)
		text, err := yaml.Marshal(map[string]any{
			"autoTime":        132202238630000000,
			"moonID":          moon.ID,
			"oreVolumeByType": map[int64]float64{ore.ID: 500},
			"structureID":     1000000000001,
			"structureName":   "Munin - Mining Outpost",
			"structureTypeID": 35835,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.MoonminingExtractionFinished, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "Munin - Mining Outpost")
		assert.Contains(t, body, moon.Name)
		assert.Contains(t, body, ore.Name)
	})

	t.Run("MoonminingExtractionCancelled with canceller", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		moon := f.CreateEveMoon()
		canceller := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"cancelledBy":     canceller.ID,
			"moonID":          moon.ID,
			"structureID":     1000000000001,
			"structureName":   "Munin - Mining Outpost",
			"structureTypeID": 35835,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.MoonminingExtractionCancelled, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "Munin - Mining Outpost")
		assert.Contains(t, body, moon.Name)
		assert.Contains(t, body, canceller.Name)
	})

	t.Run("MoonminingExtractionCancelled without canceller", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		moon := f.CreateEveMoon()
		text, err := yaml.Marshal(map[string]any{
			"cancelledBy":     0,
			"moonID":          moon.ID,
			"structureID":     1000000000001,
			"structureName":   "Munin - Mining Outpost",
			"structureTypeID": 35835,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.MoonminingExtractionCancelled, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "Munin - Mining Outpost")
		assert.Contains(t, body, moon.Name)
		assert.NotContains(t, body, " by ")
	})

	t.Run("MoonminingLaserFired with firer", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		moon := f.CreateEveMoon()
		ore := f.CreateEveEntityWithCategory(app.EveEntityInventoryType)
		firer := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"firedBy":         firer.ID,
			"moonID":          moon.ID,
			"oreVolumeByType": map[int64]float64{ore.ID: 500},
			"structureID":     1000000000001,
			"structureName":   "Munin - Mining Outpost",
			"structureTypeID": 35835,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.MoonminingLaserFired, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "Munin - Mining Outpost")
		assert.Contains(t, body, moon.Name)
		assert.Contains(t, body, ore.Name)
		assert.Contains(t, body, firer.Name)
	})

	t.Run("MoonminingLaserFired without firer", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		moon := f.CreateEveMoon()
		ore := f.CreateEveEntityWithCategory(app.EveEntityInventoryType)
		text, err := yaml.Marshal(map[string]any{
			"firedBy":         0,
			"moonID":          moon.ID,
			"oreVolumeByType": map[int64]float64{ore.ID: 500},
			"structureID":     1000000000001,
			"structureName":   "Munin - Mining Outpost",
			"structureTypeID": 35835,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.MoonminingLaserFired, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Contains(t, title, "Munin - Mining Outpost")
		assert.Contains(t, body, moon.Name)
		assert.Contains(t, body, ore.Name)
	})
}

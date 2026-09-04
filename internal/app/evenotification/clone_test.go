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

func TestClone_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("CloneActivationMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		podKiller := f.CreateEveEntityCharacter()
		cloneStation := f.CreateEveEntityWithCategory(app.EveEntityStation)
		corpStation := f.CreateEveEntityWithCategory(app.EveEntityStation)
		skill := f.CreateEveType()
		text, err := yaml.Marshal(map[string]any{
			"cloneBought":     0,
			"cloneStationID":  cloneStation.ID,
			"cloneTypeID":     34,
			"corpStationID":   corpStation.ID,
			"lastCloned":      132202237630000000,
			"podKillerID":     podKiller.ID,
			"skillID":         skill.ID,
			"skillPointsLost": 12000,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CloneActivationMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Clone activated", title)
		assert.Contains(t, body, podKiller.Name)
		assert.Contains(t, body, cloneStation.Name)
		assert.Contains(t, body, skill.Name)
		assert.Contains(t, body, "12000")
	})

	t.Run("CloneActivationMsg2", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		podKiller := f.CreateEveEntityCharacter()
		cloneStation := f.CreateEveEntityWithCategory(app.EveEntityStation)
		corpStation := f.CreateEveEntityWithCategory(app.EveEntityStation)
		text, err := yaml.Marshal(map[string]any{
			"cloneStationID": cloneStation.ID,
			"corpStationID":  corpStation.ID,
			"lastCloned":     132202238630000000,
			"podKillerID":    podKiller.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CloneActivationMsg2, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Clone activated", title)
		assert.Contains(t, body, podKiller.Name)
		assert.Contains(t, body, cloneStation.Name)
	})

	t.Run("CloneMovedMsg", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		corp := f.CreateEveEntityCorporation()
		newStation := f.CreateEveEntityWithCategory(app.EveEntityStation)
		station := f.CreateEveEntityWithCategory(app.EveEntityStation)
		text, err := yaml.Marshal(map[string]any{
			"charsInCorpID": corp.ID,
			"corpID":        corp.ID,
			"newStationID":  newStation.ID,
			"stationID":     station.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CloneMovedMsg, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Medical clone moved", title)
		assert.Contains(t, body, station.Name)
		assert.Contains(t, body, newStation.Name)
		assert.Contains(t, body, corp.Name)
	})

	t.Run("CloneRevokedMsg2", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		corp := f.CreateEveEntityCorporation()
		newStation := f.CreateEveEntityWithCategory(app.EveEntityStation)
		station := f.CreateEveLocationStation()
		text, err := yaml.Marshal(map[string]any{
			"corpID":       corp.ID,
			"newStationID": newStation.ID,
			"stationID":    station.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.CloneRevokedMsg2, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Clone contract revoked", title)
		assert.Contains(t, body, corp.Name)
		assert.Contains(t, body, station.Name)
		assert.Contains(t, body, newStation.Name)
	})

	t.Run("JumpCloneDeletedMsg1", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		owner := f.CreateEveEntityCharacter()
		locationOwner := f.CreateEveEntityCorporation()
		location := f.CreateEveLocationStructure()
		implant := f.CreateEveEntityWithCategory(app.EveEntityInventoryType)
		text, err := yaml.Marshal(map[string]any{
			"locationID":      location.ID,
			"locationOwnerID": locationOwner.ID,
			"ownerID":         owner.ID,
			"typeIDs":         []int64{implant.ID},
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.JumpCloneDeletedMsg1, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Jump clone deleted", title)
		assert.Contains(t, body, owner.Name)
		assert.Contains(t, body, locationOwner.Name)
		assert.Contains(t, body, implant.Name)
	})

	t.Run("JumpCloneDeletedMsg2", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		owner := f.CreateEveEntityCharacter()
		locationOwner := f.CreateEveEntityCorporation()
		destroyer := f.CreateEveEntityCharacter()
		location := f.CreateEveLocationStructure()
		text, err := yaml.Marshal(map[string]any{
			"destroyerID":     destroyer.ID,
			"locationID":      location.ID,
			"locationOwnerID": locationOwner.ID,
			"ownerID":         owner.ID,
			"typeIDs":         []int64{},
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.JumpCloneDeletedMsg2, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Jump clone deleted", title)
		assert.Contains(t, body, owner.Name)
		assert.Contains(t, body, destroyer.Name)
	})
}

package screens

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestColonies(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	now := time.Now().UTC()
	lastUpdate := now.Add(-65 * time.Minute)
	cp := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{LastUpdate: lastUpdate})
	ecuGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{ID: app.EveGroupExtractorControlUnits})
	ecuType := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: ecuGroup.ID})
	storageGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{ID: app.EveGroupStorageFacilities})
	storageType := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: storageGroup.ID})
	product := factory.CreateEveType(storage.CreateEveTypeParams{Volume: optional.New(0.01)})
	expiry := lastUpdate.Add(4 * time.Hour)
	factory.CreatePlanetPin(storage.CreatePlanetPinParams{
		CharacterPlanetID:      cp.ID,
		PinID:                  1,
		TypeID:                 ecuType.ID,
		ExtractorProductTypeID: optional.New(product.ID),
		ExtractorQtyPerCycle:   optional.New[int64](1081),
		ExtractorCycleTime:     optional.New(30 * time.Minute),
		InstallTime:            optional.New(lastUpdate),
		ExpiryTime:             optional.New(expiry),
		LastCycleStart:         optional.New(lastUpdate),
	})
	factory.CreatePlanetPin(storage.CreatePlanetPinParams{
		CharacterPlanetID: cp.ID,
		PinID:             2,
		TypeID:            storageType.ID,
	})
	factory.CreatePlanetRoute(storage.CreatePlanetRouteParams{
		CharacterPlanetID: cp.ID,
		SourcePinID:       1,
		DestinationPinID:  2,
		ContentTypeID:     product.ID,
		Quantity:          10_000,
	})
	factory.CreateCharacterPlanet() // colony without pins

	newColonies := func(t *testing.T) *Colonies {
		return NewColonies(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
	}
	statusByPlanet := func(rows []colonyRow) map[int64]app.ColonyStatus {
		m := make(map[int64]app.ColonyStatus)
		for _, r := range rows {
			m[r.planetID] = r.status
		}
		return m
	}

	t.Run("should show forecasted status and work end", func(t *testing.T) {
		a := newColonies(t)
		rows, err := a.fetchRows(t.Context())
		require.NoError(t, err)
		require.Len(t, rows, 2)
		assert.Equal(t, app.ColonyExtracting, statusByPlanet(rows)[cp.EvePlanet.ID])
		for _, r := range rows {
			if r.planetID == cp.EvePlanet.ID {
				assert.True(t, r.workEndsAt.MustValue().Equal(expiry))
			} else {
				assert.Equal(t, app.ColonyIdle, r.status)
				assert.True(t, r.workEndsAt.IsEmpty())
			}
		}
	})
	t.Run("should recalculate forecasts", func(t *testing.T) {
		a := newColonies(t)
		a.Update(t.Context())
		for i := range a.rows {
			a.rows[i].status = app.ColonyStatusUndefined
		}
		a.refreshForecasts()
		assert.Equal(t, app.ColonyExtracting, statusByPlanet(a.rows)[cp.EvePlanet.ID])
	})
	t.Run("should not invalidate a running update when refreshing", func(t *testing.T) {
		a := newColonies(t)
		isLatest := a.rowsRun.start() // simulates an update in flight
		a.refreshForecasts()
		assert.True(t, isLatest())
	})
	t.Run("should discard a refresh when update replaced the rows", func(t *testing.T) {
		var pending []func()
		orig := runAsync
		runAsync = func(f func()) { pending = append(pending, f) }
		t.Cleanup(func() { runAsync = orig })
		a := newColonies(t)
		a.refreshForecasts() // started with no rows
		a.Update(t.Context())
		for len(pending) > 0 {
			f := pending[0]
			pending = pending[1:]
			f()
		}
		assert.Len(t, a.rows, 2)
		assert.Len(t, a.rowsFiltered, 2)
	})
	t.Run("can filter by status", func(t *testing.T) {
		a := newColonies(t)
		a.Update(t.Context())
		a.selectStatus.SetSelected(app.ColonyExtracting.Display())
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.Equal(t, cp.EvePlanet.ID, a.rowsFiltered[0].planetID)
		}
	})
}

package characterservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestNotifyStoppedColonies(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	earliest := now.Add(-24 * time.Hour)

	type colonyParams struct {
		characterID   int64
		lastUpdate    time.Time
		lastNotified  time.Time
		expiry        time.Time // of the extractor
		productVolume float64   // m3 per unit
		capacity      float64   // m3 of the storage
		routeQuantity int64
	}
	// createColony creates a colony with an extractor routed into a storage.
	createColony := func(arg colonyParams) *app.CharacterPlanet {
		if arg.productVolume == 0 {
			arg.productVolume = 0.01
		}
		if arg.capacity == 0 {
			arg.capacity = 12_000
		}
		if arg.routeQuantity == 0 {
			arg.routeQuantity = 10_000
		}
		ecuType := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: app.EveGroupExtractorControlUnits})
		storageType := factory.CreateEveType(storage.CreateEveTypeParams{
			GroupID:  app.EveGroupStorageFacilities,
			Capacity: optional.New(arg.capacity),
		})
		product := factory.CreateEveType(storage.CreateEveTypeParams{Volume: optional.New(arg.productVolume)})
		p := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
			CharacterID:  arg.characterID,
			LastUpdate:   arg.lastUpdate,
			LastNotified: arg.lastNotified,
		})
		factory.CreatePlanetPin(storage.CreatePlanetPinParams{
			CharacterPlanetID:      p.ID,
			PinID:                  1,
			TypeID:                 ecuType.ID,
			ExtractorProductTypeID: optional.New(product.ID),
			ExtractorQtyPerCycle:   optional.New[int64](1081),
			ExtractorCycleTime:     optional.New(30 * time.Minute),
			InstallTime:            optional.New(arg.lastUpdate),
			ExpiryTime:             optional.New(arg.expiry),
			LastCycleStart:         optional.New(arg.lastUpdate),
		})
		factory.CreatePlanetPin(storage.CreatePlanetPinParams{
			CharacterPlanetID: p.ID,
			PinID:             2,
			TypeID:            storageType.ID,
		})
		factory.CreatePlanetRoute(storage.CreatePlanetRouteParams{
			CharacterPlanetID: p.ID,
			SourcePinID:       1,
			DestinationPinID:  2,
			ContentTypeID:     product.ID,
			Quantity:          arg.routeQuantity,
		})
		return p
	}
	reset := func() {
		testutil.MustTruncateTables(db)
		for _, id := range []int64{
			app.EveGroupExtractorControlUnits,
			app.EveGroupStorageFacilities,
		} {
			factory.CreateEveGroup(storage.CreateEveGroupParams{ID: id})
		}
	}
	notify := func(t *testing.T, characterID int64) (int, string, string) {
		var count int
		var title, content string
		err := cs.NotifyStoppedColonies(ctx, characterID, earliest, func(t, c string) {
			count++
			title, content = t, c
		})
		require.NoError(t, err)
		return count, title, content
	}

	t.Run("should notify when extractor expired after the snapshot", func(t *testing.T) {
		reset()
		lastUpdate := now.Add(-3 * time.Hour)
		p := createColony(colonyParams{lastUpdate: lastUpdate, expiry: lastUpdate.Add(time.Hour)})
		count, title, content := notify(t, p.CharacterID)
		xassert.Equal(t, 1, count)
		assert.Contains(t, title, "1 planet(s)")
		assert.Contains(t, content, p.EvePlanet.Name)
		assert.Contains(t, content, app.PinExtractorExpired.Display())
	})
	t.Run("should notify when storage is full", func(t *testing.T) {
		reset()
		lastUpdate := now.Add(-4 * time.Hour)
		p := createColony(colonyParams{
			lastUpdate:    lastUpdate,
			expiry:        now.Add(24 * time.Hour),
			productVolume: 1,
			capacity:      500,
			routeQuantity: 100, // fills the storage after 5 cycles
		})
		count, _, content := notify(t, p.CharacterID)
		xassert.Equal(t, 1, count)
		assert.Contains(t, content, app.PinStorageFull.Display())
	})
	t.Run("should not notify while colony is working", func(t *testing.T) {
		reset()
		p := createColony(colonyParams{lastUpdate: now.Add(-time.Hour), expiry: now.Add(3 * time.Hour)})
		count, _, _ := notify(t, p.CharacterID)
		xassert.Equal(t, 0, count)
	})
	t.Run("should not notify when colony was not working at the snapshot", func(t *testing.T) {
		reset()
		lastUpdate := now.Add(-3 * time.Hour)
		p := createColony(colonyParams{lastUpdate: lastUpdate, expiry: lastUpdate.Add(-time.Hour)})
		count, _, _ := notify(t, p.CharacterID)
		xassert.Equal(t, 0, count)
	})
	t.Run("should notify once per snapshot", func(t *testing.T) {
		reset()
		lastUpdate := now.Add(-3 * time.Hour)
		p := createColony(colonyParams{lastUpdate: lastUpdate, expiry: lastUpdate.Add(time.Hour)})
		count1, _, _ := notify(t, p.CharacterID)
		count2, _, _ := notify(t, p.CharacterID)
		xassert.Equal(t, 1, count1)
		xassert.Equal(t, 0, count2)
	})
	t.Run("should notify again for a new snapshot", func(t *testing.T) {
		reset()
		lastUpdate := now.Add(-3 * time.Hour)
		p := createColony(colonyParams{
			lastUpdate:   lastUpdate,
			lastNotified: lastUpdate.Add(-24 * time.Hour), // notified for an older snapshot
			expiry:       lastUpdate.Add(time.Hour),
		})
		count, _, _ := notify(t, p.CharacterID)
		xassert.Equal(t, 1, count)
	})
	t.Run("should not notify when colony stopped before earliest", func(t *testing.T) {
		reset()
		lastUpdate := now.Add(-72 * time.Hour)
		p := createColony(colonyParams{lastUpdate: lastUpdate, expiry: lastUpdate.Add(time.Hour)})
		count, _, _ := notify(t, p.CharacterID)
		xassert.Equal(t, 0, count)
	})
	t.Run("should notify once for multiple stopped colonies", func(t *testing.T) {
		reset()
		c := factory.CreateCharacter()
		lastUpdate := now.Add(-3 * time.Hour)
		p1 := createColony(colonyParams{characterID: c.ID, lastUpdate: lastUpdate, expiry: lastUpdate.Add(time.Hour)})
		p2 := createColony(colonyParams{characterID: c.ID, lastUpdate: lastUpdate, expiry: lastUpdate.Add(time.Hour)})
		count, title, content := notify(t, c.ID)
		xassert.Equal(t, 1, count)
		assert.Contains(t, title, "2 planet(s)")
		assert.Contains(t, content, p1.EvePlanet.Name)
		assert.Contains(t, content, p2.EvePlanet.Name)
	})
}

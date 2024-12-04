package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestPlanet(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		lastUpdate := time.Now().UTC()
		evePlanet := factory.CreateEvePlanet()
		arg := storage.UpdateOrCreateCharacterPlanetParams{
			CharacterID:  c.ID,
			EvePlanetID:  evePlanet.ID,
			LastUpdate:   lastUpdate,
			UpgradeLevel: 3,
		}
		// when
		_, err := r.UpdateOrCreateCharacterPlanet(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := r.GetCharacterPlanet(ctx, c.ID, evePlanet.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, c.ID, i.CharacterID)
				assert.Equal(t, evePlanet, i.EvePlanet)
				assert.Equal(t, lastUpdate, i.LastUpdate)
				assert.Equal(t, 3, i.UpgradeLevel)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		evePlanet := factory.CreateEvePlanet()
		lastNotified := time.Now().Add(-5 * time.Minute).UTC()
		factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
			CharacterID:  c.ID,
			EvePlanetID:  evePlanet.ID,
			LastUpdate:   time.Now().Add(-1 * time.Hour).UTC(),
			LastNotified: lastNotified,
			UpgradeLevel: 2,
		})
		lastUpdate := time.Now().UTC()
		arg := storage.UpdateOrCreateCharacterPlanetParams{
			CharacterID:  c.ID,
			EvePlanetID:  evePlanet.ID,
			LastUpdate:   lastUpdate,
			UpgradeLevel: 3,
		}
		// when
		_, err := r.UpdateOrCreateCharacterPlanet(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := r.GetCharacterPlanet(ctx, c.ID, evePlanet.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, c.ID, i.CharacterID)
				assert.Equal(t, evePlanet, i.EvePlanet)
				assert.Equal(t, lastUpdate, i.LastUpdate)
				assert.Equal(t, lastNotified, i.LastNotified.ValueOrZero())
				assert.Equal(t, 3, i.UpgradeLevel)
			}
		}
	})
	t.Run("can list planets", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		p1 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p2 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p3 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		// when
		oo, err := r.ListCharacterPlanets(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 3)
			assert.ElementsMatch(
				t,
				[]int32{p1.EvePlanet.ID, p2.EvePlanet.ID, p3.EvePlanet.ID},
				[]int32{oo[0].EvePlanet.ID, oo[1].EvePlanet.ID, oo[2].EvePlanet.ID},
			)
		}
	})
	t.Run("can delete planets", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		p1 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p2 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p3 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		// when
		err := r.DeleteCharacterPlanet(ctx, c.ID, []int32{p1.EvePlanet.ID, p2.EvePlanet.ID})
		// then
		if assert.NoError(t, err) {
			oo, err := r.ListCharacterPlanets(ctx, c.ID)
			if err != nil {
				t.Fatal(err)
			}
			assert.Len(t, oo, 1)
			assert.ElementsMatch(t, []int32{p3.EvePlanet.ID}, []int32{oo[0].EvePlanet.ID})
		}
	})
	t.Run("can update last notified", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		planet := factory.CreateCharacterPlanet()
		lastNotified := factory.RandomTime()
		arg := storage.UpdateCharacterPlanetLastNotifiedParams{
			CharacterID:  planet.CharacterID,
			EvePlanetID:  planet.EvePlanet.ID,
			LastNotified: lastNotified,
		}
		// when
		err := r.UpdateCharacterPlanetLastNotified(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := r.GetCharacterPlanet(ctx, planet.CharacterID, planet.EvePlanet.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, lastNotified, i.LastNotified.ValueOrZero())
			}
		}
	})
}

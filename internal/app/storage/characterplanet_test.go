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
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		lastUpdate := time.Now().UTC()
		evePlanet := factory.CreateEvePlanet()
		arg := storage.CreateCharacterPlanetParams{
			CharacterID:  c.ID,
			EvePlanetID:  evePlanet.ID,
			LastUpdate:   lastUpdate,
			NumPins:      7,
			UpgradeLevel: 3,
		}
		// when
		_, err := r.CreateCharacterPlanet(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := r.GetCharacterPlanet(ctx, c.ID, evePlanet.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, c.ID, i.CharacterID)
				assert.Equal(t, evePlanet, i.EvePlanet)
				assert.Equal(t, lastUpdate, i.LastUpdate)
				assert.Equal(t, 7, i.NumPins)
				assert.Equal(t, 3, i.UpgradeLevel)
			}
		}
	})
	t.Run("can list items", func(t *testing.T) {
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
	t.Run("can replace character planets", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		lastUpdate := time.Now().UTC()
		evePlanet := factory.CreateEvePlanet()
		arg := storage.CreateCharacterPlanetParams{
			CharacterID:  c.ID,
			EvePlanetID:  evePlanet.ID,
			LastUpdate:   lastUpdate,
			NumPins:      7,
			UpgradeLevel: 3,
		}
		// when
		_, err := r.ReplaceCharacterPlanets(ctx, c.ID, []storage.CreateCharacterPlanetParams{arg})
		// then
		if assert.NoError(t, err) {
			i, err := r.GetCharacterPlanet(ctx, c.ID, evePlanet.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, c.ID, i.CharacterID)
				assert.Equal(t, evePlanet, i.EvePlanet)
				assert.Equal(t, lastUpdate, i.LastUpdate)
				assert.Equal(t, 7, i.NumPins)
				assert.Equal(t, 3, i.UpgradeLevel)
			}
		}
	})
}

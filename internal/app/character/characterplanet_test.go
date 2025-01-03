package character_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

// TODO: Add more test cases

func TestNotifyExpiredExtractions(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := newCharacterService(st)
	ctx := context.Background()
	t.Run("send notification when an extraction has expired and not notified", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		p := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		factory.CreatePlanetPinExtractor(storage.CreatePlanetPinParams{
			CharacterPlanetID: p.ID,
			ExpiryTime:        time.Now().Add(-3 * time.Hour),
		})
		earliest := time.Now().Add(-24 * time.Hour)
		var sendCount int
		// when
		err := cs.NotifyExpiredExtractions(ctx, c.ID, earliest, func(title string, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 1)
		}
	})
	t.Run("send no notification when extraction has expired and already notified", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		expires := time.Now().Add(-3 * time.Hour)
		p := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
			CharacterID:  c.ID,
			LastNotified: expires,
		})
		factory.CreatePlanetPinExtractor(storage.CreatePlanetPinParams{
			CharacterPlanetID: p.ID,
			ExpiryTime:        expires,
		})
		earliest := time.Now().Add(-24 * time.Hour)
		var sendCount int
		// when
		err := cs.NotifyExpiredExtractions(ctx, c.ID, earliest, func(title string, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 0)
		}
	})
}

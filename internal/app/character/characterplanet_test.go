package character_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNotifyExpiredExtractions(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := newCharacterService(st)
	ctx := context.Background()
	now := time.Now().UTC()
	earliest := now.Add(-24 * time.Hour)
	cases := []struct {
		name         string
		isExtractor  bool
		expiryTime   time.Time
		lastNotified time.Time
		shouldNotify bool
	}{
		{"extraction expired and not yet notified", true, now.Add(-3 * time.Hour), time.Time{}, true},
		{"extraction expired and already notified", true, now.Add(-3 * time.Hour), now.Add(-3 * time.Hour), false},
		{"extraction not expired", true, now.Add(3 * time.Hour), time.Time{}, false},
		{"extraction expired old", true, now.Add(-48 * time.Hour), time.Time{}, false},
		{"no expiration date", true, time.Time{}, time.Time{}, false},
		{"non extractor expired", false, now.Add(-3 * time.Hour), time.Time{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.TruncateTables(db)
			p := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
				LastNotified: tc.lastNotified,
			})
			if tc.isExtractor {
				factory.CreatePlanetPinExtractor(storage.CreatePlanetPinParams{
					CharacterPlanetID: p.ID,
					ExpiryTime:        tc.expiryTime,
				})
			} else {
				factory.CreatePlanetPin(storage.CreatePlanetPinParams{
					CharacterPlanetID: p.ID,
					ExpiryTime:        tc.expiryTime,
				})
			}
			var sendCount int
			// when
			err := cs.NotifyExpiredExtractions(ctx, p.CharacterID, earliest, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				assert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

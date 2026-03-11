package characterservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestNotifyExpiredExtractions(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := testdouble.NewCharacterService(characterservice.Params{Storage: st})
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
			testutil.MustTruncateTables(db)
			product := factory.CreateEveType()
			p := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
				LastNotified: tc.lastNotified,
			})
			if tc.isExtractor {
				factory.CreatePlanetPinExtractor(storage.CreatePlanetPinParams{
					CharacterPlanetID:      p.ID,
					ExpiryTime:             optional.New(tc.expiryTime),
					ExtractorProductTypeID: optional.New(product.ID),
				})
			} else {
				factory.CreatePlanetPin(storage.CreatePlanetPinParams{
					CharacterPlanetID: p.ID,
					ExpiryTime:        optional.New(tc.expiryTime),
				})
			}
			var sendCount int
			// when
			err := cs.NotifyExpiredExtractions(ctx, p.CharacterID, earliest, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				 xassert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

func TestNotifyExpiredExtractions_ShouldNoifyOnceForMultipleExpired(t *testing.T) {
	// given
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := testdouble.NewCharacterService(characterservice.Params{Storage: st})
	ctx := context.Background()
	now := time.Now().UTC()
	earliest := now.Add(-24 * time.Hour)
	testutil.MustTruncateTables(db)
	product := factory.CreateEveType()
	c := factory.CreateCharacter()
	p1 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
		CharacterID:  c.ID,
		LastNotified: time.Time{},
	})
	factory.CreatePlanetPinExtractor(storage.CreatePlanetPinParams{
		CharacterPlanetID:      p1.ID,
		ExpiryTime:             optional.New(now.Add(-3 * time.Hour)),
		ExtractorProductTypeID: optional.New(product.ID),
	})
	p2 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
		CharacterID:  c.ID,
		LastNotified: time.Time{},
	})
	factory.CreatePlanetPinExtractor(storage.CreatePlanetPinParams{
		CharacterPlanetID:      p2.ID,
		ExpiryTime:             optional.New(now.Add(-3 * time.Hour)),
		ExtractorProductTypeID: optional.New(product.ID),
	})
	// when
	var sendCount int
	var title, content string
	err := cs.NotifyExpiredExtractions(ctx, c.ID, earliest, func(t string, c string) {
		title = t
		content = c
		sendCount++
	})
	// then
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	 xassert.Equal(t, 1, sendCount)
	assert.Contains(t, content, p1.EvePlanet.Name)
	assert.Contains(t, content, p2.EvePlanet.Name)
	assert.Contains(t, title, "2")
}

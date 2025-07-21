package characterservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/stretchr/testify/assert"
)

func TestNotifyCommunications(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	now := time.Now().UTC()
	earliest := now.Add(-12 * time.Hour)
	typesEnabled := set.Of(string(evenotification.StructureUnderAttack))
	cases := []struct {
		name         string
		typ          evenotification.Type
		timestamp    time.Time
		isProcessed  bool
		shouldNotify bool
	}{
		{"send unprocessed", evenotification.StructureUnderAttack, now, false, true},
		{"don't send old unprocessed", evenotification.StructureUnderAttack, now.Add(-16 * time.Hour), false, false},
		{"don't send not enabled types", evenotification.SkyhookOnline, now, false, false},
		{"don't resend already processed", evenotification.StructureUnderAttack, now, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.TruncateTables(db)
			n := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
				IsProcessed: tc.isProcessed,
				Title:       optional.New("title"),
				Body:        optional.New("body"),
				Type:        string(tc.typ),
				Timestamp:   tc.timestamp,
			})
			var sendCount int
			// when
			err := cs.NotifyCommunications(ctx, n.CharacterID, earliest, typesEnabled, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				assert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

func TestCountNotifications(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	// given
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	c := factory.CreateCharacterFull()
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		CharacterID: c.ID,
		Type:        string(evenotification.StructureDestroyed),
	})
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		CharacterID: c.ID,
		Type:        string(evenotification.MoonminingExtractionStarted),
	})
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		CharacterID: c.ID,
		Type:        string(evenotification.MoonminingExtractionStarted),
		IsRead:      true,
	})
	factory.CreateCharacterNotification()
	// when
	got, err := cs.CountNotifications(ctx, c.ID)
	if assert.NoError(t, err) {
		want := map[app.NotificationGroup][]int{
			app.GroupStructure:  {1, 1},
			app.GroupMoonMining: {2, 1},
		}
		assert.Equal(t, want, got)
	}
}

package character_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/stretchr/testify/assert"
)

func TestNotifyCommunications(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := newCharacterService(st)
	ctx := context.Background()
	now := time.Now().UTC()
	earliest := now.Add(-12 * time.Hour)
	typesEnabled := set.New(string(evenotification.StructureUnderAttack))
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

package characterservice_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestNotifyCommunications(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	now := time.Now().UTC()
	earliest := now.Add(-12 * time.Hour)
	typesEnabled := set.Of(app.StructureUnderAttack)
	cases := []struct {
		name         string
		typ          app.EveNotificationType
		timestamp    time.Time
		isProcessed  bool
		shouldNotify bool
	}{
		{"send unprocessed", app.StructureUnderAttack, now, false, true},
		{"don't send old unprocessed", app.StructureUnderAttack, now.Add(-16 * time.Hour), false, false},
		{"don't send not enabled types", app.SkyhookOnline, now, false, false},
		{"don't resend already processed", app.StructureUnderAttack, now, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			s, _ := storage.EveNotificationTypeToESIString(tc.typ)
			n := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
				IsProcessed: tc.isProcessed,
				Title:       optional.New("title"),
				Body:        optional.New("body"),
				Type:        s,
				Timestamp:   tc.timestamp,
			})
			var sendCount int
			cs := characterservice.NewFake(characterservice.Params{
				SendDesktopNotification: func(title string, content string) {
					sendCount++
				},
				Storage: st,
			})
			// when
			err := cs.NotifyNotifications(t.Context(), n.CharacterID, earliest, typesEnabled)
			// then
			if assert.NoError(t, err) {
				xassert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

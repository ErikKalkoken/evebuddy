package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/stretchr/testify/assert"
)

func TestCharacterNotification(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		timestamp := time.Now().UTC()
		sender := factory.CreateEveEntityCharacter()
		arg := storage.CreateCharacterNotificationParams{
			CharacterID:    c.ID,
			IsRead:         true,
			NotificationID: 42,
			SenderID:       sender.ID,
			Text:           "text",
			Timestamp:      timestamp,
			Type:           "type",
		}
		// when
		err := r.CreateCharacterNotification(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := r.GetCharacterNotification(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, c.ID, o.CharacterID)
				assert.True(t, o.IsRead)
				assert.Equal(t, int64(42), o.NotificationID)
				assert.Equal(t, sender, o.Sender)
				assert.Equal(t, "text", o.Text)
				assert.Equal(t, timestamp.UTC(), o.Timestamp.UTC())
				assert.Equal(t, "type", o.Type)
			}
		}
	})
	t.Run("can list IDs of existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		e1 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		e2 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		e3 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		// when
		ids, err := r.ListCharacterNotificationIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.NewFromSlice(ids)
			want := set.NewFromSlice([]int64{e1.NotificationID, e2.NotificationID, e3.NotificationID})
			assert.Equal(t, want, got)
		}
	})
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID, Type: "bravo"})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID, Type: "alpha"})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID, Type: "alpha"})
		// when
		ee, err := r.ListCharacterNotificationsTypes(ctx, c.ID, []string{"alpha"})
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ee, 2)
		}
	})
	t.Run("can updates IsRead 1", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		n := factory.CreateCharacterNotification()
		// when
		err := r.UpdateCharacterNotification(ctx, storage.UpdateCharacterNotificationParams{ID: n.ID, IsRead: true})
		// then
		if assert.NoError(t, err) {
			o, err := r.GetCharacterNotification(ctx, n.CharacterID, n.ID)
			if assert.NoError(t, err) {
				assert.True(t, o.IsRead)
			}
		}
	})
	t.Run("can updates IsRead 2", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		n := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{IsRead: true})
		// when
		err := r.UpdateCharacterNotification(ctx, storage.UpdateCharacterNotificationParams{ID: n.ID, IsRead: false})
		// then
		if assert.NoError(t, err) {
			o, err := r.GetCharacterNotification(ctx, n.CharacterID, n.ID)
			if assert.NoError(t, err) {
				assert.False(t, o.IsRead)
			}
		}
	})
	t.Run("can update title", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		n := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{})
		// when
		err := r.UpdateCharacterNotification(ctx, storage.UpdateCharacterNotificationParams{ID: n.ID, Title: optional.New("title")})
		// then
		if assert.NoError(t, err) {
			o, err := r.GetCharacterNotification(ctx, n.CharacterID, n.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, "title", o.Title.ValueOrZero())
			}
		}
	})
	t.Run("can update body", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		n := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{})
		// when
		err := r.UpdateCharacterNotification(ctx, storage.UpdateCharacterNotificationParams{ID: n.ID, Body: optional.New("body")})
		// then
		if assert.NoError(t, err) {
			o, err := r.GetCharacterNotification(ctx, n.CharacterID, n.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, "body", o.Body.ValueOrZero())
			}
		}
	})
	t.Run("can calculate unread counts", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID, Type: "bravo"})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID, Type: "alpha"})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID, Type: "alpha", IsRead: true})
		factory.CreateCharacterNotification()
		// when
		x, err := r.CalcCharacterNotificationUnreadCounts(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := map[string]int{
				"alpha": 1,
				"bravo": 1,
			}
			assert.Equal(t, want, x)
		}
	})
	t.Run("can list unread notifs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID, Type: "bravo"})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID, Type: "alpha"})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID, Type: "alpha", IsRead: true})
		// when
		ee, err := r.ListCharacterNotificationsUnread(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ee, 2)
		}
	})
}

func TestNotificationType(t *testing.T) {
	db, st, _ := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		x, err := st.GetOrCreateNotificationType(ctx, "alpha")
		// then
		if assert.NoError(t, err) {
			assert.NotEqual(t, 0, x)
		}

	})
	t.Run("can get existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x1, err := st.GetOrCreateNotificationType(ctx, "alpha")
		if err != nil {
			t.Fatal(err)
		}
		// when
		x2, err := st.GetOrCreateNotificationType(ctx, "alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
}
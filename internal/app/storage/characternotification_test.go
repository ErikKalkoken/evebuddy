package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCharacterNotification(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
			Type:           "StructureDestroyed",
		}
		// when
		err := st.CreateCharacterNotification(ctx, arg)
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		o, err := st.GetCharacterNotification(ctx, c.ID, 42)
		if assert.NoError(t, err) {
			assert.Equal(t, c.ID, o.CharacterID)
			assert.True(t, o.IsRead)
			assert.Equal(t, int64(42), o.NotificationID)
			assert.Equal(t, sender, o.Sender)
			assert.Equal(t, "text", o.Text)
			assert.Equal(t, timestamp.UTC(), o.Timestamp.UTC())
			assert.Equal(t, app.StructureDestroyed, o.Type)
			assert.Nil(t, o.Recipient)
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		timestamp := time.Now().UTC()
		sender := factory.CreateEveEntityCharacter()
		recipient := factory.CreateEveEntityAlliance()
		arg := storage.CreateCharacterNotificationParams{
			Body:           optional.New("body"),
			CharacterID:    c.ID,
			IsRead:         true,
			NotificationID: 42,
			RecipientID:    optional.New(recipient.ID),
			SenderID:       sender.ID,
			Text:           "text",
			Timestamp:      timestamp,
			Title:          optional.New("title"),
			Type:           "StructureDestroyed",
		}
		// when
		err := st.CreateCharacterNotification(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterNotification(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, c.ID, o.CharacterID)
				assert.True(t, o.IsRead)
				assert.Equal(t, int64(42), o.NotificationID)
				assert.Equal(t, sender, o.Sender)
				assert.Equal(t, "text", o.Text)
				assert.Equal(t, timestamp.UTC(), o.Timestamp.UTC())
				assert.Equal(t, app.StructureDestroyed, o.Type)
				assert.Equal(t, "body", o.Body.ValueOrZero())
				assert.Equal(t, "title", o.Title.ValueOrZero())
				assert.Equal(t, recipient, o.Recipient)
			}
		}
	})
	t.Run("should map unknown notif types", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
			Type:           "Invalid",
		}
		// when
		err := st.CreateCharacterNotification(ctx, arg)
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		o, err := st.GetCharacterNotification(ctx, c.ID, 42)
		if assert.NoError(t, err) {
			assert.Equal(t, app.UnknownNotification, o.Type)
		}
	})
	t.Run("can updates IsRead 1", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		n := factory.CreateCharacterNotification()
		// when
		err := st.UpdateCharacterNotification(ctx, storage.UpdateCharacterNotificationParams{
			ID:     n.ID,
			IsRead: true,
		})
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		o, err := st.GetCharacterNotification(ctx, n.CharacterID, n.ID)
		if assert.NoError(t, err) {
			assert.True(t, o.IsRead)
		}
	})
	t.Run("can updates IsRead 2", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		n := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{IsRead: true})
		// when
		err := st.UpdateCharacterNotification(ctx, storage.UpdateCharacterNotificationParams{
			ID:     n.ID,
			IsRead: false,
		})
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		o, err := st.GetCharacterNotification(ctx, n.CharacterID, n.ID)
		if assert.NoError(t, err) {
			assert.False(t, o.IsRead)
		}
	})
	t.Run("can update title", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		n := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{})
		// when
		err := st.UpdateCharacterNotification(ctx, storage.UpdateCharacterNotificationParams{
			ID:    n.ID,
			Title: optional.New("title"),
		})
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		o, err := st.GetCharacterNotification(ctx, n.CharacterID, n.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, "title", o.Title.ValueOrZero())
		}
	})
	t.Run("can update body", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		n := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{})
		// when
		err := st.UpdateCharacterNotification(ctx, storage.UpdateCharacterNotificationParams{
			ID:   n.ID,
			Body: optional.New("body"),
		})
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		o, err := st.GetCharacterNotification(ctx, n.CharacterID, n.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, "body", o.Body.ValueOrZero())
		}
	})
	t.Run("can mark notifs as processed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacter()
		n1 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c1.ID,
			Body:        optional.New("Body"),
			Title:       optional.New("Title"),
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID:    c1.ID,
			NotificationID: 42,
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			NotificationID: 42,
		})
		// when
		err := st.UpdateCharacterNotificationsSetProcessed(ctx, 42)
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		ee, err := st.ListCharacterNotificationsUnprocessed(ctx, c1.ID, time.Now().Add(-24*time.Hour))
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
	})
	t.Run("can calculate counts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureDestroyed",
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureUnderAttack",
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureUnderAttack",
			IsRead:      true,
		})
		factory.CreateCharacterNotification()
		// when
		x, err := st.CountCharacterNotifications(ctx, c.ID)
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		want := map[app.EveNotificationType][]int{
			app.StructureUnderAttack: {2, 1},
			app.StructureDestroyed:   {1, 1},
		}
		assert.Equal(t, want, x)
	})
}

func TestCharacterNotification_List(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can list IDs of existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		e1 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		e2 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		e3 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		// when
		got, err := st.ListCharacterNotificationIDs(ctx, c.ID)
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		want := set.Of(e1.NotificationID, e2.NotificationID, e3.NotificationID)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
	})
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureUnderAttack",
		})
		n1 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureDestroyed",
		})
		n2 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureDestroyed",
		})
		// when
		ee, err := st.ListCharacterNotificationsForTypes(ctx, c.ID, set.Of(app.StructureDestroyed))
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		want := set.Of(n1.NotificationID, n2.NotificationID)
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.NotificationID
		}))
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
	})
	t.Run("can list unread notifs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		n1 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "bravo",
		})
		n2 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "alpha",
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "alpha",
			IsRead:      true,
		})
		// when
		ee, err := st.ListCharacterNotificationsUnread(ctx, c.ID)
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID, n2.ID)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
	})
}

func TestCharacterNotification_ListUnprocessed(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should not return already processed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		now := time.Now().UTC()
		n1 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now,
			Title:       optional.New("title"),
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			IsProcessed: true,
			Type:        "bravo",
			Timestamp:   now,
			Title:       optional.New("title"),
		})
		// when
		ee, err := st.ListCharacterNotificationsUnprocessed(ctx, c.ID, now.Add(-24*time.Hour))
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
	})
	t.Run("should not return stale notifs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		now := time.Now().UTC()
		n1 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now,
			Title:       optional.New("title"),
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now.Add(-25 * time.Hour),
			Title:       optional.New("title"),
		})
		// when
		ee, err := st.ListCharacterNotificationsUnprocessed(ctx, c.ID, now.Add(-24*time.Hour))
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
	})
	t.Run("should not return notifs which have no title or body", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		now := time.Now().UTC()
		n1 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now,
			Title:       optional.New("title"),
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now,
			Title:       optional.New("title"),
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now,
		})
		// when
		ee, err := st.ListCharacterNotificationsUnprocessed(ctx, c.ID, now.Add(-24*time.Hour))
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
	})
	t.Run("should not return duplicates of processed notifs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		now := time.Now().UTC()
		n1 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now,
			Title:       optional.New("title"),
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:           optional.New("body"),
			CharacterID:    c.ID,
			NotificationID: 42,
			Type:           "bravo",
			Timestamp:      now,
			Title:          optional.New("title"),
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:           optional.New("body"),
			NotificationID: 42,
			IsProcessed:    true,
			Type:           "bravo",
			Timestamp:      now,
			Title:          optional.New("title"),
		})
		// when
		ee, err := st.ListCharacterNotificationsUnprocessed(ctx, c.ID, now.Add(-24*time.Hour))
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
	})
}

func TestNotificationType(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		x, err := st.GetOrCreateNotificationType(ctx, "alpha")
		// then
		if assert.NoError(t, err) {
			assert.NotEqual(t, 0, x)
		}

	})
	t.Run("can get existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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

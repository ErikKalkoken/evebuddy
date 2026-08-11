package storage_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCharacterNotification(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		timestamp := time.Now().UTC()
		sender := f.CreateEveEntityCharacter()
		arg := storage.CreateCharacterNotificationParams{
			CharacterID:    c.ID,
			IsRead:         optional.New(true),
			NotificationID: 42,
			SenderID:       sender.ID,
			Text:           optional.New("text"),
			Timestamp:      timestamp,
			Type:           "StructureDestroyed",
		}
		// when
		err := st.CreateCharacterNotification(t.Context(), arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterNotification(t.Context(), c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, c.ID, o.CharacterID)
		assert.True(t, o.IsRead.ValueOrZero())
		xassert.Equal(t, 42, o.NotificationID)
		xassert.Equal(t, sender, o.Sender)
		xassert.EqualOptional(t, "text", o.Text)
		xassert.Equal(t, timestamp.UTC(), o.Timestamp.UTC())
		xassert.Equal(t, app.StructureDestroyed, o.Type)
		assert.True(t, o.Recipient.IsEmpty())
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		timestamp := time.Now().UTC()
		sender := f.CreateEveEntityCharacter()
		recipient := f.CreateEveEntityAlliance()
		arg := storage.CreateCharacterNotificationParams{
			Body:           optional.New("body"),
			CharacterID:    c.ID,
			IsRead:         optional.New(true),
			NotificationID: 42,
			RecipientID:    optional.New(recipient.ID),
			SenderID:       sender.ID,
			Text:           optional.New("text"),
			Timestamp:      timestamp,
			Title:          optional.New("title"),
			Type:           "StructureDestroyed",
		}
		// when
		err := st.CreateCharacterNotification(t.Context(), arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterNotification(t.Context(), c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, c.ID, o.CharacterID)
		assert.True(t, o.IsRead.ValueOrZero())
		xassert.Equal(t, 42, o.NotificationID)
		xassert.Equal(t, sender, o.Sender)
		xassert.EqualOptional(t, "text", o.Text)
		xassert.Equal(t, timestamp.UTC(), o.Timestamp.UTC())
		xassert.Equal(t, app.StructureDestroyed, o.Type)
		xassert.EqualOptional(t, "body", o.Body)
		xassert.EqualOptional(t, "title", o.Title)
		xassert.EqualOptional(t, recipient, o.Recipient)
	})
	t.Run("should map unknown notif types", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		timestamp := time.Now().UTC()
		sender := f.CreateEveEntityCharacter()
		arg := storage.CreateCharacterNotificationParams{
			CharacterID:    c.ID,
			IsRead:         optional.New(true),
			NotificationID: 42,
			SenderID:       sender.ID,
			Text:           optional.New("text"),
			Timestamp:      timestamp,
			Type:           "Invalid",
		}
		// when
		err := st.CreateCharacterNotification(t.Context(), arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterNotification(t.Context(), c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, app.UnknownNotification, o.Type)
	})
	t.Run("can updates IsRead 1", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		n := f.CreateCharacterNotification()
		// when
		err := st.UpdateCharacterNotification(t.Context(), storage.UpdateCharacterNotificationParams{
			ID:     n.ID,
			IsRead: optional.New(true),
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterNotification(t.Context(), n.CharacterID, n.ID)
		require.NoError(t, err)
		assert.True(t, o.IsRead.ValueOrZero())
	})
	t.Run("can updates IsRead 2", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		n := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			IsRead: optional.New(true),
		})
		// when
		err := st.UpdateCharacterNotification(t.Context(), storage.UpdateCharacterNotificationParams{
			ID: n.ID,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterNotification(t.Context(), n.CharacterID, n.ID)
		require.NoError(t, err)
		assert.False(t, o.IsRead.ValueOrZero())
	})
	t.Run("can update title", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		n := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{})
		// when
		err := st.UpdateCharacterNotification(t.Context(), storage.UpdateCharacterNotificationParams{
			ID:    n.ID,
			Title: optional.New("title"),
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterNotification(t.Context(), n.CharacterID, n.ID)
		require.NoError(t, err)
		xassert.EqualOptional(t, "title", o.Title)
	})
	t.Run("can update body", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		n := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{})
		// when
		err := st.UpdateCharacterNotification(t.Context(), storage.UpdateCharacterNotificationParams{
			ID:   n.ID,
			Body: optional.New("body"),
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterNotification(t.Context(), n.CharacterID, n.ID)
		require.NoError(t, err)
		xassert.EqualOptional(t, "body", o.Body)
	})
	t.Run("can mark notifs as processed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := f.CreateCharacter()
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c1.ID,
			Body:        optional.New("Body"),
			Title:       optional.New("Title"),
		})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID:    c1.ID,
			NotificationID: 42,
		})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			NotificationID: 42,
		})
		// when
		err := st.UpdateCharacterNotificationsSetProcessed(t.Context(), c1.ID, 42)
		// then
		require.NoError(t, err)
		ee, err := st.ListCharacterNotificationsUnprocessed(t.Context(), c1.ID, time.Now().Add(-24*time.Hour))
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID)
		xassert.Equal(t, want, got)
	})

	t.Run("can calculate counts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureDestroyed",
		})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureUnderAttack",
		})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureUnderAttack",
			IsRead:      optional.New(true),
		})
		f.CreateCharacterNotification()
		// when
		x, err := st.CountCharacterNotifications(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		want := map[app.EveNotificationType][]int{
			app.StructureUnderAttack: {2, 1},
			app.StructureDestroyed:   {1, 1},
		}
		xassert.Equal(t, want, x)
	})

	t.Run("can delete notifications", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		e1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		e2 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		e3 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		// when
		err := st.DeleteCharacterNotifications(t.Context(), c.ID, set.Of(e1.NotificationID, e2.NotificationID))
		// then
		require.NoError(t, err)
		got, err := st.ListCharacterNotificationIDs(t.Context(), c.ID)
		require.NoError(t, err)
		want := set.Of(e3.NotificationID)
		xassert.Equal(t, want, got)
	})
}

func TestStorage_UpdateCharacterNotificationsSetIsRead(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	cases := []struct {
		name    string
		initial optional.Optional[bool]
		want    bool
	}{
		{"can set read when unread", optional.New(false), true},
		{"can set unread when read", optional.New(true), false},
		{"can set read when empty", optional.Optional[bool]{}, true},
		{"can set unread when empty", optional.Optional[bool]{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			c := f.CreateCharacter()
			n := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
				CharacterID: c.ID,
				IsRead:      tc.initial,
			})

			// when
			err := st.UpdateCharacterNotificationsSetIsRead(t.Context(), c.ID, n.NotificationID, tc.want)

			// then
			require.NoError(t, err)
			n2, err := st.GetCharacterNotification(t.Context(), c.ID, n.ID)
			require.NoError(t, err)
			xassert.EqualOptional(t, tc.want, n2.IsRead)
		})
	}
}

func TestCharacterNotification_List(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can list IDs of existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		e1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		e2 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		e3 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		// when
		got, err := st.ListCharacterNotificationIDs(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(e1.NotificationID, e2.NotificationID, e3.NotificationID)
		xassert.Equal(t, want, got)
	})
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureUnderAttack",
		})
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureDestroyed",
		})
		n2 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureDestroyed",
		})
		// when
		ee, err := st.ListCharacterNotificationsForTypes(t.Context(), c.ID, set.Of(app.StructureDestroyed))
		// then
		require.NoError(t, err)
		want := set.Of(n1.NotificationID, n2.NotificationID)
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.NotificationID
		}))
		xassert.Equal(t, want, got)
	})
	t.Run("can list unread notifs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "bravo",
		})
		n2 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "alpha",
		})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "alpha",
			IsRead:      optional.New(true),
		})
		// when
		ee, err := st.ListCharacterNotificationsUnread(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID, n2.ID)
		xassert.Equal(t, want, got)
	})
}

func TestCharacterNotification_ListUnprocessed(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should not return already processed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		now := time.Now().UTC()
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now,
			Title:       optional.New("title"),
		})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			IsProcessed: true,
			Type:        "bravo",
			Timestamp:   now,
			Title:       optional.New("title"),
		})
		// when
		ee, err := st.ListCharacterNotificationsUnprocessed(t.Context(), c.ID, now.Add(-24*time.Hour))
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("should not return stale notifs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		now := time.Now().UTC()
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now,
			Title:       optional.New("title"),
		})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now.Add(-25 * time.Hour),
			Title:       optional.New("title"),
		})
		// when
		ee, err := st.ListCharacterNotificationsUnprocessed(t.Context(), c.ID, now.Add(-24*time.Hour))
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("should not return notifs which have no title or body", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		now := time.Now().UTC()
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now,
			Title:       optional.New("title"),
		})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now,
			Title:       optional.New("title"),
		})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now,
		})
		// when
		ee, err := st.ListCharacterNotificationsUnprocessed(t.Context(), c.ID, now.Add(-24*time.Hour))
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("should not return duplicates of processed notifs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		now := time.Now().UTC()
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   now,
			Title:       optional.New("title"),
		})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:           optional.New("body"),
			CharacterID:    c.ID,
			NotificationID: 42,
			Type:           "bravo",
			Timestamp:      now,
			Title:          optional.New("title"),
		})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:           optional.New("body"),
			NotificationID: 42,
			IsProcessed:    true,
			Type:           "bravo",
			Timestamp:      now,
			Title:          optional.New("title"),
		})
		// when
		ee, err := st.ListCharacterNotificationsUnprocessed(t.Context(), c.ID, now.Add(-24*time.Hour))
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID)
		xassert.Equal(t, want, got)
	})
}

func TestNotificationType(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		x, err := st.GetOrCreateNotificationType(t.Context(), "alpha")
		// then
		require.NoError(t, err)
		assert.NotEqual(t, 0, x)
	})
	t.Run("can get existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x1, err := st.GetOrCreateNotificationType(t.Context(), "alpha")
		if err != nil {
			t.Fatal(err)
		}
		// when
		x2, err := st.GetOrCreateNotificationType(t.Context(), "alpha")
		// then
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
}

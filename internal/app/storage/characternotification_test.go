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
			IsRead:         true,
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
		assert.True(t, o.IsRead)
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
			IsRead:         true,
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
		assert.True(t, o.IsRead)
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
			IsRead:         true,
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

	t.Run("returns not found error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetCharacterNotification(t.Context(), 1, 2)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestStorage_UpdateCharacterNotificationsSetIsRead(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	cases := []struct {
		name    string
		initial bool
		want    bool
	}{
		{"can set read when unread", false, true},
		{"can set unread when read", true, false},
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
			err := st.UpdateCharacterNotificationsSetIsRead(t.Context(), set.Of(n.ID), tc.want)

			// then
			require.NoError(t, err)
			n2, err := st.GetCharacterNotification(t.Context(), c.ID, n.ID)
			require.NoError(t, err)
			xassert.Equal(t, tc.want, n2.IsRead)
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

	t.Run("can list notifications for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
		})
		n2 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
		})
		f.CreateCharacterNotification()

		// when
		ee, err := st.ListCharacterNotifications(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		want := set.Of(n1.NotificationID, n2.NotificationID)
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.NotificationID
		}))
		xassert.Equal(t, want, got)
	})

	t.Run("can list notifications for all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		n1 := f.CreateCharacterNotification()
		n2 := f.CreateCharacterNotification()

		// when
		ee, err := st.ListAllCharacterNotifications(t.Context())

		// then
		require.NoError(t, err)
		want := set.Of(n1.NotificationID, n2.NotificationID)
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.NotificationID
		}))
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
	t.Run("should compare with earliest correctly when notification was created in other time zone", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   time.Date(2026, 11, 2, 10, 0, 0, 0, time.FixedZone("UTC+2", 2*3600)), // 08:00 UTC
			Title:       optional.New("title"),
		})
		n2 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC),
			Title:       optional.New("title"),
		})
		earliest := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
		// when
		ee, err := st.ListCharacterNotificationsUnprocessed(t.Context(), c.ID, earliest)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n2.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("should compare with earliest correctly when earliest is not in UTC", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC),
			Title:       optional.New("title"),
		})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			Type:        "bravo",
			Timestamp:   time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC),
			Title:       optional.New("title"),
		})
		earliest := time.Date(2026, 11, 2, 10, 30, 0, 0, time.FixedZone("UTC+1", 3600)) // 09:30 UTC
		// when
		ee, err := st.ListCharacterNotificationsUnprocessed(t.Context(), c.ID, earliest)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(ee, func(x *app.CharacterNotification) int64 {
			return x.ID
		}))
		want := set.Of(n1.ID)
		xassert.Equal(t, want, got)
	})
}

func TestStorage_CreateCharacterNotifications(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should create all notifications", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		sender := f.CreateEveEntityCharacter()
		recipient := f.CreateEveEntityCorporation()
		timestamp := time.Now().UTC()
		args := []storage.CreateCharacterNotificationParams{
			{
				Body:           optional.New("body"),
				CharacterID:    c.ID,
				IsProcessed:    true,
				IsRead:         true,
				NotificationID: 1,
				RecipientID:    optional.New(recipient.ID),
				SenderID:       sender.ID,
				Text:           optional.New("text"),
				Timestamp:      timestamp,
				Title:          optional.New("title"),
				Type:           "StructureDestroyed",
			},
			{
				CharacterID:    c.ID,
				NotificationID: 2,
				SenderID:       sender.ID,
				Timestamp:      timestamp,
				Type:           "BountyClaimMsg",
			},
		}
		// when
		err := st.CreateCharacterNotifications(t.Context(), args)
		// then
		require.NoError(t, err)
		o1, err := st.GetCharacterNotification(t.Context(), c.ID, 1)
		require.NoError(t, err)
		xassert.EqualOptional(t, "body", o1.Body)
		xassert.Equal(t, c.ID, o1.CharacterID)
		assert.True(t, o1.IsProcessed)
		assert.True(t, o1.IsRead)
		xassert.Equal(t, recipient, o1.Recipient.MustValue())
		xassert.Equal(t, sender, o1.Sender)
		xassert.EqualOptional(t, "text", o1.Text)
		xassert.Equal(t, timestamp, o1.Timestamp.UTC())
		xassert.EqualOptional(t, "title", o1.Title)
		xassert.Equal(t, app.StructureDestroyed, o1.Type)
		o2, err := st.GetCharacterNotification(t.Context(), c.ID, 2)
		require.NoError(t, err)
		assert.False(t, o2.IsRead)
		assert.True(t, o2.Title.IsEmpty())
		xassert.Equal(t, app.BountyClaimMsg, o2.Type)
	})
	t.Run("should do nothing when called without args", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := st.CreateCharacterNotifications(t.Context(), nil)
		// then
		require.NoError(t, err)
	})
	t.Run("should create none when an arg is invalid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		sender := f.CreateEveEntityCharacter()
		args := []storage.CreateCharacterNotificationParams{
			{
				CharacterID:    c.ID,
				NotificationID: 1,
				SenderID:       sender.ID,
				Type:           "StructureDestroyed",
			},
			{
				CharacterID:    c.ID,
				NotificationID: 2,
				Type:           "StructureDestroyed",
			},
		}
		// when
		err := st.CreateCharacterNotifications(t.Context(), args)
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
		ids, err := st.ListCharacterNotificationIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, ids.Size())
	})
	t.Run("should create none when storing an arg fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		sender := f.CreateEveEntityCharacter()
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID:    c.ID,
			NotificationID: 2,
		})
		args := []storage.CreateCharacterNotificationParams{
			{
				CharacterID:    c.ID,
				NotificationID: 1,
				SenderID:       sender.ID,
				Type:           "StructureDestroyed",
			},
			{
				CharacterID:    c.ID,
				NotificationID: 2,
				SenderID:       sender.ID,
				Type:           "StructureDestroyed",
			},
		}
		// when
		err := st.CreateCharacterNotifications(t.Context(), args)
		// then
		assert.Error(t, err)
		ids, err := st.ListCharacterNotificationIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](2), ids)
	})
	t.Run("should return error when type is empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		sender := f.CreateEveEntityCharacter()
		args := []storage.CreateCharacterNotificationParams{{
			CharacterID:    c.ID,
			NotificationID: 1,
			SenderID:       sender.ID,
		}}
		// when
		err := st.CreateCharacterNotifications(t.Context(), args)
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("should create missing types and keep existing ones", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		sender := f.CreateEveEntityCharacter()
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID:    c.ID,
			NotificationID: 3,
			Type:           "StructureDestroyed",
		})
		args := []storage.CreateCharacterNotificationParams{
			{
				CharacterID:    c.ID,
				NotificationID: 1,
				SenderID:       sender.ID,
				Type:           "StructureDestroyed",
			},
			{
				CharacterID:    c.ID,
				NotificationID: 2,
				SenderID:       sender.ID,
				Type:           "BountyClaimMsg",
			},
		}
		// when
		err := st.CreateCharacterNotifications(t.Context(), args)
		// then
		require.NoError(t, err)
		o0, err := st.GetCharacterNotification(t.Context(), c.ID, 3)
		require.NoError(t, err)
		xassert.Equal(t, app.StructureDestroyed, o0.Type)
		o1, err := st.GetCharacterNotification(t.Context(), c.ID, 1)
		require.NoError(t, err)
		xassert.Equal(t, app.StructureDestroyed, o1.Type)
		o2, err := st.GetCharacterNotification(t.Context(), c.ID, 2)
		require.NoError(t, err)
		xassert.Equal(t, app.BountyClaimMsg, o2.Type)
	})
}

func TestStorage_SyncCharacterNotifications(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should apply deletes, updates and creates", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		sender := f.CreateEveEntityCharacter()
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID:    c.ID,
			NotificationID: 1,
		})
		n2 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID:    c.ID,
			NotificationID: 2,
		})
		arg := storage.SyncCharacterNotificationsParams{
			CharacterID: c.ID,
			Create: []storage.CreateCharacterNotificationParams{{
				CharacterID:    c.ID,
				NotificationID: 3,
				SenderID:       sender.ID,
				Type:           "BountyClaimMsg",
			}},
			Delete: set.Of(n1.NotificationID),
			Update: []storage.UpdateCharacterNotificationParams{{
				ID:    n2.ID,
				Title: optional.New("title"),
			}},
		}
		// when
		err := st.SyncCharacterNotifications(t.Context(), arg)
		// then
		require.NoError(t, err)
		ids, err := st.ListCharacterNotificationIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](2, 3), ids)
		o2, err := st.GetCharacterNotification(t.Context(), c.ID, 2)
		require.NoError(t, err)
		xassert.EqualOptional(t, "title", o2.Title)
		o3, err := st.GetCharacterNotification(t.Context(), c.ID, 3)
		require.NoError(t, err)
		xassert.Equal(t, app.BountyClaimMsg, o3.Type)
	})
	t.Run("should delete only notifications of the character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		n2 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		other := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			NotificationID: n1.NotificationID,
		})
		// when
		err := st.SyncCharacterNotifications(t.Context(), storage.SyncCharacterNotificationsParams{
			CharacterID: c.ID,
			Delete:      set.Of(n1.NotificationID),
		})
		// then
		require.NoError(t, err)
		ids, err := st.ListCharacterNotificationIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(n2.NotificationID), ids)
		otherIDs, err := st.ListCharacterNotificationIDs(t.Context(), other.CharacterID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(other.NotificationID), otherIDs)
	})
	t.Run("should update only notifications of the character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		other := f.CreateCharacterNotification()
		// when
		err := st.SyncCharacterNotifications(t.Context(), storage.SyncCharacterNotificationsParams{
			CharacterID: c.ID,
			Update: []storage.UpdateCharacterNotificationParams{{
				ID:    other.ID,
				Title: optional.New("title"),
			}},
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterNotification(t.Context(), other.CharacterID, other.NotificationID)
		require.NoError(t, err)
		xassert.Equal(t, other.Title, o.Title)
	})
	t.Run("should update all notifications", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		n1 := f.CreateCharacterNotification()
		n2 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: n1.CharacterID,
			IsRead:      true,
		})
		arg := storage.SyncCharacterNotificationsParams{
			CharacterID: n1.CharacterID,
			Update: []storage.UpdateCharacterNotificationParams{
				{
					Body:   optional.New("body"),
					ID:     n1.ID,
					IsRead: optional.New(true),
					Title:  optional.New("title"),
				},
				{
					ID:     n2.ID,
					IsRead: optional.New(false),
				},
			},
		}
		// when
		err := st.SyncCharacterNotifications(t.Context(), arg)
		// then
		require.NoError(t, err)
		o1, err := st.GetCharacterNotification(t.Context(), n1.CharacterID, n1.NotificationID)
		require.NoError(t, err)
		xassert.EqualOptional(t, "body", o1.Body)
		assert.True(t, o1.IsRead)
		xassert.EqualOptional(t, "title", o1.Title)
		o2, err := st.GetCharacterNotification(t.Context(), n2.CharacterID, n2.NotificationID)
		require.NoError(t, err)
		assert.True(t, o2.Body.IsEmpty())
		assert.False(t, o2.IsRead)
		assert.True(t, o2.Title.IsEmpty())
	})
	t.Run("should keep is read when not set", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		n := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			IsRead: true,
		})
		arg := storage.SyncCharacterNotificationsParams{
			CharacterID: n.CharacterID,
			Update: []storage.UpdateCharacterNotificationParams{{
				ID:    n.ID,
				Title: optional.New("title"),
			}},
		}
		// when
		err := st.SyncCharacterNotifications(t.Context(), arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterNotification(t.Context(), n.CharacterID, n.NotificationID)
		require.NoError(t, err)
		assert.True(t, o.IsRead)
		xassert.EqualOptional(t, "title", o.Title)
	})
	t.Run("should do nothing when called without changes", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		// when
		err := st.SyncCharacterNotifications(t.Context(), storage.SyncCharacterNotificationsParams{
			CharacterID: c.ID,
		})
		// then
		require.NoError(t, err)
	})
	t.Run("should return error when character ID is missing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := st.SyncCharacterNotifications(t.Context(), storage.SyncCharacterNotificationsParams{})
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("should apply none when an arg is invalid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		n := f.CreateCharacterNotification()
		arg := storage.SyncCharacterNotificationsParams{
			CharacterID: n.CharacterID,
			Delete:      set.Of(n.NotificationID),
			Update:      []storage.UpdateCharacterNotificationParams{{}},
		}
		// when
		err := st.SyncCharacterNotifications(t.Context(), arg)
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
		ids, err := st.ListCharacterNotificationIDs(t.Context(), n.CharacterID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(n.NotificationID), ids)
	})
	t.Run("should apply none when a create has another character ID", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		other := f.CreateCharacter()
		sender := f.CreateEveEntityCharacter()
		arg := storage.SyncCharacterNotificationsParams{
			CharacterID: c.ID,
			Create: []storage.CreateCharacterNotificationParams{{
				CharacterID:    other.ID,
				NotificationID: 1,
				SenderID:       sender.ID,
				Type:           "BountyClaimMsg",
			}},
		}
		// when
		err := st.SyncCharacterNotifications(t.Context(), arg)
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
		ids, err := st.ListCharacterNotificationIDs(t.Context(), other.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, ids.Size())
	})
	t.Run("should apply none when storing fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		sender := f.CreateEveEntityCharacter()
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID:    c.ID,
			NotificationID: 1,
		})
		n2 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID:    c.ID,
			NotificationID: 2,
			Title:          optional.New("title"),
		})
		arg := storage.SyncCharacterNotificationsParams{
			CharacterID: c.ID,
			Create: []storage.CreateCharacterNotificationParams{{
				CharacterID:    c.ID,
				NotificationID: n2.NotificationID, // already stored
				SenderID:       sender.ID,
				Type:           "BountyClaimMsg",
			}},
			Delete: set.Of(n1.NotificationID),
			Update: []storage.UpdateCharacterNotificationParams{{
				ID:    n2.ID,
				Title: optional.New("title2"),
			}},
		}
		// when
		err := st.SyncCharacterNotifications(t.Context(), arg)
		// then
		assert.Error(t, err)
		ids, err := st.ListCharacterNotificationIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(n1.NotificationID, n2.NotificationID), ids)
		o2, err := st.GetCharacterNotification(t.Context(), c.ID, n2.NotificationID)
		require.NoError(t, err)
		xassert.EqualOptional(t, "title", o2.Title)
	})
}

func TestStorage_ListCharacterNotificationStates(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should return states of the character's notifications", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			Body:        optional.New("body"),
			CharacterID: c.ID,
			IsRead:      true,
			Title:       optional.New("title"),
			Type:        "StructureDestroyed",
		})
		n2 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "alpha",
		})
		f.CreateCharacterNotification() // other character
		// when
		got, err := st.ListCharacterNotificationStates(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		want := []storage.CharacterNotificationState{
			{
				Body:           optional.New("body"),
				ID:             n1.ID,
				IsRead:         true,
				NotificationID: n1.NotificationID,
				Title:          optional.New("title"),
				Type:           app.StructureDestroyed,
			},
			{
				ID:             n2.ID,
				NotificationID: n2.NotificationID,
				Type:           app.UnknownNotification,
			},
		}
		assert.ElementsMatch(t, want, got)
	})
}

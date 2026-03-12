package characterservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateCharacterNotificationsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("should create new notification from scratch 1", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(*c.EveCharacter.EveEntity())
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		sender := factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_read":         true,
				"notification_id": 42,
				"sender_id":       sender.ID,
				"sender_type":     "corporation",
				"text":            "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n",
				"timestamp":       "2017-08-16T10:08:00Z",
				"type":            "InsurancePayoutMsg",
			}}),
		)
		// when
		changed, err := s.updateNotificationsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterNotifications,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterNotification(ctx, c.ID, 42)
		require.NoError(t, err)
		assert.True(t, o.IsRead.ValueOrZero())
		xassert.Equal(t, int64(42), o.NotificationID)
		xassert.Equal(t, sender, o.Sender)
		xassert.Equal(t, app.InsurancePayoutMsg, o.Type)
		xassert.Equal(t, "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n", o.Text.ValueOrZero())
		xassert.Equal(t, time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC), o.Timestamp)
		xassert.Equal(t, c.ID, o.Recipient.ValueOrZero().ID)
		ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](42), ids)
	})
	// t.Run("should create new notification from scratch 2", func(t *testing.T) {
	// 	// given
	// 	testutil.TruncateTables(db)
	// 	httpmock.Reset()
	// 	c := factory.CreateCharacter()
	// 	factory.CreateEveEntityCharacter(*c.EveCharacter.EveEntity())
	// 	factory.CreateEveEntityCharacter(app.EveEntity{ID: 1234})
	// 	factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
	// 	sender := factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
	// 	httpmock.RegisterResponder(
	// 		"GET",
	// 		fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
	// 		httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
	// 			"is_read":         true,
	// 			"notification_id": 42,
	// 			"sender_id":       sender.ID,
	// 			"sender_type":     "corporation",
	// 			"text":            "applicationText: example\ncharID: 1234\ncorpID: 54321\n",
	// 			"timestamp":       "2017-08-16T10:08:00Z",
	// 			"type":            "CharAppAcceptMsg",
	// 		}}),
	// 	)
	// 	// when
	// 	changed, err := s.updateNotificationsESI(ctx, CharacterSectionUpdateParams{
	// 		CharacterID: c.ID,
	// 		Section:     app.SectionCharacterNotifications,
	// 	})
	// 	// then
	// 	require.NoError(t, err)
	// 		assert.True(t, changed)
	// 		o, err := st.GetCharacterNotification(ctx, c.ID, 42)
	// 		require.NoError(t, err)
	// 			assert.True(t, o.IsRead)
	// 			 xassert.Equal(t, int64(42), o.NotificationID)
	// 			 xassert.Equal(t, sender, o.Sender)
	// 			 xassert.Equal(t, app.CharAppAcceptMsg, o.Type)
	// 			 xassert.Equal(t, time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC), o.Timestamp)
	// 			 xassert.Equal(t, sender.ID, o.Recipient.ID)
	// 		}
	// 		ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
	// 		require.NoError(t, err)
	// 			 xassert.Equal(t, 1, ids.Size())
	// 		}
	// 	}
	// })
	t.Run("should add new notification", func(t *testing.T) {
		// given
		const (
			notificationID = 42
			text           = "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n"
		)
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(*c.EveCharacter.EveEntity())
		n1 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		sender := factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_read":         true,
				"notification_id": notificationID,
				"sender_id":       sender.ID,
				"sender_type":     "corporation",
				"text":            text,
				"timestamp":       "2017-08-16T10:08:00Z",
				"type":            "InsurancePayoutMsg",
			}}))
		// when
		changed, err := s.updateNotificationsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterNotifications,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterNotification(ctx, c.ID, 42)
		require.NoError(t, err)
		assert.True(t, o.IsRead.ValueOrZero())
		xassert.Equal(t, 42, o.NotificationID)
		xassert.Equal(t, sender, o.Sender)
		xassert.Equal(t, app.InsurancePayoutMsg, o.Type)
		xassert.Equal(t, text, o.Text.ValueOrZero())
		xassert.Equal(t, time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC), o.Timestamp)
		ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(42, n1.NotificationID), ids)
	})
	t.Run("should update isRead for existing notification", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(*c.EveCharacter.EveEntity())
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID:    c.ID,
			NotificationID: 42,
		})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		sender := factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
		data := []map[string]any{{
			"is_read":         true,
			"notification_id": 42,
			"sender_id":       sender.ID,
			"sender_type":     "corporation",
			"text":            "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n",
			"timestamp":       "2017-08-16T10:08:00Z",
			"type":            "InsurancePayoutMsg",
		}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateNotificationsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterNotifications,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterNotification(ctx, c.ID, 42)
		require.NoError(t, err)
		assert.True(t, o.IsRead.ValueOrZero())
		ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 1, ids.Size())
	})

	t.Run("should resolve a complex notification", func(t *testing.T) {
		// given
		const (
			notificationID = 1000000201
		)
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(*c.EveCharacter.EveEntity())
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		recruit := factory.CreateEveEntityCharacter()
		corporation := c.EveCharacter.Corporation
		text := fmt.Sprintf("charID: %d\ncorpID: %d\n", recruit.ID, corporation.ID)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_read":         true,
				"notification_id": notificationID,
				"sender_id":       corporation.ID,
				"sender_type":     "corporation",
				"text":            text,
				"timestamp":       "2017-08-16T10:08:00Z",
				"type":            "CharLeftCorpMsg",
			}}),
		)
		// when
		_, err := s.updateNotificationsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterNotifications,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterNotification(ctx, c.ID, notificationID)
		require.NoError(t, err)
		assert.True(t, o.IsRead.ValueOrZero())
		xassert.Equal(t, notificationID, o.NotificationID)
		xassert.Equal(t, corporation, o.Sender)
		xassert.Equal(t, app.CharLeftCorpMsg, o.Type)
		xassert.Equal(t, text, o.Text.ValueOrZero())
		xassert.Equal(t, time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC), o.Timestamp)
		xassert.Equal(t, corporation, o.Recipient.ValueOrZero())
	})

	t.Run("should abort when sender can not be resolved", func(t *testing.T) {
		// given
		const (
			notificationID  = 1000000201
			text            = "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n"
			invalidSenderID = 666
		)
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(*c.EveCharacter.EveEntity())
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_read":         true,
				"notification_id": notificationID,
				"sender_id":       invalidSenderID,
				"sender_type":     "corporation",
				"text":            text,
				"timestamp":       "2017-08-16T10:08:00Z",
				"type":            "InsurancePayoutMsg",
			}}),
		)
		httpmock.RegisterResponder("POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewErrorResponder(fmt.Errorf("failed")),
		)
		// when
		_, err := s.updateNotificationsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterNotifications,
		})
		// then
		assert.Error(t, err)
		ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](), ids)
	})

	t.Run("should not abort when entities inside notification can not be resolved", func(t *testing.T) {
		// given
		const (
			notificationID = 1000000201
			recruitID      = 666
		)
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(*c.EveCharacter.EveEntity())
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		corporation := c.EveCharacter.Corporation
		text := fmt.Sprintf("charID: %d\ncorpID: %d\n", recruitID, corporation.ID)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_read":         true,
				"notification_id": notificationID,
				"sender_id":       corporation.ID,
				"sender_type":     "corporation",
				"text":            text,
				"timestamp":       "2017-08-16T10:08:00Z",
				"type":            "CharLeftCorpMsg",
			}}),
		)
		httpmock.RegisterResponder("POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewErrorResponder(fmt.Errorf("failed")),
		)
		// when
		_, err := s.updateNotificationsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterNotifications,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterNotification(ctx, c.ID, notificationID)
		require.NoError(t, err)
		assert.True(t, o.IsRead.ValueOrZero())
		xassert.Equal(t, notificationID, o.NotificationID)
		xassert.Equal(t, corporation, o.Sender)
		xassert.Equal(t, app.CharLeftCorpMsg, o.Type)
		xassert.Equal(t, text, o.Text.ValueOrZero())
		xassert.Equal(t, time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC), o.Timestamp)
		xassert.Equal(t, corporation, o.Recipient.ValueOrZero())
	})
}

func TestListCharacterNotifications(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureDestroyed",
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "StructureDestroyed",
		})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: c.ID,
			Type:        "alpha",
		})
		// when
		tt, err := s.ListNotificationsForGroup(ctx, c.ID, app.GroupStructure)
		// then
		require.NoError(t, err)
		assert.Len(t, tt, 2)
	})
}

package character

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestUpdateCharacterNotificationsESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newCharacterService(st)
	ctx := context.Background()
	t.Run("should create new notification from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		sender := factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
		data := []map[string]any{{
			"is_read":         true,
			"notification_id": 42,
			"sender_id":       sender.ID,
			"sender_type":     "corporation",
			"text":            "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n",
			"timestamp":       "2017-08-16T10:08:00Z",
			"type":            "InsurancePayoutMsg"}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/notifications/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateCharacterNotificationsESI(ctx, UpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionNotifications,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterNotification(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, o.IsRead)
				assert.Equal(t, int64(42), o.NotificationID)
				assert.Equal(t, sender, o.Sender)
				assert.Equal(t, "InsurancePayoutMsg", o.Type)
				assert.Equal(t, "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n", o.Text)
				assert.Equal(t, time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC), o.Timestamp)
			}
			ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids, 1)
			}
		}
	})
	t.Run("should add new notification", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		sender := factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
		data := []map[string]any{{
			"is_read":         true,
			"notification_id": 42,
			"sender_id":       sender.ID,
			"sender_type":     "corporation",
			"text":            "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n",
			"timestamp":       "2017-08-16T10:08:00Z",
			"type":            "InsurancePayoutMsg"}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/notifications/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateCharacterNotificationsESI(ctx, UpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionNotifications,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterNotification(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, o.IsRead)
				assert.Equal(t, int64(42), o.NotificationID)
				assert.Equal(t, sender, o.Sender)
				assert.Equal(t, "InsurancePayoutMsg", o.Type)
				assert.Equal(t, "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n", o.Text)
				assert.Equal(t, time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC), o.Timestamp)
			}
			ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids, 2)
			}
		}
	})
	t.Run("should update isRead for existing notification", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID:    c.ID,
			NotificationID: 42,
		})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		sender := factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
		data := []map[string]any{{
			"is_read":         true,
			"notification_id": 42,
			"sender_id":       sender.ID,
			"sender_type":     "corporation",
			"text":            "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n",
			"timestamp":       "2017-08-16T10:08:00Z",
			"type":            "InsurancePayoutMsg"}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/notifications/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateCharacterNotificationsESI(ctx, UpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionNotifications,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterNotification(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, o.IsRead)
			}
			ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids, 1)
			}
		}
	})
	// t.Run("should fetch multiple pages", func(t *testing.T) {
	// 	// given
	// 	testutil.TruncateTables(db)
	// 	httpmock.Reset()
	// 	c := factory.CreateCharacter()
	// 	factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
	// 	factory.CreateEveEntityCharacter(app.EveEntity{ID: 54321})
	// 	factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60014719})
	// 	factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})

	// 	data := make([]map[string]any, 0)
	// 	for i := range 2500 {
	// 		data = append(data, map[string]any{
	// 			"client_id":      54321,
	// 			"date":           "2016-10-24T09:00:00Z",
	// 			"is_buy":         true,
	// 			"is_personal":    true,
	// 			"journal_ref_id": 67890,
	// 			"location_id":    60014719,
	// 			"quantity":       1,
	// 			"transaction_id": 1000002500 - i,
	// 			"type_id":        587,
	// 			"unit_price":     1.23,
	// 		})
	// 	}
	// 	httpmock.RegisterResponder(
	// 		"GET",
	// 		fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/", c.ID),
	// 		httpmock.NewJsonResponderOrPanic(200, data))
	// 	httpmock.RegisterResponder(
	// 		"GET",
	// 		fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/?from_id=1000000001", c.ID),
	// 		httpmock.NewJsonResponderOrPanic(200, []map[string]any{
	// 			{
	// 				"client_id":      54321,
	// 				"date":           "2016-10-24T08:00:00Z",
	// 				"is_buy":         true,
	// 				"is_personal":    true,
	// 				"journal_ref_id": 67891,
	// 				"location_id":    60014719,
	// 				"quantity":       1,
	// 				"transaction_id": 1000000000,
	// 				"type_id":        587,
	// 				"unit_price":     9.23,
	// 			},
	// 		}))
	// 	// when
	// 	_, err := s.updateCharacterNotificationsESI(ctx, UpdateSectionParams{
	// 		CharacterID: c.ID,
	// 		Section:     app.SectionWalletTransactions,
	// 	})
	// 	// then
	// 	if assert.NoError(t, err) {
	// 		ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
	// 		if assert.NoError(t, err) {
	// 			assert.Len(t, ids, 2501)
	// 		}
	// 	}
	// })
}

func TestListCharacterNotifications(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	s := newCharacterService(st)
	ctx := context.Background()
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		// when
		tt, err := s.ListCharacterNotifications(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, tt, 3)
		}
	})
}

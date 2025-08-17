package characterservice

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
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestUpdateCharacterNotificationsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create new notification from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
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
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/notifications/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateNotificationsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterNotifications,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterNotification(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, o.IsRead)
				assert.Equal(t, int64(42), o.NotificationID)
				assert.Equal(t, sender, o.Sender)
				assert.Equal(t, app.InsurancePayoutMsg, o.Type)
				assert.Equal(t, "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n", o.Text)
				assert.Equal(t, time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC), o.Timestamp)
			}
			ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 1, ids.Size())
			}
		}
	})
	t.Run("should add new notification", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
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
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/notifications/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateNotificationsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterNotifications,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterNotification(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, o.IsRead)
				assert.Equal(t, int64(42), o.NotificationID)
				assert.Equal(t, sender, o.Sender)
				assert.Equal(t, app.InsurancePayoutMsg, o.Type)
				assert.Equal(t, "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n", o.Text)
				assert.Equal(t, time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC), o.Timestamp)
			}
			ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 2, ids.Size())
			}
		}
	})
	t.Run("should update isRead for existing notification", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
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
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/notifications/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateNotificationsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterNotifications,
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
				assert.Equal(t, 1, ids.Size())
			}
		}
	})
}

func TestListCharacterNotifications(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
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
		if assert.NoError(t, err) {
			assert.Len(t, tt, 2)
		}
	})
}

func TestRenderNotificationContent(t *testing.T) {
	cases := []struct {
		name        string
		notif       *app.CharacterNotification
		character   *app.Character
		wantTitle   string
		wantContent string
	}{
		{
			"corporation name for corporation notifs",
			&app.CharacterNotification{
				CharacterID: 42,
				Type:        app.StructureDestroyed,
				Title:       optional.New("Structure destroyed"),
				Sender:      &app.EveEntity{Name: "Concord"},
			}, &app.Character{
				ID: 42,
				EveCharacter: &app.EveCharacter{
					Name:        "Bruce Wayne",
					Corporation: &app.EveEntity{Name: "Wayne Technology"},
				},
			},
			"Wayne Technology: New Communication from Concord",
			"Structure destroyed",
		},
		{
			"alliance name for alliance notifs",
			&app.CharacterNotification{
				CharacterID: 42,
				Type:        app.SovStructureDestroyed,
				Title:       optional.New("Structure destroyed"),
				Sender:      &app.EveEntity{Name: "Concord"},
			}, &app.Character{
				ID: 42,
				EveCharacter: &app.EveCharacter{
					Name:        "Bruce Wayne",
					Corporation: &app.EveEntity{Name: "Wayne Technology"},
					Alliance:    &app.EveEntity{Name: "Wayne Inc"},
				},
			},
			"Wayne Inc: New Communication from Concord",
			"Structure destroyed",
		},
		{
			"character name for character notifs",
			&app.CharacterNotification{
				CharacterID: 42,
				Type:        app.StructureItemsDelivered,
				Title:       optional.New("Items delivered"),
				Sender:      &app.EveEntity{Name: "Concord"},
			}, &app.Character{
				ID: 42,
				EveCharacter: &app.EveCharacter{
					Name:        "Bruce Wayne",
					Corporation: &app.EveEntity{Name: "Wayne Technology"},
				},
			},
			"Bruce Wayne: New Communication from Concord",
			"Items delivered",
		},
		{
			"corporation for alliance notif as fallback",
			&app.CharacterNotification{
				CharacterID: 42,
				Type:        app.SovStructureDestroyed,
				Title:       optional.New("Structure destroyed"),
				Sender:      &app.EveEntity{Name: "Concord"},
			}, &app.Character{
				ID: 42,
				EveCharacter: &app.EveCharacter{
					Name:        "Bruce Wayne",
					Corporation: &app.EveEntity{Name: "Wayne Technology"},
				},
			},
			"Wayne Technology: New Communication from Concord",
			"Structure destroyed",
		},
		{
			"character name as fallback for notifs with unknown category",
			&app.CharacterNotification{
				CharacterID: 42,
				Type:        app.AgentRetiredTrigravian,
				Title:       optional.New("Agent retired"),
				Sender:      &app.EveEntity{Name: "Concord"},
			}, &app.Character{
				ID: 42,
				EveCharacter: &app.EveCharacter{
					Name:        "Bruce Wayne",
					Corporation: &app.EveEntity{Name: "Wayne Technology"},
				},
			},
			"Bruce Wayne: New Communication from Concord",
			"Agent retired",
		},
	}
	s := &CharacterService{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			title, content := s.RenderNotificationSummary(tc.character, tc.notif)
			assert.Equal(t, tc.wantTitle, title)
			assert.Equal(t, tc.wantContent, content)
		})
	}
}

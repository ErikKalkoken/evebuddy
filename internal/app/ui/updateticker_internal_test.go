package ui

import (
	"context"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

type mockApp struct {
	fyne.App

	notifications []*fyne.Notification
}

func newMockApp() *mockApp {
	a := &mockApp{
		notifications: make([]*fyne.Notification, 0),
	}
	return a
}

func (a *mockApp) SendNotification(n *fyne.Notification) {
	a.notifications = append(a.notifications, n)
}

func TestUpdateTickerNotifyExpiredTraining(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	u := newUI(st)
	ctx := context.Background()
	t.Run("send notification when watched & expired", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		a := newMockApp()
		u.fyneApp = a
		c := factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{IsTrainingWatched: true})
		// when
		err := u.notifyExpiredTraining(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, len(a.notifications), 1)
		}
	})
	t.Run("do nothing when not watched", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		a := newMockApp()
		u.fyneApp = a
		c := factory.CreateCharacter()
		// when
		err := u.notifyExpiredTraining(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, len(a.notifications), 0)
		}
	})
	t.Run("don't send notification when watched and training ongoing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		a := newMockApp()
		u.fyneApp = a
		c := factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{IsTrainingWatched: true})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		// when
		err := u.notifyExpiredTraining(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, len(a.notifications), 0)
		}
	})
}

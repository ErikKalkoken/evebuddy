package characterservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestUpdateTickerNotifyExpiredTraining(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := newCharacterService(st)
	ctx := context.Background()
	t.Run("send notification when watched & expired", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: true})
		var sendCount int
		// when
		err := cs.NotifyExpiredTraining(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 1)
		}
	})
	t.Run("do nothing when not watched", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		var sendCount int
		// when
		err := cs.NotifyExpiredTraining(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 0)
		}
	})
	t.Run("don't send notification when watched and training ongoing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: true})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		var sendCount int
		// when
		err := cs.NotifyExpiredTraining(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 0)
		}
	})
}

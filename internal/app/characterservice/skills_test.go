package characterservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestIsTrainingActive(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("should return true when training is active", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		character := factory.CreateCharacter()
		now := time.Now().UTC()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: character.ID,
			StartDate:   now.Add(-1 * time.Hour),
			FinishDate:  now.Add(3 * time.Hour),
		})
		// when
		got, err := cs.IsTrainingActive(ctx, character.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, true, got)
		}
	})
	t.Run("should return false when training is inactive", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		character := factory.CreateCharacter()
		// when
		got, err := cs.IsTrainingActive(ctx, character.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, false, got)
		}
	})
}

func TestUpdateTickerNotifyExpiredTraining(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("send notification when watched & expired", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
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
		c := factory.CreateCharacterFull()
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
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
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

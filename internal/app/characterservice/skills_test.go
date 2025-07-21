package characterservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestTotalTrainingTime(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("should return time when has valid update", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		character := factory.CreateCharacter()
		now := time.Now().UTC()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: character.ID,
			StartDate:   now.Add(-1 * time.Hour),
			FinishDate:  now.Add(3 * time.Hour),
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: character.ID,
			Section:     app.SectionCharacterSkillqueue,
			CompletedAt: now,
		})
		// when
		got, err := cs.TotalTrainingTime(ctx, character.ID)
		// then
		if assert.NoError(t, err) {
			assert.InDelta(t, 3*time.Hour, got.ValueOrZero(), float64(time.Second))
		}
	})
	t.Run("should return no time when has no valid update", func(t *testing.T) {
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
		got, err := cs.TotalTrainingTime(ctx, character.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, got.IsEmpty())
		}
	})
	t.Run("should return 0 when training is inactive", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		character := factory.CreateCharacter()
		now := time.Now().UTC()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: character.ID,
			Section:     app.SectionCharacterSkillqueue,
			CompletedAt: now,
		})
		// when
		got, err := cs.TotalTrainingTime(ctx, character.ID)
		// then
		if assert.NoError(t, err) {
			assert.EqualValues(t, 0, got.MustValue())
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
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkillqueue,
			CompletedAt: time.Now().UTC(),
		})
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

package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSkillqueueItems(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		eveType := factory.CreateEveType()
		arg := storage.SkillqueueItemParams{
			CharacterID:   c.ID,
			EveTypeID:     eveType.ID,
			FinishedLevel: 5,
			QueuePosition: 4,
		}
		// when
		err := st.CreateCharacterSkillqueueItem(ctx, arg)
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterSkillqueueItem(ctx, c.ID, 4)
		require.NoError(t, err)
		assert.Equal(t, arg.CharacterID, got.CharacterID)
		assert.Equal(t, arg.EveTypeID, got.SkillID)
		assert.Equal(t, arg.FinishedLevel, got.FinishedLevel)
		assert.Equal(t, arg.QueuePosition, got.QueuePosition)
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		eveType := factory.CreateEveType()
		finishDate := time.Now().Add(10 * time.Hour)
		startDate := time.Now().Add(-4 * time.Hour)
		arg := storage.SkillqueueItemParams{
			CharacterID:     c.ID,
			EveTypeID:       eveType.ID,
			FinishDate:      finishDate,
			FinishedLevel:   5,
			LevelEndSP:      10000,
			LevelStartSP:    100,
			QueuePosition:   4,
			StartDate:       startDate,
			TrainingStartSP: 50,
		}
		// when
		err := st.CreateCharacterSkillqueueItem(ctx, arg)
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterSkillqueueItem(ctx, c.ID, 4)
		require.NoError(t, err)
		assert.Equal(t, arg.CharacterID, got.CharacterID)
		assert.Equal(t, arg.EveTypeID, got.SkillID)
		assert.Equal(t, arg.FinishedLevel, got.FinishedLevel)
		assert.Equal(t, arg.LevelEndSP, got.LevelEndSP)
		assert.Equal(t, arg.QueuePosition, got.QueuePosition)
		assert.Equal(t, arg.TrainingStartSP, got.TrainingStartSP)
		xassert.EqualTime(t, arg.FinishDate, got.FinishDate)
		xassert.EqualTime(t, arg.StartDate, got.StartDate)
	})
	t.Run("can list items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		// when
		ii, err := st.ListCharacterSkillqueueItems(ctx, c.ID)
		// then
		require.NoError(t, err)
		assert.Len(t, ii, 3)
	})
	t.Run("can replace items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		eveType := factory.CreateEveType()
		arg := storage.SkillqueueItemParams{
			EveTypeID:     eveType.ID,
			FinishedLevel: 5,
			CharacterID:   c.ID,
			QueuePosition: 0,
			StartDate:     time.Now().Add(1 * time.Hour),
			FinishDate:    time.Now().Add(3 * time.Hour),
		}
		// when
		err := st.ReplaceCharacterSkillqueueItems(ctx, c.ID, []storage.SkillqueueItemParams{arg})
		// then
		require.NoError(t, err)
		ii, err := st.ListCharacterSkillqueueItems(ctx, c.ID)
		require.NoError(t, err)
		assert.Len(t, ii, 1)
	})
}

func TestSkillqueueItems_GetCharacterTotalTrainingTime(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can calculate total training time", func(t *testing.T) {
		// given
		now := time.Now()
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   now.Add(1 * time.Hour),
			FinishDate:  now.Add(3 * time.Hour),
		})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   now.Add(3 * time.Hour),
			FinishDate:  now.Add(4 * time.Hour),
		})
		// when
		v, err := st.GetCharacterTotalTrainingTime(ctx, c.ID)
		// then
		require.NoError(t, err)
		xassert.EqualDuration(t, 3*time.Hour, v, time.Second)
	})
	t.Run("should report when training is not active", func(t *testing.T) {
		// given
		now := time.Now()
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   now.Add(-3 * time.Hour),
			FinishDate:  now.Add(-1 * time.Hour),
		})
		// when
		v, err := st.GetCharacterTotalTrainingTime(ctx, c.ID)
		// then
		require.NoError(t, err)
		assert.EqualValues(t, 0, v)
	})
}

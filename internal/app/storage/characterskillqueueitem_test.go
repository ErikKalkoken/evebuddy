package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSkillqueueItems(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		eveType := factory.CreateEveType()
		arg := storage.SkillqueueItemParams{
			EveTypeID:     eveType.ID,
			FinishedLevel: 5,
			CharacterID:   c.ID,
			QueuePosition: 4,
		}
		// when
		err := st.CreateCharacterSkillqueueItem(ctx, arg)
		// then
		if assert.NoError(t, err) {
			got, err := st.GetCharacterSkillqueueItem(ctx, c.ID, 4)
			if assert.NoError(t, err) {
				assert.Equal(t, c.ID, got.CharacterID)
				assert.Equal(t, 5, got.FinishedLevel)
				assert.Equal(t, eveType.ID, got.SkillID)
			}
		}
	})
	t.Run("can list items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		// when
		ii, err := st.ListCharacterSkillqueueItems(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ii, 3)
		}
	})
	t.Run("can replace items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
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
		if assert.NoError(t, err) {
			ii, err := st.ListCharacterSkillqueueItems(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ii, 1)
			}
		}
	})

}

func TestSkillqueueItemsCalculateTrainingTime(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can calculate total training time", func(t *testing.T) {
		// given
		now := time.Now()
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
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
		if assert.NoError(t, err) {
			assert.InDelta(t, 3*time.Hour, v, float64(time.Second*1))
		}
	})
	t.Run("should return 0 when training is not active", func(t *testing.T) {
		// given
		now := time.Now()
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   now.Add(-3 * time.Hour),
			FinishDate:  now.Add(-1 * time.Hour),
		})
		// when
		v, err := st.GetCharacterTotalTrainingTime(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.EqualValues(t, 0, v)
		}
	})
}

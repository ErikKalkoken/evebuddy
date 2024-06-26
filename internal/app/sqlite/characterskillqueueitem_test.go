package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSkillqueueItems(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		eveType := factory.CreateEveType()
		arg := sqlite.SkillqueueItemParams{
			EveTypeID:     eveType.ID,
			FinishedLevel: 5,
			CharacterID:   c.ID,
			QueuePosition: 4,
		}
		// when
		err := r.CreateSkillqueueItem(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := r.GetSkillqueueItem(ctx, c.ID, 4)
			if assert.NoError(t, err) {
				assert.Equal(t, 5, i.FinishedLevel)
			}
		}
	})
	t.Run("can list items", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(sqlite.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(sqlite.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(sqlite.SkillqueueItemParams{CharacterID: c.ID})
		// when
		ii, err := r.ListSkillqueueItems(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ii, 3)
		}
	})
	t.Run("can replace items", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(sqlite.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(sqlite.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(sqlite.SkillqueueItemParams{CharacterID: c.ID})
		eveType := factory.CreateEveType()
		arg := sqlite.SkillqueueItemParams{
			EveTypeID:     eveType.ID,
			FinishedLevel: 5,
			CharacterID:   c.ID,
			QueuePosition: 0,
			StartDate:     time.Now().Add(1 * time.Hour),
			FinishDate:    time.Now().Add(3 * time.Hour),
		}
		// when
		err := r.ReplaceCharacterSkillqueueItems(ctx, c.ID, []sqlite.SkillqueueItemParams{arg})
		// then
		if assert.NoError(t, err) {
			ii, err := r.ListSkillqueueItems(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ii, 1)
			}
		}
	})

}

func TestSkillqueueItemsCalculateTrainingTime(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can calculate total training time", func(t *testing.T) {
		// given
		now := time.Now()
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(sqlite.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   now.Add(1 * time.Hour),
			FinishDate:  now.Add(3 * time.Hour),
		})
		factory.CreateCharacterSkillqueueItem(sqlite.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   now.Add(3 * time.Hour),
			FinishDate:  now.Add(4 * time.Hour),
		})
		// when
		v, err := r.GetTotalTrainingTime(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.InDelta(t, 3*time.Hour, v.MustValue(), float64(time.Second*1))
		}
	})
}

package storage_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestSkillqueueItems(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		eveType := factory.CreateEveType()
		arg := storage.SkillqueueItemParams{
			EveTypeID:     eveType.ID,
			FinishedLevel: 5,
			MyCharacterID: c.ID,
			QueuePosition: 4,
		}
		// when
		err := r.CreateSkillqueueItem(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := r.GetSkillqueueItems(ctx, c.ID, 4)
			if assert.NoError(t, err) {
				assert.Equal(t, 5, i.FinishedLevel)
			}
		}
	})
	t.Run("can list items", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		factory.CreateSkillqueueItem(storage.SkillqueueItemParams{MyCharacterID: c.ID})
		factory.CreateSkillqueueItem(storage.SkillqueueItemParams{MyCharacterID: c.ID})
		factory.CreateSkillqueueItem(storage.SkillqueueItemParams{MyCharacterID: c.ID})
		// when
		ii, err := r.ListSkillqueueItems(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ii, 3)
		}
	})
	t.Run("can delete items", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		factory.CreateSkillqueueItem(storage.SkillqueueItemParams{MyCharacterID: c.ID})
		factory.CreateSkillqueueItem(storage.SkillqueueItemParams{MyCharacterID: c.ID})
		factory.CreateSkillqueueItem(storage.SkillqueueItemParams{MyCharacterID: c.ID})
		eveType := factory.CreateEveType()
		arg := storage.SkillqueueItemParams{
			EveTypeID:     eveType.ID,
			FinishedLevel: 5,
			MyCharacterID: c.ID,
			QueuePosition: 0,
		}
		// when
		err := r.ReplaceSkillqueueItems(ctx, c.ID, []storage.SkillqueueItemParams{arg})
		// then
		if assert.NoError(t, err) {
			ii, err := r.ListSkillqueueItems(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ii, 1)
			}
		}
	})
	t.Run("can replace queue", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		factory.CreateSkillqueueItem(storage.SkillqueueItemParams{MyCharacterID: c.ID})
		factory.CreateSkillqueueItem(storage.SkillqueueItemParams{MyCharacterID: c.ID})
		factory.CreateSkillqueueItem(storage.SkillqueueItemParams{MyCharacterID: c.ID})
		// when
		err := r.DeleteSkillqueueItems(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			ii, err := r.ListSkillqueueItems(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ii, 0)
			}
		}
	})
}

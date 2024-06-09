package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestCharacterUpdateStatus(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		completedAt := time.Now()
		startedAt := time.Now()
		arg := storage.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Error:       "error",
			Section:     model.SectionSkillqueue,
			ContentHash: "content-hash",
			CompletedAt: completedAt,
			StartedAt:   startedAt,
		}
		// when
		err := r.UpdateOrCreateCharacterUpdateStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterUpdateStatus(ctx, c.ID, model.SectionSkillqueue)
			if assert.NoError(t, err) {
				assert.Equal(t, "error", l.ErrorMessage)
				assert.Equal(t, "content-hash", l.ContentHash)
				assert.Equal(t, completedAt.UTC(), l.CompletedAt.UTC())
				assert.Equal(t, startedAt.UTC(), l.StartedAt.UTC())
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.SectionSkillqueue,
		})
		updatedAt := time.Now().Add(1 * time.Hour)
		arg := storage.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.SectionSkillqueue,
			Error:       "error",
			ContentHash: "content-hash",
			CompletedAt: updatedAt,
		}
		// when
		err := r.UpdateOrCreateCharacterUpdateStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterUpdateStatus(ctx, c.ID, model.SectionSkillqueue)
			if assert.NoError(t, err) {
				assert.Equal(t, "content-hash", l.ContentHash)
				assert.Equal(t, "error", l.ErrorMessage)
				assert.Equal(t, updatedAt.UTC(), l.CompletedAt.UTC())
			}
		}
	})
	t.Run("can list", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.SectionSkillqueue,
		})
		factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.SectionImplants,
		})
		// when
		oo, err := r.ListCharacterUpdateStatus(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 2)
		}
	})
}

func TestSetCharacterUpdateStatusError(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can set from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		arg := storage.CharacterUpdateStatusOptionals{
			Error: storage.NewNullString("error"),
		}
		err := r.UpdateOrCreateCharacterUpdateStatus2(ctx, c.ID, model.SectionImplants, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterUpdateStatus(ctx, c.ID, model.SectionImplants)
			if assert.NoError(t, err) {
				assert.Equal(t, "", l.ContentHash)
				assert.Equal(t, "error", l.ErrorMessage)
				assert.True(t, l.CompletedAt.IsZero())
			}
		}
	})
	t.Run("can set existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		x := factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.SectionImplants,
		})
		// when
		arg := storage.CharacterUpdateStatusOptionals{
			Error: storage.NewNullString("error"),
		}
		err := r.UpdateOrCreateCharacterUpdateStatus2(ctx, c.ID, x.Section, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterUpdateStatus(ctx, c.ID, x.Section)
			if assert.NoError(t, err) {
				assert.Equal(t, x.ContentHash, l.ContentHash)
				assert.Equal(t, "error", l.ErrorMessage)
				assert.Equal(t, x.CompletedAt.UTC(), l.CompletedAt.UTC())
				assert.Equal(t, x.StartedAt.UTC(), l.StartedAt.UTC())
			}
		}
	})

}

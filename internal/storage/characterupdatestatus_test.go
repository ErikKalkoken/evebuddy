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

func TestCharacterSectionStatus(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		completedAt := time.Now()
		startedAt := time.Now()
		arg := storage.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Error:       "error",
			Section:     model.SectionSkillqueue,
			ContentHash: "content-hash",
			CompletedAt: completedAt,
			StartedAt:   startedAt,
		}
		// when
		x1, err := r.UpdateOrCreateCharacterSectionStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "error", x1.ErrorMessage)
			assert.Equal(t, "content-hash", x1.ContentHash)
			assert.Equal(t, completedAt.UTC(), x1.CompletedAt.UTC())
			assert.Equal(t, startedAt.UTC(), x1.StartedAt.UTC())
			x2, err := r.GetCharacterSectionStatus(ctx, c.ID, model.SectionSkillqueue)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     model.SectionSkillqueue,
		})
		updatedAt := time.Now().Add(1 * time.Hour)
		arg := storage.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     model.SectionSkillqueue,
			Error:       "error",
			ContentHash: "content-hash",
			CompletedAt: updatedAt,
		}
		// when
		x1, err := r.UpdateOrCreateCharacterSectionStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "content-hash", x1.ContentHash)
			assert.Equal(t, "error", x1.ErrorMessage)
			assert.Equal(t, updatedAt.UTC(), x1.CompletedAt.UTC())
			x2, err := r.GetCharacterSectionStatus(ctx, c.ID, model.SectionSkillqueue)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("can list", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     model.SectionSkillqueue,
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     model.SectionImplants,
		})
		// when
		oo, err := r.ListCharacterSectionStatus(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 2)
		}
	})
}

func TestUpdateOrCreateCharacterSectionStatus2(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can set from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		error := "error"
		arg := storage.CharacterSectionStatusOptionals{
			Error: &error,
		}
		x1, err := r.UpdateOrCreateCharacterSectionStatus2(ctx, c.ID, model.SectionImplants, arg)
		// then
		if assert.NoError(t, err) {
			if assert.NoError(t, err) {
				assert.Equal(t, "", x1.ContentHash)
				assert.Equal(t, "error", x1.ErrorMessage)
				assert.True(t, x1.CompletedAt.IsZero())
			}
			x2, err := r.GetCharacterSectionStatus(ctx, c.ID, model.SectionImplants)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("can set existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		x := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     model.SectionImplants,
		})
		// when
		s := "error"
		arg := storage.CharacterSectionStatusOptionals{
			Error: &s,
		}
		x1, err := r.UpdateOrCreateCharacterSectionStatus2(ctx, c.ID, x.Section, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x.ContentHash, x1.ContentHash)
			assert.Equal(t, "error", x1.ErrorMessage)
			assert.Equal(t, x.CompletedAt, x1.CompletedAt)
			assert.Equal(t, x.StartedAt, x1.StartedAt)
			x2, err := r.GetCharacterSectionStatus(ctx, c.ID, x.Section)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

package storage_test

import (
	"context"
	"database/sql"
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
		updatedAt := sql.NullTime{Time: time.Now(), Valid: true}
		arg := storage.CharacterUpdateStatusParams{
			CharacterID:   c.ID,
			Error:         "error",
			Section:       model.CharacterSectionSkillqueue,
			ContentHash:   "content-hash",
			LastUpdatedAt: updatedAt,
		}
		// when
		err := r.UpdateOrCreateCharacterUpdateStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterUpdateStatus(ctx, c.ID, model.CharacterSectionSkillqueue)
			if assert.NoError(t, err) {
				assert.Equal(t, "error", l.ErrorMessage)
				assert.Equal(t, "content-hash", l.ContentHash)
				assert.Equal(t, updatedAt.Time.UTC(), l.LastUpdatedAt.Time.UTC())
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterUpdateStatus(storage.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionSkillqueue,
		})
		updatedAt := sql.NullTime{Time: time.Now().Add(1 * time.Hour), Valid: true}
		arg := storage.CharacterUpdateStatusParams{
			CharacterID:   c.ID,
			Section:       model.CharacterSectionSkillqueue,
			Error:         "error",
			ContentHash:   "content-hash",
			LastUpdatedAt: updatedAt,
		}
		// when
		err := r.UpdateOrCreateCharacterUpdateStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterUpdateStatus(ctx, c.ID, model.CharacterSectionSkillqueue)
			if assert.NoError(t, err) {
				assert.Equal(t, "content-hash", l.ContentHash)
				assert.Equal(t, "error", l.ErrorMessage)
				assert.Equal(t, updatedAt.Time.UTC(), l.LastUpdatedAt.Time.UTC())
			}
		}
	})
	t.Run("can list", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterUpdateStatus(storage.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionSkillqueue,
		})
		factory.CreateCharacterUpdateStatus(storage.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionImplants,
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
		err := r.SetCharacterUpdateStatusError(ctx, c.ID, model.CharacterSectionImplants, "error")
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterUpdateStatus(ctx, c.ID, model.CharacterSectionImplants)
			if assert.NoError(t, err) {
				assert.Equal(t, "", l.ContentHash)
				assert.Equal(t, "error", l.ErrorMessage)
				assert.False(t, l.LastUpdatedAt.Valid)
			}
		}
	})
	t.Run("can set existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		x := factory.CreateCharacterUpdateStatus(storage.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionImplants,
		})
		// when
		err := r.SetCharacterUpdateStatusError(ctx, c.ID, x.Section, "error")
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterUpdateStatus(ctx, c.ID, x.Section)
			if assert.NoError(t, err) {
				assert.Equal(t, x.ContentHash, l.ContentHash)
				assert.Equal(t, "error", l.ErrorMessage)
				assert.Equal(t, x.LastUpdatedAt.Time.UTC(), l.LastUpdatedAt.Time.UTC())
			}
		}
	})

}

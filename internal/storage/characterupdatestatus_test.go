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

func TestMyCharacterUpdateStatus(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		updatedAt := time.Now()
		arg := storage.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionSkillqueue,
			ContentHash: "content-hash",
			UpdatedAt:   updatedAt,
		}
		// when
		err := r.UpdateOrCreateCharacterUpdateStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterUpdateStatus(ctx, c.ID, model.CharacterSectionSkillqueue)
			if assert.NoError(t, err) {
				assert.Equal(t, "content-hash", l.ContentHash)
				assert.Equal(t, updatedAt.Unix(), l.UpdatedAt.Unix())
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
		updatedAt := time.Now().Add(1 * time.Hour)
		arg := storage.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionSkillqueue,
			ContentHash: "content-hash",
			UpdatedAt:   updatedAt,
		}
		// when
		err := r.UpdateOrCreateCharacterUpdateStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterUpdateStatus(ctx, c.ID, model.CharacterSectionSkillqueue)
			if assert.NoError(t, err) {
				assert.Equal(t, "content-hash", l.ContentHash)
				assert.Equal(t, updatedAt.Unix(), l.UpdatedAt.Unix())
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

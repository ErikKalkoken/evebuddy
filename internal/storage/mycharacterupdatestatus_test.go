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
		c := factory.CreateMyCharacter()
		updatedAt := time.Now()
		arg := storage.MyCharacterUpdateStatusParams{
			MyCharacterID: c.ID,
			Section:       model.UpdateSectionSkillqueue,
			ContentHash:   "content-hash",
			UpdatedAt:     updatedAt,
		}
		// when
		err := r.UpdateOrCreateMyCharacterUpdateStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetMyCharacterUpdateStatus(ctx, c.ID, model.UpdateSectionSkillqueue)
			if assert.NoError(t, err) {
				assert.Equal(t, "content-hash", l.ContentHash)
				assert.Equal(t, updatedAt.Unix(), l.UpdatedAt.Unix())
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		factory.CreateMyCharacterUpdateStatus(storage.MyCharacterUpdateStatusParams{
			MyCharacterID: c.ID,
			Section:       model.UpdateSectionSkillqueue,
		})
		updatedAt := time.Now().Add(1 * time.Hour)
		arg := storage.MyCharacterUpdateStatusParams{
			MyCharacterID: c.ID,
			Section:       model.UpdateSectionSkillqueue,
			ContentHash:   "content-hash",
			UpdatedAt:     updatedAt,
		}
		// when
		err := r.UpdateOrCreateMyCharacterUpdateStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetMyCharacterUpdateStatus(ctx, c.ID, model.UpdateSectionSkillqueue)
			if assert.NoError(t, err) {
				assert.Equal(t, "content-hash", l.ContentHash)
				assert.Equal(t, updatedAt.Unix(), l.UpdatedAt.Unix())
			}
		}
	})
}
package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestCharacterSectionStatus(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can list", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkillqueue,
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		// when
		oo, err := st.ListCharacterSectionStatus(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 2)
		}
	})
	t.Run("can set from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		// when
		error := "error"
		arg := storage.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      app.SectionCharacterImplants,
			ErrorMessage: &error,
		}
		x1, err := st.UpdateOrCreateCharacterSectionStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			if assert.NoError(t, err) {
				assert.Equal(t, "", x1.ContentHash)
				assert.Equal(t, "error", x1.ErrorMessage)
				assert.True(t, x1.CompletedAt.IsZero())
				assert.False(t, x1.UpdatedAt.IsZero())
			}
			x2, err := st.GetCharacterSectionStatus(ctx, c.ID, app.SectionCharacterImplants)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("can set existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		x := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		// when
		s := "error"
		arg := storage.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      x.Section,
			ErrorMessage: &s,
		}
		x1, err := st.UpdateOrCreateCharacterSectionStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x.ContentHash, x1.ContentHash)
			assert.Equal(t, "error", x1.ErrorMessage)
			assert.Equal(t, x.CompletedAt, x1.CompletedAt)
			assert.Equal(t, x.StartedAt, x1.StartedAt)
			assert.False(t, x1.UpdatedAt.IsZero())
			x2, err := st.GetCharacterSectionStatus(ctx, c.ID, x.Section)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

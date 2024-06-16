package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/testutil"
)

func TestGeneralSectionStatus(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can list", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionSkillqueue,
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionImplants,
		})
		// when
		oo, err := r.ListCharacterSectionStatus(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 2)
		}
	})
	t.Run("can set from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		error := "error"
		arg := sqlite.UpdateOrCreateGeneralSectionStatusParams{
			Section: app.SectionEveCategories,
			Error:   &error,
		}
		x1, err := r.UpdateOrCreateGeneralSectionStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			if assert.NoError(t, err) {
				assert.Equal(t, "", x1.ContentHash)
				assert.Equal(t, "error", x1.ErrorMessage)
				assert.True(t, x1.CompletedAt.IsZero())
			}
			x2, err := r.GetGeneralSectionStatus(ctx, app.SectionEveCategories)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("can set existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: app.SectionEveCategories,
		})
		// when
		error := "error"
		arg := sqlite.UpdateOrCreateGeneralSectionStatusParams{
			Section: app.SectionEveCategories,
			Error:   &error,
		}
		x1, err := r.UpdateOrCreateGeneralSectionStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x.ContentHash, x1.ContentHash)
			assert.Equal(t, "error", x1.ErrorMessage)
			assert.Equal(t, x.CompletedAt, x1.CompletedAt)
			assert.Equal(t, x.StartedAt, x1.StartedAt)
			x2, err := r.GetGeneralSectionStatus(ctx, app.SectionEveCategories)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

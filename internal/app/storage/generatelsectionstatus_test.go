package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestGeneralSectionStatus(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can list", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		s1 := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: app.SectionEveTypes,
		})
		s2 := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: app.SectionEveCharacters,
		})
		// when
		oo, err := st.ListGeneralSectionStatus(ctx)
		// then
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, []*app.GeneralSectionStatus{s1, s2}, oo)
		}
	})
	t.Run("can set from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		error := "error"
		arg := storage.UpdateOrCreateGeneralSectionStatusParams{
			Section: app.SectionEveTypes,
			Error:   &error,
		}
		x1, err := st.UpdateOrCreateGeneralSectionStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			if assert.NoError(t, err) {
				assert.Equal(t, "", x1.ContentHash)
				assert.Equal(t, "error", x1.ErrorMessage)
				assert.True(t, x1.CompletedAt.IsZero())
			}
			x2, err := st.GetGeneralSectionStatus(ctx, app.SectionEveTypes)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("can set existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: app.SectionEveTypes,
		})
		// when
		error := "error"
		arg := storage.UpdateOrCreateGeneralSectionStatusParams{
			Section: app.SectionEveTypes,
			Error:   &error,
		}
		x1, err := st.UpdateOrCreateGeneralSectionStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x.ContentHash, x1.ContentHash)
			assert.Equal(t, "error", x1.ErrorMessage)
			assert.Equal(t, x.CompletedAt, x1.CompletedAt)
			assert.Equal(t, x.StartedAt, x1.StartedAt)
			x2, err := st.GetGeneralSectionStatus(ctx, app.SectionEveTypes)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

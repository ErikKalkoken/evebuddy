package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestGeneralSectionStatus(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can list", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		s1 := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: app.SectionEveTypes,
		})
		s2 := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: app.SectionEveCharacters,
		})
		// when
		oo, err := st.ListGeneralSectionStatus(ctx)
		// then
		require.NoError(t, err)
		assert.ElementsMatch(t, []*app.EveUniverseSectionStatus{s1, s2}, oo)
	})
	t.Run("can set from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		x := "error"
		arg := storage.UpdateOrCreateGeneralSectionStatusParams{
			Section: app.SectionEveTypes,
			Error:   &x,
		}
		x1, err := st.UpdateOrCreateGeneralSectionStatus(ctx, arg)
		// then
		require.NoError(t, err)
		require.NoError(t, err)
		xassert.Equal(t, "", x1.ContentHash)
		xassert.Equal(t, "error", x1.ErrorMessage)
		assert.True(t, x1.CompletedAt.IsZero())
		x2, err := st.GetGeneralSectionStatus(ctx, app.SectionEveTypes)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
	t.Run("can set existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: app.SectionEveTypes,
		})
		// when
		y := "error"
		arg := storage.UpdateOrCreateGeneralSectionStatusParams{
			Section: app.SectionEveTypes,
			Error:   &y,
		}
		x1, err := st.UpdateOrCreateGeneralSectionStatus(ctx, arg)
		// then
		require.NoError(t, err)
		xassert.Equal(t, x.ContentHash, x1.ContentHash)
		xassert.Equal(t, "error", x1.ErrorMessage)
		xassert.Equal(t, x.CompletedAt, x1.CompletedAt)
		xassert.Equal(t, x.StartedAt, x1.StartedAt)
		x2, err := st.GetGeneralSectionStatus(ctx, app.SectionEveTypes)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
	t.Run("can clear started at", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		running := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section:      app.SectionEveTypes,
			ErrorMessage: "error",
		})
		factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: app.SectionEveCharacters,
		})
		_, err := st.UpdateOrCreateGeneralSectionStatus(ctx, storage.UpdateOrCreateGeneralSectionStatusParams{
			Section:   app.SectionEveCharacters,
			StartedAt: new(optional.Optional[time.Time]),
		})
		require.NoError(t, err)
		// when
		oo, err := st.ClearGeneralSectionStatusStartedAt(ctx)
		// then
		require.NoError(t, err)
		require.Len(t, oo, 1)
		got := oo[0]
		assert.True(t, got.StartedAt.IsZero())
		assert.Equal(t, running.Section, got.Section)
		assert.True(t, running.CompletedAt.Equal(got.CompletedAt))
		assert.Equal(t, running.ContentHash, got.ContentHash)
		assert.Equal(t, running.ErrorMessage, got.ErrorMessage)
		assert.True(t, running.UpdatedAt.Equal(got.UpdatedAt))
		x, err := st.GetGeneralSectionStatus(ctx, app.SectionEveTypes)
		require.NoError(t, err)
		assert.True(t, x.StartedAt.IsZero())
	})
}

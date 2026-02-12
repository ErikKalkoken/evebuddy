package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveGroup(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new and get", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateEveCategory()
		arg := storage.CreateEveGroupParams{
			ID:          42,
			Name:        "name",
			CategoryID:  c.ID,
			IsPublished: true,
		}
		// when
		err := st.CreateEveGroup(ctx, arg)
		// then
		if assert.NoError(t, err) {
			g, err := st.GetEveGroup(ctx, 42)
			if assert.NoError(t, err) {
				xassert.Equal(t, 42, g.ID)
				xassert.Equal(t, "name", g.Name)
				xassert.Equal(t, true, g.IsPublished)
				xassert.Equal(t, c, g.Category)
			}
		}
	})
	t.Run("can get already existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		g := factory.CreateEveGroup()
		// when
		got, err := st.GetOrCreateEveGroup(ctx, storage.CreateEveGroupParams{
			ID: g.ID,
		})
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, g.ID, got.ID)
			xassert.Equal(t, g.Name, got.Name)
			xassert.Equal(t, g.IsPublished, got.IsPublished)
			xassert.Equal(t, g.Category, got.Category)
		}
	})
	t.Run("can create new when not existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateEveCategory()
		// when
		got, err := st.GetOrCreateEveGroup(ctx, storage.CreateEveGroupParams{
			ID:          42,
			Name:        "Alpha",
			CategoryID:  c.ID,
			IsPublished: true,
		})
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, 42, got.ID)
			xassert.Equal(t, "Alpha", got.Name)
			assert.True(t, got.IsPublished)
			xassert.Equal(t, c, got.Category)
		}
	})
}

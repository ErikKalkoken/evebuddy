package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
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
				assert.Equal(t, int32(42), g.ID)
				assert.Equal(t, "name", g.Name)
				assert.Equal(t, true, g.IsPublished)
				assert.Equal(t, c, g.Category)
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
			assert.EqualValues(t, g.ID, got.ID)
			assert.Equal(t, g.Name, got.Name)
			assert.Equal(t, g.IsPublished, got.IsPublished)
			assert.Equal(t, g.Category, got.Category)
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
			assert.EqualValues(t, 42, got.ID)
			assert.Equal(t, "Alpha", got.Name)
			assert.True(t, got.IsPublished)
			assert.Equal(t, c, got.Category)
		}
	})
}

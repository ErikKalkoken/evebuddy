package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestEveCategory(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		arg := storage.CreateEveCategoryParams{
			ID:          42,
			Name:        "Alpha",
			IsPublished: true,
		}
		// when
		c, err := st.CreateEveCategory(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.EqualValues(t, 42, c.ID)
			assert.EqualValues(t, "Alpha", c.Name)
			assert.True(t, c.IsPublished)
		}
	})
	t.Run("can get", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateEveCategory()
		// when
		c2, err := st.GetEveCategory(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c2, c2)
		}
	})
	t.Run("can get already existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateEveCategory()
		// when
		c2, err := st.GetOrCreateEveCategory(ctx, storage.CreateEveCategoryParams{
			ID: c1.ID,
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c2, c2)
		}
	})
	t.Run("can create when not existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		c, err := st.GetOrCreateEveCategory(ctx, storage.CreateEveCategoryParams{
			ID:          42,
			Name:        "Alpha",
			IsPublished: true,
		})
		// then
		if assert.NoError(t, err) {
			assert.EqualValues(t, 42, c.ID)
			assert.EqualValues(t, "Alpha", c.Name)
			assert.True(t, c.IsPublished)
		}
	})
}

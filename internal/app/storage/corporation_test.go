package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestCorporation(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		ec := factory.CreateEveCorporation()
		// when
		err := st.CreateCorporation(ctx, ec.ID)
		// then
		if assert.NoError(t, err) {
			r, err := st.GetCorporation(ctx, ec.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, ec.Name, r.Corporation.Name)
			}
		}
	})
	t.Run("raise specfic error when tyring to re-create existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		// when
		err := st.CreateCorporation(ctx, c.ID)
		// then
		assert.ErrorIs(t, err, app.ErrAlreadyExists)
	})
	t.Run("can fetch by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCorporation()
		// when
		c2, err := st.GetEveCorporation(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.Corporation.Name, c2.Name)
		}
	})
	t.Run("can create when not exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		ec := factory.CreateEveCorporation()
		// when
		c, err := st.GetOrCreateCorporation(ctx, ec.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, ec.Name, c.Corporation.Name)
		}
	})
	t.Run("can get when exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCorporation()
		// when
		c2, err := st.GetOrCreateCorporation(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c2, c1)
		}
	})
}

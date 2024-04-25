package storage_test

import (
	"context"
	"example/evebuddy/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveType(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		g := factory.CreateEveGroup()
		// when
		err := r.CreateEveType(ctx, 42, "description", g.ID, "name", true)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetEveType(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), x.ID)
				assert.Equal(t, "name", x.Name)
				assert.Equal(t, true, x.IsPublished)
				assert.Equal(t, g, x.Group)
			}
		}
	})
}

package storage_test

import (
	"context"
	"example/evebuddy/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveGroup(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateEveCategory()
		// when
		err := r.CreateEveGroup(ctx, 42, c.ID, "dummy", true)
		// then
		if assert.NoError(t, err) {
			g, err := r.GetEveGroup(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), g.ID)
				assert.Equal(t, "dummy", g.Name)
				assert.Equal(t, true, g.IsPublished)
				assert.Equal(t, c, g.Category)
			}
		}
	})
}

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
		g1, err := r.CreateEveGroup(ctx, 42, c.ID, "dummy", true)
		// then
		if assert.NoError(t, err) {
			g2, err := r.GetEveGroup(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, g1, g2)
			}
		}
	})
}

package storage_test

import (
	"context"
	"example/evebuddy/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveCategory(t *testing.T) {
	db, r, _ := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		c1, err := r.CreateEveCategory(ctx, 42, "dummy", true)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetEveCategory(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, c1, c2)
			}
		}
	})
}

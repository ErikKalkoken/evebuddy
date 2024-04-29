package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
)

func TestEveCategory(t *testing.T) {
	db, r, _ := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		c1, err := r.CreateEveCategory(ctx, 42, "name", true)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetEveCategory(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, c1, c2)
			}
		}
	})
}

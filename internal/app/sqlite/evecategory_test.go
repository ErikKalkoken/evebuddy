package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/testutil"
)

func TestEveCategory(t *testing.T) {
	db, r, _ := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		arg := sqlite.CreateEveCategoryParams{
			ID:          42,
			Name:        "name",
			IsPublished: true,
		}
		// when
		c1, err := r.CreateEveCategory(ctx, arg)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetEveCategory(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, c1, c2)
			}
		}
	})
}

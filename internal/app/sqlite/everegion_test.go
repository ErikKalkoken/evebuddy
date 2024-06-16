package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/testutil"
)

func TestEveRegion(t *testing.T) {
	db, r, _ := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		arg := sqlite.CreateEveRegionParams{
			ID:          42,
			Description: "description",
			Name:        "name",
		}
		// when
		x1, err := r.CreateEveRegion(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x2, err := r.GetEveRegion(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

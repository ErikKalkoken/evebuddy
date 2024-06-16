package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/testutil"
)

func TestEveConstellation(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		region := factory.CreateEveRegion()
		arg := sqlite.CreateEveConstellationParams{
			ID:       42,
			RegionID: region.ID,
			Name:     "name",
		}
		// when
		err := r.CreateEveConstellation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := r.GetEveConstellation(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), o.ID)
				assert.Equal(t, "name", o.Name)
				assert.Equal(t, region, o.Region)
			}
		}
	})
}

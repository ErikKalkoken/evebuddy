package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveConstellation(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		region := factory.CreateEveRegion()
		arg := storage.CreateEveConstellationParams{
			ID:       42,
			RegionID: region.ID,
			Name:     "name",
		}
		// when
		err := st.CreateEveConstellation(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetEveConstellation(ctx, 42)
		require.NoError(t, err)
		xassert.Equal(t, 42, o.ID)
		xassert.Equal(t, "name", o.Name)
		xassert.Equal(t, region, o.Region)
	})
}

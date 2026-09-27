package storage_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveRegion(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		arg := storage.CreateEveRegionParams{
			ID:          42,
			Description: optional.New("description"),
			Name:        "name",
		}
		// when
		x1, err := st.CreateEveRegion(ctx, arg)
		// then
		require.NoError(t, err)
		x2, err := st.GetEveRegion(ctx, 42)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
	t.Run("can list IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		r1 := factory.CreateEveRegion()
		r2 := factory.CreateEveRegion()
		// when
		got, err := st.ListEveRegionIDs(ctx)
		require.NoError(t, err)
		want := set.Of(r1.ID, r2.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("can return missing IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		r1 := factory.CreateEveRegion(storage.CreateEveRegionParams{ID: 42})
		// when
		got, err := st.MissingEveRegions(ctx, set.Of(r1.ID, 99))
		require.NoError(t, err)
		want := set.Of[int64](99)
		xassert.Equal(t, want, got)
	})
}

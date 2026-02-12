package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestStorage_UpdateOrCreateEveMarketPrice(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		x, err := st.UpdateOrCreateEveMarketPrice(ctx, storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: optional.New(1.23),
			AveragePrice:  optional.New(4.56),
		})
		// then
		require.NoError(t, err)
		xassert.Equal(t, 42, x.TypeID)
		xassert.Equal(t, 1.23, x.AdjustedPrice.ValueOrZero())
		xassert.Equal(t, 4.56, x.AveragePrice.ValueOrZero())
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: optional.New(4.0),
			AveragePrice:  optional.New(5.0),
		})
		// when
		x, err := st.UpdateOrCreateEveMarketPrice(ctx, storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: optional.New(1.23),
			AveragePrice:  optional.New(4.56),
		})
		// then
		require.NoError(t, err)
		xassert.Equal(t, 42, x.TypeID)
		xassert.Equal(t, 1.23, x.AdjustedPrice.ValueOrZero())
		xassert.Equal(t, 4.56, x.AveragePrice.ValueOrZero())
	})

	t.Run("can fetch existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x := factory.CreateEveMarketPrice()
		// when
		got, err := st.GetEveMarketPrice(ctx, x.TypeID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, x, got)
	})
	t.Run("should return not found when price does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetEveMarketPrice(ctx, 42)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})

	t.Run("can list prices", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateEveMarketPrice()
		o2 := factory.CreateEveMarketPrice()
		// when
		got, err := st.ListEveMarketPrices(ctx)
		// then
		require.NoError(t, err)
		assert.ElementsMatch(t, []*app.EveMarketPrice{o1, o2}, got)
	})
}

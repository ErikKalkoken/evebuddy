package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestStorage_EveMarketPrices(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := st.UpdateOrCreateEveMarketPrice(ctx, storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: optional.New(1.23),
			AveragePrice:  optional.New(4.56),
		})
		// then
		require.NoError(t, err)
		x, err := st.GetEveMarketPrice(ctx, 42)
		require.NoError(t, err)
		xassert.EqualOptional(t, 1.23, x.AdjustedPrice)
		xassert.EqualOptional(t, 4.56, x.AveragePrice)
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
		err := st.UpdateOrCreateEveMarketPrice(ctx, storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: optional.New(1.23),
			AveragePrice:  optional.New(4.56),
		})
		// then
		require.NoError(t, err)
		x, err := st.GetEveMarketPrice(ctx, 42)
		require.NoError(t, err)
		xassert.EqualOptional(t, 1.23, x.AdjustedPrice)
		xassert.EqualOptional(t, 4.56, x.AveragePrice)
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

	t.Run("can list ids", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateEveMarketPrice()
		o2 := factory.CreateEveMarketPrice()
		// when
		got, err := st.ListEveMarketPriceIDs(ctx)
		// then
		require.NoError(t, err)
		want := set.Of(o1.TypeID, o2.TypeID)
		xassert.Equal(t, want, got)
	})

}

func TestStorage_ReplaceEveMarketPrices(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should create new and update existing prices", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: optional.New(4.0),
			AveragePrice:  optional.New(5.0),
		})
		// when
		changed, err := st.ReplaceEveMarketPrices(ctx, []storage.UpdateOrCreateEveMarketPriceParams{
			{TypeID: 42, AdjustedPrice: optional.New(1.23), AveragePrice: optional.New(4.56)},
			{TypeID: 43, AdjustedPrice: optional.New(7.0), AveragePrice: optional.New(8.0)},
		})
		// then
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](42, 43), changed)
		x1, err := st.GetEveMarketPrice(ctx, 42)
		require.NoError(t, err)
		xassert.EqualOptional(t, 1.23, x1.AdjustedPrice)
		xassert.EqualOptional(t, 4.56, x1.AveragePrice)
		x2, err := st.GetEveMarketPrice(ctx, 43)
		require.NoError(t, err)
		xassert.EqualOptional(t, 7.0, x2.AdjustedPrice)
		xassert.EqualOptional(t, 8.0, x2.AveragePrice)
	})
	t.Run("should delete prices missing from input", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateEveMarketPrice()
		o2 := factory.CreateEveMarketPrice()
		// when
		changed, err := st.ReplaceEveMarketPrices(ctx, []storage.UpdateOrCreateEveMarketPriceParams{
			{TypeID: o1.TypeID, AdjustedPrice: o1.AdjustedPrice, AveragePrice: o1.AveragePrice},
		})
		// then
		require.NoError(t, err)
		xassert.Equal(t, set.Of(o2.TypeID), changed)
		got, err := st.ListEveMarketPriceIDs(ctx)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(o1.TypeID), got)
	})
	t.Run("should delete prices missing from input with chunking", func(t *testing.T) {
		// given
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 1
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		testutil.MustTruncateTables(db)
		factory.CreateEveMarketPrice()
		factory.CreateEveMarketPrice()
		factory.CreateEveMarketPrice()
		// when
		_, err := st.ReplaceEveMarketPrices(ctx, nil)
		// then
		require.NoError(t, err)
		got, err := st.ListEveMarketPriceIDs(ctx)
		require.NoError(t, err)
		xassert.Equal(t, 0, got.Size())
	})
	t.Run("should do nothing when input and table are empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		changed, err := st.ReplaceEveMarketPrices(ctx, nil)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, changed.Size())
		got, err := st.ListEveMarketPriceIDs(ctx)
		require.NoError(t, err)
		xassert.Equal(t, 0, got.Size())
	})
	t.Run("should store nothing when a type ID is invalid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: optional.New(4.0),
			AveragePrice:  optional.New(5.0),
		})
		o2 := factory.CreateEveMarketPrice()
		// when
		_, err := st.ReplaceEveMarketPrices(ctx, []storage.UpdateOrCreateEveMarketPriceParams{
			{TypeID: 42, AdjustedPrice: optional.New(1.23), AveragePrice: optional.New(4.56)},
			{TypeID: 0, AdjustedPrice: optional.New(7.0), AveragePrice: optional.New(8.0)},
		})
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
		got, err := st.ListEveMarketPrices(ctx)
		require.NoError(t, err)
		assert.ElementsMatch(t, []*app.EveMarketPrice{o1, o2}, got)
	})
	t.Run("should not report or write unchanged prices", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: optional.New(4.0),
			AveragePrice:  optional.New(5.0),
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        43,
			AdjustedPrice: optional.New(6.0),
			AveragePrice:  optional.New(7.0),
		})
		// when
		changed, err := st.ReplaceEveMarketPrices(ctx, []storage.UpdateOrCreateEveMarketPriceParams{
			{TypeID: 42, AdjustedPrice: optional.New(4.0), AveragePrice: optional.New(5.0)},
			{TypeID: 43, AdjustedPrice: optional.New(6.0), AveragePrice: optional.New(7.0005)},
		})
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, changed.Size())
		x, err := st.GetEveMarketPrice(ctx, 43)
		require.NoError(t, err)
		xassert.EqualOptional(t, 7.0, x.AveragePrice)
	})
}

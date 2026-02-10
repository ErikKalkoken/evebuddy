package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveMarketPrice(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		arg := storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: optional.New(1.23),
			AveragePrice:  optional.New(4.56),
		}
		// when
		x, err := st.UpdateOrCreateEveMarketPrice(ctx, arg)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, int64(42), x.TypeID)
			xassert.Equal(t, 1.23, x.AdjustedPrice.ValueOrZero())
			xassert.Equal(t, 4.56, x.AveragePrice.ValueOrZero())
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: optional.New(4.0),
			AveragePrice:  optional.New(5.0),
		})
		arg := storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: optional.New(1.23),
			AveragePrice:  optional.New(4.56),
		}
		// when
		x, err := st.UpdateOrCreateEveMarketPrice(ctx, arg)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, int64(42), x.TypeID)
			xassert.Equal(t, 1.23, x.AdjustedPrice.ValueOrZero())
			xassert.Equal(t, 4.56, x.AveragePrice.ValueOrZero())
		}
	})
}

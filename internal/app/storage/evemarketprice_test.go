package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
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
			AdjustedPrice: 1.23,
			AveragePrice:  4.56,
		}
		// when
		x, err := st.UpdateOrCreateEveMarketPrice(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(42), x.TypeID)
			assert.Equal(t, 1.23, x.AdjustedPrice)
			assert.Equal(t, 4.56, x.AveragePrice)
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: 4,
			AveragePrice:  5,
		})
		arg := storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: 1.23,
			AveragePrice:  4.56,
		}
		// when
		x, err := st.UpdateOrCreateEveMarketPrice(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(42), x.TypeID)
			assert.Equal(t, 1.23, x.AdjustedPrice)
			assert.Equal(t, 4.56, x.AveragePrice)
		}
	})
}

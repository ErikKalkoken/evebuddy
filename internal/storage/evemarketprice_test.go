package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/ErikKalkoken/evebuddy/internal/storage/testutil"
)

func TestEveMarketPrice(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		arg := storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        42,
			AdjustedPrice: 1.23,
			AveragePrice:  4.56,
		}
		// when
		err := r.UpdateOrCreateEveMarketPrice(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := r.GetEveMarketPrice(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), o.TypeID)
				assert.Equal(t, 1.23, o.AdjustedPrice)
				assert.Equal(t, 4.56, o.AveragePrice)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		err := r.UpdateOrCreateEveMarketPrice(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := r.GetEveMarketPrice(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), o.TypeID)
				assert.Equal(t, 1.23, o.AdjustedPrice)
				assert.Equal(t, 4.56, o.AveragePrice)
			}
		}
	})
}

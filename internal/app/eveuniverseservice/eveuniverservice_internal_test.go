package eveuniverseservice

import (
	"context"
	"testing"

	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func NewTestService(st *storage.Storage) *EveUniverseService {
	s := New(Params{
		Storage:   st,
		ESIClient: goesi.NewAPIClient(nil, "test@kalkoken.net"),
	})
	return s
}

func TestUpdateEveMarketPricesESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewTestService(st)
	ctx := context.Background()
	t.Run("should create new objects from ESI", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/markets/prices/",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"adjusted_price": 306988.09,
					"average_price":  306292.67,
					"type_id":        32772,
				},
			}))
		// when
		err := s.updateMarketPricesESI(ctx)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetEveMarketPrice(ctx, 32772)
			if assert.NoError(t, err) {
				assert.Equal(t, 306988.09, o.AdjustedPrice)
				assert.Equal(t, 306292.67, o.AveragePrice)
			}
		}
	})
	t.Run("should update existing objects from ESI", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        32772,
			AdjustedPrice: 2,
			AveragePrice:  3,
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/markets/prices/",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"adjusted_price": 306988.09,
					"average_price":  306292.67,
					"type_id":        32772,
				},
			}))
		// when
		err := s.updateMarketPricesESI(ctx)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetEveMarketPrice(ctx, 32772)
			if assert.NoError(t, err) {
				assert.Equal(t, 306988.09, o.AdjustedPrice)
				assert.Equal(t, 306292.67, o.AveragePrice)
			}
		}
	})
}

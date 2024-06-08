package eveuniverse

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (eu *EveUniverseService) updateEveMarketPricesESI(ctx context.Context) error {
	prices, _, err := eu.esiClient.ESI.MarketApi.GetMarketsPrices(ctx, nil)
	if err != nil {
		return err
	}
	for _, p := range prices {
		arg := storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        p.TypeId,
			AdjustedPrice: p.AdjustedPrice,
			AveragePrice:  p.AveragePrice,
		}
		if err := eu.st.UpdateOrCreateEveMarketPrice(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

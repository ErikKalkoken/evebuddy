package eveuniverse

import (
	"context"
	"errors"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (eu *EveUniverseService) GetEveMarketPrice(ctx context.Context, typeID int32) (*model.EveMarketPrice, error) {
	o, err := eu.st.GetEveMarketPrice(ctx, typeID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return o, nil
}

// TODO: Change to bulk create

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
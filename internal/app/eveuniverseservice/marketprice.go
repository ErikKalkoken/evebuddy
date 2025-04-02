package eveuniverseservice

import (
	"context"
	"errors"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func (s *EveUniverseService) GetMarketPrice(ctx context.Context, typeID int32) (*app.EveMarketPrice, error) {
	o, err := s.st.GetEveMarketPrice(ctx, typeID)
	if errors.Is(err, app.ErrNotFound) {
		return nil, app.ErrNotFound
	}
	return o, err
}

// TODO: Change to bulk create

func (s *EveUniverseService) updateMarketPricesESI(ctx context.Context) error {
	prices, _, err := s.esiClient.ESI.MarketApi.GetMarketsPrices(ctx, nil)
	if err != nil {
		return err
	}
	for _, p := range prices {
		arg := storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        p.TypeId,
			AdjustedPrice: p.AdjustedPrice,
			AveragePrice:  p.AveragePrice,
		}
		if err := s.st.UpdateOrCreateEveMarketPrice(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func (st *Storage) GetEveMarketPrice(ctx context.Context, typeID int64) (*app.EveMarketPrice, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetEveMarketPrice %d: %w", typeID, err)
	}
	if typeID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	row, err := st.qRO.GetEveMarketPrice(ctx, typeID)
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	return eveMarketPriceFromDBModel(row), nil
}

func (st *Storage) ListEveMarketPrices(ctx context.Context) ([]*app.EveMarketPrice, error) {
	rows, err := st.qRO.ListEveMarketPrices(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListEveMarketPrices: %w", err)
	}
	oo := make([]*app.EveMarketPrice, 0)
	for _, r := range rows {
		oo = append(oo, eveMarketPriceFromDBModel(r))
	}
	return oo, nil
}

type UpdateOrCreateEveMarketPriceParams struct {
	TypeID        int64
	AdjustedPrice optional.Optional[float64]
	AveragePrice  optional.Optional[float64]
}

func (st *Storage) UpdateOrCreateEveMarketPrice(ctx context.Context, arg UpdateOrCreateEveMarketPriceParams) (*app.EveMarketPrice, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateEveMarketPrice %+v: %w", arg, err)
	}
	if arg.TypeID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRW.UpdateOrCreateEveMarketPrice(ctx, queries.UpdateOrCreateEveMarketPriceParams{
		TypeID:        arg.TypeID,
		AdjustedPrice: arg.AdjustedPrice.ValueOrZero(),
		AveragePrice:  arg.AveragePrice.ValueOrZero(),
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	return eveMarketPriceFromDBModel(r), nil
}

func eveMarketPriceFromDBModel(r queries.EveMarketPrice) *app.EveMarketPrice {
	o := &app.EveMarketPrice{
		TypeID:        r.TypeID,
		AdjustedPrice: optional.FromZeroValue(r.AdjustedPrice),
		AveragePrice:  optional.FromZeroValue(r.AveragePrice),
	}
	return o
}

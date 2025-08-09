package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) GetEveMarketPrice(ctx context.Context, typeID int32) (*app.EveMarketPrice, error) {
	row, err := st.qRO.GetEveMarketPrice(ctx, int64(typeID))
	if err != nil {
		return nil, fmt.Errorf("get eve market price for type %d: %w", typeID, convertGetError(err))
	}
	t2 := eveMarketPriceFromDBModel(row)
	return t2, nil
}

type UpdateOrCreateEveMarketPriceParams struct {
	TypeID        int32
	AdjustedPrice float64
	AveragePrice  float64
}

func (st *Storage) UpdateOrCreateEveMarketPrice(ctx context.Context, arg UpdateOrCreateEveMarketPriceParams) (*app.EveMarketPrice, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateEveMarketPrice %+v: %w", arg, err)
	}
	if arg.TypeID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRW.UpdateOrCreateEveMarketPrice(ctx, queries.UpdateOrCreateEveMarketPriceParams{
		TypeID:        int64(arg.TypeID),
		AdjustedPrice: arg.AdjustedPrice,
		AveragePrice:  arg.AveragePrice,
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	return eveMarketPriceFromDBModel(r), nil
}

func eveMarketPriceFromDBModel(o queries.EveMarketPrice) *app.EveMarketPrice {
	if o.TypeID == 0 {
		panic("missing type ID")
	}
	return &app.EveMarketPrice{
		TypeID:        int32(o.TypeID),
		AdjustedPrice: o.AdjustedPrice,
		AveragePrice:  o.AveragePrice,
	}
}

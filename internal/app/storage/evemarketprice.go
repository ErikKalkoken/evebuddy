package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) GetEveMarketPrice(ctx context.Context, typeID int32) (*app.EveMarketPrice, error) {
	row, err := st.qRO.GetEveMarketPrice(ctx, int64(typeID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("get eve market price for type %d: %w", typeID, err)
	}
	t2 := eveMarketPriceFromDBModel(row)
	return t2, nil
}

type UpdateOrCreateEveMarketPriceParams struct {
	TypeID        int32
	AdjustedPrice float64
	AveragePrice  float64
}

func (st *Storage) UpdateOrCreateEveMarketPrice(ctx context.Context, arg UpdateOrCreateEveMarketPriceParams) error {
	arg2 := queries.UpdateOrCreateEveMarketPriceParams{
		TypeID:        int64(arg.TypeID),
		AdjustedPrice: arg.AdjustedPrice,
		AveragePrice:  arg.AveragePrice,
	}
	if err := st.qRW.UpdateOrCreateEveMarketPrice(ctx, arg2); err != nil {
		return fmt.Errorf("update or create eve market price %v: %w", arg, err)
	}
	return nil
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

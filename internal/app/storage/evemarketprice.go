package storage

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xmaps"
)

func (st *Storage) deleteEveMarketPrices(ctx context.Context, q *queries.Queries, typeIDs set.Set[int64]) error {
	for ids := range slices.Chunk(slices.Collect(typeIDs.All()), st.MaxIDsPerQuery) {
		if err := q.DeleteEveMarketPrices(ctx, ids); err != nil {
			return fmt.Errorf("delete eve market prices for %d ids: %w", len(ids), err)
		}
	}
	return nil
}

// ReplaceEveMarketPrices replaces all stored prices with prices
// and returns the type IDs of created, changed and deleted prices.
func (st *Storage) ReplaceEveMarketPrices(ctx context.Context, prices []UpdateOrCreateEveMarketPriceParams) (set.Set[int64], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceEveMarketPrices: %d prices: %w", len(prices), err)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	rows, err := qtx.ListEveMarketPrices(ctx)
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	current := make(map[int64]app.EveMarketPrice, len(rows))
	for _, r := range rows {
		current[r.TypeID] = *eveMarketPriceFromDBModel(r)
	}
	var changed set.Set[int64]
	for _, arg := range prices {
		o, found := xmaps.Pop(current, arg.TypeID) // what remains in current is obsolete
		if found && o.Equal(app.EveMarketPrice{
			TypeID:        arg.TypeID,
			AdjustedPrice: arg.AdjustedPrice,
			AveragePrice:  arg.AveragePrice,
		}) {
			continue
		}
		if err := updateOrCreateEveMarketPrice(ctx, qtx, arg); err != nil {
			return set.Set[int64]{}, wrapErr(err)
		}
		changed.Add(arg.TypeID)
	}
	obsolete := set.Collect(maps.Keys(current))
	if err := st.deleteEveMarketPrices(ctx, qtx, obsolete); err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	if err := tx.Commit(); err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	if obsolete.Size() > 0 {
		slog.Info("Market prices deleted", "count", obsolete.Size())
	}
	changed.AddSeq(obsolete.All())
	return changed, nil
}

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
	oo := make([]*app.EveMarketPrice, len(rows))
	for i, r := range rows {
		oo[i] = eveMarketPriceFromDBModel(r)
	}
	return oo, nil
}

func (st *Storage) ListEveMarketPriceIDs(ctx context.Context) (set.Set[int64], error) {
	ids, err := st.qRO.ListEveMarketPriceIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("ListEveMarketPriceIDs: %w", err)
	}
	ids2 := set.Collect(slices.Values(ids))
	return ids2, nil
}

type UpdateOrCreateEveMarketPriceParams struct {
	TypeID        int64
	AdjustedPrice optional.Optional[float64]
	AveragePrice  optional.Optional[float64]
}

func (st *Storage) UpdateOrCreateEveMarketPrice(ctx context.Context, arg UpdateOrCreateEveMarketPriceParams) error {
	return updateOrCreateEveMarketPrice(ctx, st.qRW, arg)
}

func updateOrCreateEveMarketPrice(ctx context.Context, q *queries.Queries, arg UpdateOrCreateEveMarketPriceParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateEveMarketPrice %+v: %w", arg, err)
	}
	if arg.TypeID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := q.UpdateOrCreateEveMarketPrice(ctx, queries.UpdateOrCreateEveMarketPriceParams{
		TypeID:        arg.TypeID,
		AdjustedPrice: arg.AdjustedPrice.ValueOrZero(),
		AveragePrice:  arg.AveragePrice.ValueOrZero(),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func eveMarketPriceFromDBModel(r queries.EveMarketPrice) *app.EveMarketPrice {
	o := &app.EveMarketPrice{
		TypeID:        r.TypeID,
		AdjustedPrice: optional.FromZeroValue(r.AdjustedPrice),
		AveragePrice:  optional.FromZeroValue(r.AveragePrice),
	}
	return o
}

package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateEveRegionParams struct {
	Description optional.Optional[string]
	ID          int64
	Name        string
}

func (st *Storage) CreateEveRegion(ctx context.Context, arg CreateEveRegionParams) (*app.EveRegion, error) {
	if arg.ID == 0 {
		return nil, fmt.Errorf("CreateEveRegion for id %d: %w", arg.ID, app.ErrInvalid)
	}
	arg2 := queries.CreateEveRegionParams{
		ID:          arg.ID,
		Description: arg.Description.ValueOrZero(),
		Name:        arg.Name,
	}
	e, err := st.qRW.CreateEveRegion(ctx, arg2)
	if err != nil {
		return nil, fmt.Errorf("CreateEveRegion for id %d: %w", arg.ID, err)
	}
	return eveRegionFromDBModel(e), nil
}

func (st *Storage) GetEveRegion(ctx context.Context, id int64) (*app.EveRegion, error) {
	c, err := st.qRO.GetEveRegion(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get EveRegion for id %d: %w", id, convertGetError(err))
	}
	return eveRegionFromDBModel(c), nil
}

func eveRegionFromDBModel(c queries.EveRegion) *app.EveRegion {
	return &app.EveRegion{
		ID:          c.ID,
		Description: optional.FromZeroValue(c.Description),
		Name:        c.Name,
	}
}

func (st *Storage) ListEveRegionIDs(ctx context.Context) (set.Set[int64], error) {
	ids, err := st.qRO.ListEveRegionIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("ListEveRegionIDs: %w", err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) MissingEveRegions(ctx context.Context, ids set.Set[int64]) (set.Set[int64], error) {
	currentIDs, err := st.qRO.ListEveRegionIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, err
	}
	current := set.Of(currentIDs...)
	missing := set.Difference(ids, current)
	return missing, nil
}

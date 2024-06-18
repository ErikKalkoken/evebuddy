package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type UpdateOrCreateLocationParams struct {
	ID               int64
	EveSolarSystemID optional.Optional[int32]
	EveTypeID        optional.Optional[int32]
	Name             string
	OwnerID          optional.Optional[int32]
	UpdatedAt        time.Time
}

func (st *Storage) UpdateOrCreateEveLocation(ctx context.Context, arg UpdateOrCreateLocationParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("invalid ID for eve location %d", arg.ID)
	}
	arg2 := queries.UpdateOrCreateLocationParams{
		ID:               int64(arg.ID),
		EveSolarSystemID: optional.ToNullInt64(arg.EveSolarSystemID),
		EveTypeID:        optional.ToNullInt64(arg.EveTypeID),
		Name:             arg.Name,
		OwnerID:          optional.ToNullInt64(arg.OwnerID),
		UpdatedAt:        arg.UpdatedAt,
	}
	if err := st.q.UpdateOrCreateLocation(ctx, arg2); err != nil {
		return fmt.Errorf("failed to update or create eve location %v, %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEveLocation(ctx context.Context, id int64) (*app.EveLocation, error) {
	o, err := st.q.GetLocation(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get eve location for id %d: %w", id, err)
	}
	x, err := st.eveLocationFromDBModel(ctx, o)
	if err != nil {
		return nil, err
	}
	return x, nil
}

func (st *Storage) ListEveLocation(ctx context.Context) ([]*app.EveLocation, error) {
	rows, err := st.q.ListEveLocations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list eve locations: %w", err)
	}
	oo := make([]*app.EveLocation, len(rows))
	for i, r := range rows {
		o, err := st.eveLocationFromDBModel(ctx, r)
		if err != nil {
			return nil, err
		}
		oo[i] = o
	}
	return oo, nil
}

func (st *Storage) MissingEveLocations(ctx context.Context, ids []int64) ([]int64, error) {
	currentIDs, err := st.q.ListLocationIDs(ctx)
	if err != nil {
		return nil, err
	}
	current := set.NewFromSlice(currentIDs)
	incoming := set.NewFromSlice(ids)
	missing := incoming.Difference(current)
	return missing.ToSlice(), nil
}

func (st *Storage) eveLocationFromDBModel(ctx context.Context, l queries.EveLocation) (*app.EveLocation, error) {
	l2 := &app.EveLocation{
		ID:        l.ID,
		Name:      l.Name,
		UpdatedAt: l.UpdatedAt,
	}
	if l.EveTypeID.Valid {
		x, err := st.GetEveType(ctx, int32(l.EveTypeID.Int64))
		if err != nil {
			return nil, err
		}
		l2.Type = x
	}
	if l.EveSolarSystemID.Valid {
		x, err := st.GetEveSolarSystem(ctx, int32(l.EveSolarSystemID.Int64))
		if err != nil {
			return nil, err
		}
		l2.SolarSystem = x
	}
	if l.OwnerID.Valid {
		x, err := st.GetEveEntity(ctx, int32(l.OwnerID.Int64))
		if err != nil {
			return nil, err
		}
		l2.Owner = x
	}
	return l2, nil
}

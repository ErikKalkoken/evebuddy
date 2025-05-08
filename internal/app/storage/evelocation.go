package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
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
		return fmt.Errorf("UpdateOrCreateEveLocation: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.UpdateOrCreateLocationParams{
		ID:               int64(arg.ID),
		EveSolarSystemID: optional.ToNullInt64(arg.EveSolarSystemID),
		EveTypeID:        optional.ToNullInt64(arg.EveTypeID),
		Name:             arg.Name,
		OwnerID:          optional.ToNullInt64(arg.OwnerID),
		UpdatedAt:        arg.UpdatedAt,
	}
	if err := st.qRW.UpdateOrCreateLocation(ctx, arg2); err != nil {
		return fmt.Errorf("update or create eve location %v, %w", arg, err)
	}
	return nil
}

func (st *Storage) GetLocation(ctx context.Context, id int64) (*app.EveLocation, error) {
	o, err := st.qRO.GetLocation(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = app.ErrNotFound
		}
		return nil, fmt.Errorf("get eve location for id %d: %w", id, err)
	}
	x, err := st.eveLocationFromDBModel(ctx, o)
	if err != nil {
		return nil, err
	}
	return x, nil
}

func (st *Storage) ListEveLocation(ctx context.Context) ([]*app.EveLocation, error) {
	rows, err := st.qRO.ListEveLocations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list eve locations: %w", err)
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

func (st *Storage) ListEveLocationInSolarSystem(ctx context.Context, solarSystemID int32) ([]*app.EveLocation, error) {
	rows, err := st.qRO.ListEveLocationsInSolarSystem(ctx, sql.NullInt64{Int64: int64(solarSystemID), Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list eve locations in solar system: %w", err)
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

func (st *Storage) MissingEveLocations(ctx context.Context, ids set.Set[int64]) (set.Set[int64], error) {
	currentIDs, err := st.qRO.ListLocationIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, err
	}
	current := set.Of(currentIDs...)
	missing := set.Difference(ids, current)
	return missing, nil
}

// TODO: Refactor for better performance
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

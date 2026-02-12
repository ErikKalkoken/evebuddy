package storage

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type UpdateOrCreateLocationParams struct {
	ID            int64
	Name          string
	OwnerID       optional.Optional[int64]
	SolarSystemID optional.Optional[int64]
	TypeID        optional.Optional[int64]
	UpdatedAt     time.Time
}

func (st *Storage) UpdateOrCreateEveLocation(ctx context.Context, arg UpdateOrCreateLocationParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("UpdateOrCreateEveLocation: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.UpdateOrCreateEveLocationParams{
		ID:               arg.ID,
		EveSolarSystemID: optional.ToNullInt64(arg.SolarSystemID),
		EveTypeID:        optional.ToNullInt64(arg.TypeID),
		Name:             arg.Name,
		OwnerID:          optional.ToNullInt64(arg.OwnerID),
		UpdatedAt:        arg.UpdatedAt,
	}
	if err := st.qRW.UpdateOrCreateEveLocation(ctx, arg2); err != nil {
		return fmt.Errorf("update or create eve location %+v, %w", arg, err)
	}
	return nil
}

func (st *Storage) GetLocation(ctx context.Context, id int64) (*app.EveLocation, error) {
	o, err := st.qRO.GetEveLocation(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get eve location for id %d: %w", id, convertGetError(err))
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

func (st *Storage) ListEveLocationIDs(ctx context.Context) (set.Set[int64], error) {
	ids, err := st.qRO.ListEveLocationIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list eve locations: %w", err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListEveLocationInSolarSystem(ctx context.Context, solarSystemID int64) ([]*app.EveLocation, error) {
	rows, err := st.qRO.ListEveLocationsInSolarSystem(ctx, sql.NullInt64{Int64: solarSystemID, Valid: true})
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

// MissingEveLocations returns which ids for eve locations are missing.
func (st *Storage) MissingEveLocations(ctx context.Context, ids set.Set[int64]) (set.Set[int64], error) {
	currentIDs, err := st.qRO.ListEveLocationIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, err
	}
	current := set.Collect(slices.Values(currentIDs))
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
		o, err := st.GetEveType(ctx, l.EveTypeID.Int64)
		if err != nil {
			return nil, err
		}
		l2.Type = optional.New(o)
	}
	if l.EveSolarSystemID.Valid {
		o, err := st.GetEveSolarSystem(ctx, l.EveSolarSystemID.Int64)
		if err != nil {
			return nil, err
		}
		l2.SolarSystem = optional.New(o)
	}
	if l.OwnerID.Valid {
		o, err := st.GetEveEntity(ctx, l.OwnerID.Int64)
		if err != nil {
			return nil, err
		}
		l2.Owner = optional.New(o)
	}
	return l2, nil
}

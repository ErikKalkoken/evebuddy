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
	oo, err := st.eveLocationsFromDBModels(ctx, []queries.EveLocation{o})
	if err != nil {
		return nil, err
	}
	return oo[0], nil
}

func (st *Storage) ListEveLocation(ctx context.Context) ([]*app.EveLocation, error) {
	rows, err := st.qRO.ListEveLocations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list eve locations: %w", err)
	}
	return st.eveLocationsFromDBModels(ctx, rows)
}

func (st *Storage) ListEveLocationIDs(ctx context.Context) (set.Set[int64], error) {
	ids, err := st.qRO.ListEveLocationIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list eve locations: %w", err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListEveLocationInSolarSystem(ctx context.Context, solarSystemID int64) ([]*app.EveLocation, error) {
	rows, err := st.qRO.ListEveLocationsInSolarSystem(ctx, sql.NullInt64{Int64: solarSystemID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list eve locations in solar system: %w", err)
	}
	return st.eveLocationsFromDBModels(ctx, rows)
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

// eveLocationsFromDBModels converts rows to locations, batch loading related objects.
func (st *Storage) eveLocationsFromDBModels(ctx context.Context, rows []queries.EveLocation) ([]*app.EveLocation, error) {
	var typeIDs, solarSystemIDs, ownerIDs set.Set[int64]
	for _, r := range rows {
		if r.EveTypeID.Valid {
			typeIDs.Add(r.EveTypeID.Int64)
		}
		if r.EveSolarSystemID.Valid {
			solarSystemIDs.Add(r.EveSolarSystemID.Int64)
		}
		if r.OwnerID.Valid {
			ownerIDs.Add(r.OwnerID.Int64)
		}
	}
	types, err := st.ListEveTypesForIDs(ctx, slices.Collect(typeIDs.All()))
	if err != nil {
		return nil, err
	}
	solarSystems, err := st.ListEveSolarSystemsForIDs(ctx, slices.Collect(solarSystemIDs.All()))
	if err != nil {
		return nil, err
	}
	owners, err := st.ListEveEntitiesForIDs(ctx, slices.Collect(ownerIDs.All()))
	if err != nil {
		return nil, err
	}
	typeMap := make(map[int64]*app.EveType, len(types))
	for _, o := range types {
		typeMap[o.ID] = o
	}
	solarSystemMap := make(map[int64]*app.EveSolarSystem, len(solarSystems))
	for _, o := range solarSystems {
		solarSystemMap[o.ID] = o
	}
	ownerMap := make(map[int64]*app.EveEntity, len(owners))
	for _, o := range owners {
		ownerMap[o.ID] = o
	}
	oo := make([]*app.EveLocation, len(rows))
	for i, r := range rows {
		o := &app.EveLocation{
			ID:        r.ID,
			Name:      r.Name,
			UpdatedAt: r.UpdatedAt,
		}
		if r.EveTypeID.Valid {
			o.Type = optional.New(typeMap[r.EveTypeID.Int64])
		}
		if r.EveSolarSystemID.Valid {
			o.SolarSystem = optional.New(solarSystemMap[r.EveSolarSystemID.Int64])
		}
		if r.OwnerID.Valid {
			o.Owner = optional.New(ownerMap[r.OwnerID.Int64])
		}
		oo[i] = o
	}
	return oo, nil
}

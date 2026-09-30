package storage

import (
	"context"
	"fmt"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveSolarSystemParams struct {
	ConstellationID int64
	ID              int64
	Name            string
	SecurityStatus  float64
}

func (st *Storage) CreateEveSolarSystem(ctx context.Context, arg CreateEveSolarSystemParams) error {
	if arg.ID == 0 || arg.ConstellationID == 0 {
		return fmt.Errorf("CreateEveSolarSystem: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateEveSolarSystemParams{
		ID:                 arg.ID,
		EveConstellationID: arg.ConstellationID,
		Name:               arg.Name,
		SecurityStatus:     arg.SecurityStatus,
	}
	err := st.qRW.CreateEveSolarSystem(ctx, arg2)
	if err != nil {
		return fmt.Errorf("create EveSolarSystem %+v, %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEveSolarSystem(ctx context.Context, id int64) (*app.EveSolarSystem, error) {
	row, err := st.qRO.GetEveSolarSystem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get EveSolarSystem for id %d: %w", id, convertGetError(err))
	}
	t := eveSolarSystemFromDBModel(row.EveSolarSystem, row.EveConstellation, row.EveRegion)
	return t, nil
}

// ListEveSolarSystemsForIDs returns a slice of EveSolarSystems in the same order as ids.
//
// Returns an error if at least one object can not be found.
func (st *Storage) ListEveSolarSystemsForIDs(ctx context.Context, ids []int64) ([]*app.EveSolarSystem, error) {
	if len(ids) == 0 {
		return []*app.EveSolarSystem{}, nil
	}
	m := make(map[int64]*app.EveSolarSystem)
	for idsChunk := range slices.Chunk(ids, st.MaxIDsPerQuery) {
		rows, err := st.qRO.ListEveSolarSystemsForIDs(ctx, idsChunk)
		if err != nil {
			return nil, fmt.Errorf("list eve solar systems for %d ids: %w", len(idsChunk), err)
		}
		for _, r := range rows {
			m[r.EveSolarSystem.ID] = eveSolarSystemFromDBModel(r.EveSolarSystem, r.EveConstellation, r.EveRegion)
		}
	}
	oo := make([]*app.EveSolarSystem, 0, len(ids))
	for _, id := range ids {
		o, found := m[id]
		if !found {
			return nil, fmt.Errorf("list eve solar systems: id %d: %w", id, app.ErrNotFound)
		}
		oo = append(oo, o)
	}
	return oo, nil
}

func eveSolarSystemFromDBModel(s queries.EveSolarSystem, c queries.EveConstellation, r queries.EveRegion) *app.EveSolarSystem {
	return &app.EveSolarSystem{
		Constellation:  eveConstellationFromDBModel(c, r),
		ID:             s.ID,
		Name:           s.Name,
		SecurityStatus: float32(s.SecurityStatus),
	}
}

func (st *Storage) ListEveSolarSystemIDs(ctx context.Context) (set.Set[int64], error) {
	ids, err := st.qRO.ListEveSolarSystemIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("ListEveSolarSystemIDs: %w", err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) MissingEveSolarSystems(ctx context.Context, ids set.Set[int64]) (set.Set[int64], error) {
	currentIDs, err := st.qRO.ListEveSolarSystemIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, err
	}
	current := set.Collect(slices.Values(currentIDs))
	missing := set.Difference(ids, current)
	return missing, nil
}

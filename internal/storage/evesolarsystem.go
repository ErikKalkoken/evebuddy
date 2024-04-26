package storage

import (
	"context"
	"database/sql"
	"errors"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage/queries"
	"fmt"
)

func (r *Storage) CreateEveSolarSystem(ctx context.Context, id int32, eve_constellation_id int32, name string, security_status float64) error {
	if id == 0 {
		return fmt.Errorf("invalid ID %d", id)
	}
	arg := queries.CreateEveSolarSystemParams{
		ID:                 int64(id),
		EveConstellationID: int64(eve_constellation_id),
		Name:               name,
		SecurityStatus:     float64(security_status),
	}
	err := r.q.CreateEveSolarSystem(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to create EveSolarSystem %v, %w", arg, err)
	}
	return nil
}

func (r *Storage) GetEveSolarSystem(ctx context.Context, id int32) (*model.EveSolarSystem, error) {
	row, err := r.q.GetEveSolarSystem(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get EveSolarSystem for id %d: %w", id, err)
	}
	t := eveSolarSystemFromDBModel(row.EveSolarSystem, row.EveConstellation, row.EveRegion)
	return t, nil
}

func eveSolarSystemFromDBModel(s queries.EveSolarSystem, c queries.EveConstellation, r queries.EveRegion) *model.EveSolarSystem {
	return &model.EveSolarSystem{
		Constellation:  eveConstellationFromDBModel(c, r),
		ID:             int32(s.ID),
		Name:           s.Name,
		SecurityStatus: s.SecurityStatus,
	}
}

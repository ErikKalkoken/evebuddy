package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateEveSolarSystemParams struct {
	ConstellationID int32
	ID              int32
	Name            string
	SecurityStatus  float32
}

func (st *Storage) CreateEveSolarSystem(ctx context.Context, arg CreateEveSolarSystemParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("invalid ID %d", arg.ID)
	}
	arg2 := queries.CreateEveSolarSystemParams{
		ID:                 int64(arg.ID),
		EveConstellationID: int64(arg.ConstellationID),
		Name:               arg.Name,
		SecurityStatus:     float64(arg.SecurityStatus),
	}
	err := st.q.CreateEveSolarSystem(ctx, arg2)
	if err != nil {
		return fmt.Errorf("failed to create EveSolarSystem %v, %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEveSolarSystem(ctx context.Context, id int32) (*app.EveSolarSystem, error) {
	row, err := st.q.GetEveSolarSystem(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get EveSolarSystem for id %d: %w", id, err)
	}
	t := eveSolarSystemFromDBModel(row.EveSolarSystem, row.EveConstellation, row.EveRegion)
	return t, nil
}

func eveSolarSystemFromDBModel(s queries.EveSolarSystem, c queries.EveConstellation, r queries.EveRegion) *app.EveSolarSystem {
	return &app.EveSolarSystem{
		Constellation:  eveConstellationFromDBModel(c, r),
		ID:             int32(s.ID),
		Name:           s.Name,
		SecurityStatus: float32(s.SecurityStatus),
	}
}

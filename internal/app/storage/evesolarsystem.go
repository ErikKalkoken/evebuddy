package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveSolarSystemParams struct {
	ConstellationID int32
	ID              int32
	Name            string
	SecurityStatus  float32
}

func (st *Storage) CreateEveSolarSystem(ctx context.Context, arg CreateEveSolarSystemParams) error {
	if arg.ID == 0 || arg.ConstellationID == 0 {
		return fmt.Errorf("CreateEveSolarSystem: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateEveSolarSystemParams{
		ID:                 int64(arg.ID),
		EveConstellationID: int64(arg.ConstellationID),
		Name:               arg.Name,
		SecurityStatus:     float64(arg.SecurityStatus),
	}
	err := st.qRW.CreateEveSolarSystem(ctx, arg2)
	if err != nil {
		return fmt.Errorf("create EveSolarSystem %v, %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEveSolarSystem(ctx context.Context, id int32) (*app.EveSolarSystem, error) {
	row, err := st.qRO.GetEveSolarSystem(ctx, int64(id))
	if err != nil {
		return nil, fmt.Errorf("get EveSolarSystem for id %d: %w", id, convertGetError(err))
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

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveMoonParams struct {
	ID            int32
	Name          string
	SolarSystemID int32
}

func (st *Storage) CreateEveMoon(ctx context.Context, arg CreateEveMoonParams) error {
	if arg.ID == 0 || arg.SolarSystemID == 0 {
		return fmt.Errorf("invalid IDs %v", arg)
	}
	arg2 := queries.CreateEveMoonParams{
		ID:               int64(arg.ID),
		Name:             arg.Name,
		EveSolarSystemID: int64(arg.SolarSystemID),
	}
	err := st.q.CreateEveMoon(ctx, arg2)
	if err != nil {
		return fmt.Errorf("failed to create EveMoon %v, %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEveMoon(ctx context.Context, id int32) (*app.EveMoon, error) {
	row, err := st.q.GetEveMoon(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get EveMoon for id %d: %w", id, err)
	}
	g := EveMoonFromDBModel(
		row.EveMoon,
		eveSolarSystemFromDBModel(
			row.EveSolarSystem,
			row.EveConstellation,
			row.EveRegion,
		))
	return g, nil
}

func EveMoonFromDBModel(p queries.EveMoon, ess *app.EveSolarSystem) *app.EveMoon {
	return &app.EveMoon{
		ID:          int32(p.ID),
		Name:        p.Name,
		SolarSystem: ess,
	}
}
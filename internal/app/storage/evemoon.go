package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveMoonParams struct {
	ID            int64
	Name          string
	SolarSystemID int64
}

func (st *Storage) CreateEveMoon(ctx context.Context, arg CreateEveMoonParams) error {
	if arg.ID == 0 || arg.SolarSystemID == 0 {
		return fmt.Errorf("CreateEveMoon: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateEveMoonParams{
		ID:              arg.ID,
		Name:             arg.Name,
		EveSolarSystemID:arg.SolarSystemID,
	}
	err := st.qRW.CreateEveMoon(ctx, arg2)
	if err != nil {
		return fmt.Errorf("CreateEveMoon: %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEveMoon(ctx context.Context, id int64) (*app.EveMoon, error) {
	row, err := st.qRO.GetEveMoon(ctx,id)
	if err != nil {
		return nil, fmt.Errorf("get EveMoon for id %d: %w", id, convertGetError(err))
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
		ID:         p.ID,
		Name:        p.Name,
		SolarSystem: ess,
	}
}

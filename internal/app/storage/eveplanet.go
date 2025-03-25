package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEvePlanetParams struct {
	ID            int32
	Name          string
	SolarSystemID int32
	TypeID        int32
}

func (st *Storage) CreateEvePlanet(ctx context.Context, arg CreateEvePlanetParams) error {
	if arg.ID == 0 || arg.SolarSystemID == 0 || arg.TypeID == 0 {
		return fmt.Errorf("invalid IDs %v", arg)
	}
	arg2 := queries.CreateEvePlanetParams{
		ID:               int64(arg.ID),
		Name:             arg.Name,
		EveSolarSystemID: int64(arg.SolarSystemID),
		EveTypeID:        int64(arg.TypeID),
	}
	err := st.qRW.CreateEvePlanet(ctx, arg2)
	if err != nil {
		return fmt.Errorf("create EvePlanet %v, %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEvePlanet(ctx context.Context, id int32) (*app.EvePlanet, error) {
	row, err := st.qRO.GetEvePlanet(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("get EvePlanet for id %d: %w", id, err)
	}
	g := evePlanetFromDBModel(
		row.EvePlanet,
		eveSolarSystemFromDBModel(row.EveSolarSystem, row.EveConstellation, row.EveRegion),
		eveTypeFromDBModel(row.EveType, row.EveGroup, row.EveCategory),
	)
	return g, nil
}

func evePlanetFromDBModel(p queries.EvePlanet, ess *app.EveSolarSystem, et *app.EveType) *app.EvePlanet {
	return &app.EvePlanet{
		ID:          int32(p.ID),
		Name:        p.Name,
		SolarSystem: ess,
		Type:        et,
	}
}

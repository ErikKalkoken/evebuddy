package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveSchematicParams struct {
	ID        int64
	CycleTime int64
	Name      string
}

func (st *Storage) CreateEveSchematic(ctx context.Context, arg CreateEveSchematicParams) (*app.EveSchematic, error) {
	if arg.ID == 0 {
		return nil, fmt.Errorf("invalid EveSchematic ID %d", arg.ID)
	}
	o, err := st.qRW.CreateEveSchematic(ctx, queries.CreateEveSchematicParams{
		ID:        arg.ID,
		CycleTime: arg.CycleTime,
		Name:      arg.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("create EveSchematic %+v, %w", arg, err)
	}
	return eveSchematicFromDBModel(o), nil
}

func (st *Storage) GetEveSchematic(ctx context.Context, id int64) (*app.EveSchematic, error) {
	c, err := st.qRO.GetEveSchematic(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get EveSchematic for id %d: %w", id, convertGetError(err))
	}
	return eveSchematicFromDBModel(c), nil
}

func eveSchematicFromDBModel(o queries.EveSchematic) *app.EveSchematic {
	return &app.EveSchematic{
		ID:        o.ID,
		Name:      o.Name,
		CycleTime: int(o.CycleTime),
	}
}

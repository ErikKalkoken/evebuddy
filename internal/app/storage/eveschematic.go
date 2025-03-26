package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveSchematicParams struct {
	ID        int32
	CycleTime int
	Name      string
}

func (st *Storage) CreateEveSchematic(ctx context.Context, arg CreateEveSchematicParams) (*app.EveSchematic, error) {
	if arg.ID == 0 {
		return nil, fmt.Errorf("invalid EveSchematic ID %d", arg.ID)
	}
	arg2 := queries.CreateEveSchematicParams{
		ID:        int64(arg.ID),
		CycleTime: int64(arg.CycleTime),
		Name:      arg.Name,
	}
	e, err := st.qRW.CreateEveSchematic(ctx, arg2)
	if err != nil {
		return nil, fmt.Errorf("create EveSchematic %v, %w", arg, err)
	}
	return eveSchematicFromDBModel(e), nil
}

func (st *Storage) GetEveSchematic(ctx context.Context, id int32) (*app.EveSchematic, error) {
	c, err := st.qRO.GetEveSchematic(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = app.ErrNotFound
		}
		return nil, fmt.Errorf("get EveSchematic for id %d: %w", id, err)
	}
	return eveSchematicFromDBModel(c), nil
}

func eveSchematicFromDBModel(o queries.EveSchematic) *app.EveSchematic {
	return &app.EveSchematic{
		ID:        int32(o.ID),
		Name:      o.Name,
		CycleTime: int(o.CycleTime),
	}
}

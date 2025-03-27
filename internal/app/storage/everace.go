package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveRaceParams struct {
	ID          int32
	Name        string
	Description string
}

func (st *Storage) CreateEveRace(ctx context.Context, arg CreateEveRaceParams) (*app.EveRace, error) {
	if arg.ID == 0 {
		return nil, fmt.Errorf("CreateEveRace: %+v, %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateEveRaceParams{
		ID:          int64(arg.ID),
		Description: arg.Description,
		Name:        arg.Name,
	}
	o, err := st.qRW.CreateEveRace(ctx, arg2)
	if err != nil {
		return nil, fmt.Errorf("CreateEveRace: %+v, %w", arg, err)
	}
	return eveRaceFromDBModel(o), nil
}

func (st *Storage) GetEveRace(ctx context.Context, id int32) (*app.EveRace, error) {
	o, err := st.qRO.GetEveRace(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = app.ErrNotFound
		}
		return nil, fmt.Errorf("get Race for id %d: %w", id, err)
	}
	return eveRaceFromDBModel(o), nil
}

func eveRaceFromDBModel(er queries.EveRace) *app.EveRace {
	if er.ID == 0 {
		return nil
	}
	return &app.EveRace{
		Description: er.Description,
		ID:          int32(er.ID),
		Name:        er.Name,
	}
}

package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveRaceParams struct {
	ID          int64
	Name        string
	Description string
}

func (st *Storage) CreateEveRace(ctx context.Context, arg CreateEveRaceParams) (*app.EveRace, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateEveRace: %+v, %w", arg, err)
	}
	if arg.ID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	o, err := st.qRW.CreateEveRace(ctx, queries.CreateEveRaceParams{
		ID:          arg.ID,
		Description: arg.Description,
		Name:        arg.Name,
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	return eveRaceFromDBModel(o), nil
}

func (st *Storage) GetEveRace(ctx context.Context, id int64) (*app.EveRace, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetEveRace: %d, %w", id, err)
	}
	if id == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	o, err := st.qRO.GetEveRace(ctx, id)
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	return eveRaceFromDBModel(o), nil
}

func eveRaceFromDBModel(er queries.EveRace) *app.EveRace {
	return &app.EveRace{
		Description: er.Description,
		ID:          er.ID,
		Name:        er.Name,
	}
}

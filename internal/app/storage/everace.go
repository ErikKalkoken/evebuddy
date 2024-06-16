package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) CreateEveRace(ctx context.Context, id int32, description, name string) (*app.EveRace, error) {
	arg := queries.CreateEveRaceParams{
		ID:          int64(id),
		Description: description,
		Name:        name,
	}
	o, err := st.q.CreateEveRace(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to create race %d: %w", id, err)
	}
	return eveRaceFromDBModel(o), nil
}

func (st *Storage) GetEveRace(ctx context.Context, id int32) (*app.EveRace, error) {
	o, err := st.q.GetEveRace(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get Race for id %d: %w", id, err)
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

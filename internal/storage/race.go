package storage

import (
	"context"
	"database/sql"
	"errors"
	islices "example/evebuddy/internal/helper/slices"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage/sqlc"
	"fmt"
)

func (r *Storage) CreateRace(ctx context.Context, id int32, description, name string) (model.Race, error) {
	arg := sqlc.CreateRaceParams{
		ID:          int64(id),
		Description: description,
		Name:        name,
	}
	o, err := r.q.CreateRace(ctx, arg)
	if err != nil {
		return model.Race{}, fmt.Errorf("failed to create race %d: %w", id, err)
	}
	return raceFromDBModel(o), nil
}

func (r *Storage) GetRace(ctx context.Context, id int32) (model.Race, error) {
	o, err := r.q.GetRace(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return model.Race{}, fmt.Errorf("failed to get Race for id %d: %w", id, err)
	}
	o2 := raceFromDBModel(o)
	return o2, nil
}

func (r *Storage) ListRaceIDs(ctx context.Context) ([]int32, error) {
	ids, err := r.q.ListRaceIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list race IDs: %w", err)
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func raceFromDBModel(r sqlc.Race) model.Race {
	if r.ID == 0 {
		return model.Race{}
	}
	return model.Race{
		Description: r.Description,
		ID:          int32(r.ID),
		Name:        r.Name,
	}
}

package storage

import (
	"context"
	"database/sql"
	"errors"
	islices "example/evebuddy/internal/helper/slices"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage/queries"
	"fmt"
)

func (r *Storage) CreateEveRace(ctx context.Context, id int32, description, name string) (model.EveRace, error) {
	arg := queries.CreateEveRaceParams{
		ID:          int64(id),
		Description: description,
		Name:        name,
	}
	o, err := r.q.CreateEveRace(ctx, arg)
	if err != nil {
		return model.EveRace{}, fmt.Errorf("failed to create race %d: %w", id, err)
	}
	return eveRaceFromDBModel(o), nil
}

func (r *Storage) GetEveRace(ctx context.Context, id int32) (model.EveRace, error) {
	o, err := r.q.GetEveRace(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return model.EveRace{}, fmt.Errorf("failed to get Race for id %d: %w", id, err)
	}
	return eveRaceFromDBModel(o), nil
}

func (r *Storage) ListEveRaceIDs(ctx context.Context) ([]int32, error) {
	ids, err := r.q.ListEveRaceIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list race IDs: %w", err)
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func eveRaceFromDBModel(r queries.EveRace) model.EveRace {
	if r.ID == 0 {
		return model.EveRace{}
	}
	return model.EveRace{
		Description: r.Description,
		ID:          int32(r.ID),
		Name:        r.Name,
	}
}

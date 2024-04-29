package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func (r *Storage) CreateEveRegion(ctx context.Context, description string, id int32, name string) (*model.EveRegion, error) {
	if id == 0 {
		return nil, fmt.Errorf("invalid ID %d", id)
	}
	arg := queries.CreateEveRegionParams{
		ID:          int64(id),
		Description: description,
		Name:        name,
	}
	e, err := r.q.CreateEveRegion(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to create EveRegion %v, %w", arg, err)
	}
	return eveRegionFromDBModel(e), nil
}

func (r *Storage) GetEveRegion(ctx context.Context, id int32) (*model.EveRegion, error) {
	c, err := r.q.GetEveRegion(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get EveRegion for id %d: %w", id, err)
	}
	return eveRegionFromDBModel(c), nil
}

func eveRegionFromDBModel(c queries.EveRegion) *model.EveRegion {
	return &model.EveRegion{
		ID:          int32(c.ID),
		Description: c.Description,
		Name:        c.Name,
	}
}

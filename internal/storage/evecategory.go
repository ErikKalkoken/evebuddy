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

func (r *Storage) CreateEveCategory(ctx context.Context, id int32, name string, is_published bool) (model.EveCategory, error) {
	if id == 0 {
		return model.EveCategory{}, fmt.Errorf("invalid ID %d", id)
	}
	arg := queries.CreateEveCategoryParams{
		ID:          int64(id),
		IsPublished: is_published,
		Name:        name,
	}
	e, err := r.q.CreateEveCategory(ctx, arg)
	if err != nil {
		return model.EveCategory{}, fmt.Errorf("failed to create eve category %v, %w", arg, err)
	}
	return eveCategoryFromDBModel(e), nil
}

func (r *Storage) GetEveCategory(ctx context.Context, id int32) (model.EveCategory, error) {
	c, err := r.q.GetEveCategory(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return model.EveCategory{}, fmt.Errorf("failed to get EveCategory for id %d: %w", id, err)
	}
	return eveCategoryFromDBModel(c), nil
}

func (r *Storage) ListEveCategoryIDs(ctx context.Context) ([]int32, error) {
	ids, err := r.q.ListEveCategoryIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list EveCategory IDs: %w", err)
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func eveCategoryFromDBModel(c queries.EveCategory) model.EveCategory {
	return model.EveCategory{
		ID:          int32(c.ID),
		IsPublished: c.IsPublished,
		Name:        c.Name,
	}
}

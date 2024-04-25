package storage

import (
	"context"
	"database/sql"
	"errors"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage/queries"
	"fmt"
)

func (r *Storage) CreateEveGroup(ctx context.Context, id, eve_category_id int32, name string, is_published bool) (model.EveGroup, error) {
	if id == 0 {
		return model.EveGroup{}, fmt.Errorf("invalid ID %d", id)
	}
	arg := queries.CreateEveGroupParams{
		ID:            int64(id),
		EveCategoryID: int64(eve_category_id),
		IsPublished:   is_published,
		Name:          name,
	}
	g, err := r.q.CreateEveGroup(ctx, arg)
	if err != nil {
		return model.EveGroup{}, fmt.Errorf("failed to create EveGroup %v, %w", arg, err)
	}
	return eveGroupFromDBModel(g), nil
}

func (r *Storage) GetEveGroup(ctx context.Context, id int32) (model.EveGroup, error) {
	c, err := r.q.GetEveGroup(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return model.EveGroup{}, fmt.Errorf("failed to get EveGroup for id %d: %w", id, err)
	}
	return eveGroupFromDBModel(c), nil
}

func eveGroupFromDBModel(c queries.EveGroup) model.EveGroup {
	return model.EveGroup{
		ID:          int32(c.ID),
		IsPublished: c.IsPublished,
		Name:        c.Name,
	}
}

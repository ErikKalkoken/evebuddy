package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage/queries"
)

func (r *Storage) CreateEveGroup(ctx context.Context, id, eve_category_id int32, name string, is_published bool) error {
	if id == 0 {
		return fmt.Errorf("invalid ID %d", id)
	}
	arg := queries.CreateEveGroupParams{
		ID:            int64(id),
		EveCategoryID: int64(eve_category_id),
		IsPublished:   is_published,
		Name:          name,
	}
	err := r.q.CreateEveGroup(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to create EveGroup %v, %w", arg, err)
	}
	return nil
}

func (r *Storage) GetEveGroup(ctx context.Context, id int32) (*model.EveGroup, error) {
	row, err := r.q.GetEveGroup(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get EveGroup for id %d: %w", id, err)
	}
	g := eveGroupFromDBModel(row.EveGroup, row.EveCategory)
	return g, nil
}

func eveGroupFromDBModel(g queries.EveGroup, c queries.EveCategory) *model.EveGroup {
	return &model.EveGroup{
		Category:    eveCategoryFromDBModel(c),
		ID:          int32(g.ID),
		IsPublished: g.IsPublished,
		Name:        g.Name,
	}
}

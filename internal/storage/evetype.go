package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func (r *Storage) CreateEveType(ctx context.Context, id int32, description string, eve_group_id int32, name string, is_published bool) error {
	if id == 0 {
		return fmt.Errorf("invalid ID %d", id)
	}
	arg := queries.CreateEveTypeParams{
		ID:          int64(id),
		Description: description,
		EveGroupID:  int64(eve_group_id),
		IsPublished: is_published,
		Name:        name,
	}
	err := r.q.CreateEveType(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to create EveType %v, %w", arg, err)
	}
	return nil
}

func (r *Storage) GetEveType(ctx context.Context, id int32) (*model.EveType, error) {
	row, err := r.q.GetEveType(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get EveType for id %d: %w", id, err)
	}
	t := eveTypeFromDBModel(row.EveType, row.EveGroup, row.EveCategory)
	return t, nil
}

func eveTypeFromDBModel(t queries.EveType, g queries.EveGroup, c queries.EveCategory) *model.EveType {
	return &model.EveType{
		ID:          int32(t.ID),
		Description: t.Description,
		Group:       eveGroupFromDBModel(g, c),
		IsPublished: t.IsPublished,
		Name:        t.Name,
	}
}

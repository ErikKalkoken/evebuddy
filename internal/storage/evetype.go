package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateEveTypeParams struct {
	ID          int32
	Description string
	GroupID     int32
	IsPublished bool
	Name        string
}

func (r *Storage) CreateEveType(ctx context.Context, arg CreateEveTypeParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("invalid ID %d", arg.ID)
	}
	arg2 := queries.CreateEveTypeParams{
		ID:          int64(arg.ID),
		Description: arg.Description,
		EveGroupID:  int64(arg.GroupID),
		IsPublished: arg.IsPublished,
		Name:        arg.Name,
	}
	err := r.q.CreateEveType(ctx, arg2)
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

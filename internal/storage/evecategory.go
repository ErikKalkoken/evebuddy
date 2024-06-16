package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateEveCategoryParams struct {
	ID          int32
	IsPublished bool
	Name        string
}

func (st *Storage) CreateEveCategory(ctx context.Context, arg CreateEveCategoryParams) (*app.EveCategory, error) {
	if arg.ID == 0 {
		return nil, fmt.Errorf("invalid EveCategory ID %d", arg.ID)
	}
	arg2 := queries.CreateEveCategoryParams{
		ID:          int64(arg.ID),
		IsPublished: arg.IsPublished,
		Name:        arg.Name,
	}
	e, err := st.q.CreateEveCategory(ctx, arg2)
	if err != nil {
		return nil, fmt.Errorf("failed to create EveCategory %v, %w", arg, err)
	}
	return eveCategoryFromDBModel(e), nil
}

func (st *Storage) GetEveCategory(ctx context.Context, id int32) (*app.EveCategory, error) {
	c, err := st.q.GetEveCategory(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get EveCategory for id %d: %w", id, err)
	}
	return eveCategoryFromDBModel(c), nil
}

func eveCategoryFromDBModel(c queries.EveCategory) *app.EveCategory {
	return &app.EveCategory{
		ID:          int32(c.ID),
		IsPublished: c.IsPublished,
		Name:        c.Name,
	}
}

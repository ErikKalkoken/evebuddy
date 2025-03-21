package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveGroupParams struct {
	ID          int32
	CategoryID  int32
	IsPublished bool
	Name        string
}

func (st *Storage) CreateEveGroup(ctx context.Context, arg CreateEveGroupParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("invalid ID %d", arg.ID)
	}
	arg2 := queries.CreateEveGroupParams{
		ID:            int64(arg.ID),
		EveCategoryID: int64(arg.CategoryID),
		IsPublished:   arg.IsPublished,
		Name:          arg.Name,
	}
	err := st.q.CreateEveGroup(ctx, arg2)
	if err != nil {
		return fmt.Errorf("create EveGroup %v, %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEveGroup(ctx context.Context, id int32) (*app.EveGroup, error) {
	row, err := st.q.GetEveGroup(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("get EveGroup for id %d: %w", id, err)
	}
	g := eveGroupFromDBModel(row.EveGroup, row.EveCategory)
	return g, nil
}

func eveGroupFromDBModel(g queries.EveGroup, c queries.EveCategory) *app.EveGroup {
	return &app.EveGroup{
		Category:    eveCategoryFromDBModel(c),
		ID:          int32(g.ID),
		IsPublished: g.IsPublished,
		Name:        g.Name,
	}
}

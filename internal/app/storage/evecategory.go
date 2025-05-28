package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveCategoryParams struct {
	ID          int32
	IsPublished bool
	Name        string
}

func (st *Storage) CreateEveCategory(ctx context.Context, arg CreateEveCategoryParams) (*app.EveCategory, error) {
	return createEveCategory(ctx, st.qRW, arg)
}

func createEveCategory(ctx context.Context, q *queries.Queries, arg CreateEveCategoryParams) (*app.EveCategory, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("createEveCategory: %+v: %w", arg, err)
	}
	if arg.ID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	e, err := q.CreateEveCategory(ctx, queries.CreateEveCategoryParams{
		ID:          int64(arg.ID),
		IsPublished: arg.IsPublished,
		Name:        arg.Name,
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	return eveCategoryFromDBModel(e), nil
}

func (st *Storage) GetEveCategory(ctx context.Context, id int32) (*app.EveCategory, error) {
	return getEveCategory(ctx, st.qRO, id)
}

func getEveCategory(ctx context.Context, q *queries.Queries, id int32) (*app.EveCategory, error) {
	c, err := q.GetEveCategory(ctx, int64(id))
	if err != nil {
		return nil, fmt.Errorf("getEveCategory: %+v: %w", id, convertGetError(err))
	}
	return eveCategoryFromDBModel(c), nil
}

func (st *Storage) GetOrCreateEveCategory(ctx context.Context, arg CreateEveCategoryParams) (*app.EveCategory, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetOrCreateEveCategory: %+v: %w", arg, err)
	}
	var o *app.EveCategory
	tx, err := st.dbRW.Begin()
	if err != nil {
		return nil, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	o, err = getEveCategory(ctx, qtx, arg.ID)
	if err != nil {
		if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		o, err = createEveCategory(ctx, qtx, arg)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return o, nil
}

func eveCategoryFromDBModel(c queries.EveCategory) *app.EveCategory {
	return &app.EveCategory{
		ID:          int32(c.ID),
		IsPublished: c.IsPublished,
		Name:        c.Name,
	}
}

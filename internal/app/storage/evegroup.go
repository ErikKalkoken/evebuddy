package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveGroupParams struct {
	ID          int64
	CategoryID  int64
	IsPublished bool
	Name        string
}

func (st *Storage) CreateEveGroup(ctx context.Context, arg CreateEveGroupParams) error {
	return createEveGroup(ctx, st.qRW, arg)
}

func createEveGroup(ctx context.Context, q *queries.Queries, arg CreateEveGroupParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateEveGroup: %+v: %w", arg, err)
	}
	if arg.ID == 0 || arg.CategoryID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := q.CreateEveGroup(ctx, queries.CreateEveGroupParams{
		ID:            arg.ID,
		EveCategoryID: arg.CategoryID,
		IsPublished:   arg.IsPublished,
		Name:          arg.Name,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetEveGroup(ctx context.Context, id int64) (*app.EveGroup, error) {
	return getEveGroup(ctx, st.qRO, id)
}

func getEveGroup(ctx context.Context, q *queries.Queries, id int64) (*app.EveGroup, error) {
	row, err := q.GetEveGroup(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("GetEveGroup for id %d: %w", id, convertGetError(err))
	}
	return eveGroupFromDBModel(row.EveGroup, row.EveCategory), nil
}

func (st *Storage) GetOrCreateEveGroup(ctx context.Context, arg CreateEveGroupParams) (*app.EveGroup, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetOrCreateEveGroup: %+v: %w", arg, err)
	}
	var o *app.EveGroup
	tx, err := st.dbRW.Begin()
	if err != nil {
		return nil, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	o, err = getEveGroup(ctx, qtx, arg.ID)
	if err != nil {
		if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		err := createEveGroup(ctx, qtx, arg)
		if err != nil {
			return nil, err
		}
		x, err := getEveGroup(ctx, qtx, arg.ID)
		if err != nil {
			return nil, err
		}
		o = x
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return o, nil
}

func (st *Storage) ListEveGroupsForCategory(ctx context.Context, categoryID int64) ([]*app.EveGroup, error) {
	rows, err := st.qRO.ListEveGroupsForCategory(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("ListEveGroupsForCategory: %d: %w", categoryID, err)
	}
	var oo []*app.EveGroup
	for _, r := range rows {
		oo = append(oo, eveGroupFromDBModel(r.EveGroup, r.EveCategory))
	}
	return oo, nil
}

func (st *Storage) ListEveSkillGroups(ctx context.Context) ([]*app.EveSkillGroup, error) {
	rows, err := st.qRO.ListEveSkillGroups(ctx, app.EveCategorySkill)
	if err != nil {
		return nil, fmt.Errorf("ListEveSkillGroups: %w", err)
	}
	var oo []*app.EveSkillGroup
	for _, r := range rows {
		oo = append(oo, &app.EveSkillGroup{
			ID:         r.EveGroupID,
			Name:       r.EveGroupName,
			SkillCount: int(r.SkillCount),
		})
	}
	return oo, nil
}
func eveGroupFromDBModel(g queries.EveGroup, c queries.EveCategory) *app.EveGroup {
	return &app.EveGroup{
		Category:    eveCategoryFromDBModel(c),
		ID:          g.ID,
		IsPublished: g.IsPublished,
		Name:        g.Name,
	}
}

// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: eve_groups.sql

package queries

import (
	"context"
)

const createEveGroup = `-- name: CreateEveGroup :exec
INSERT INTO eve_groups (
    id,
    eve_category_id,
    name,
    is_published
)
VALUES (
    ?, ?, ?, ?
)
`

type CreateEveGroupParams struct {
	ID            int64
	EveCategoryID int64
	Name          string
	IsPublished   bool
}

func (q *Queries) CreateEveGroup(ctx context.Context, arg CreateEveGroupParams) error {
	_, err := q.db.ExecContext(ctx, createEveGroup,
		arg.ID,
		arg.EveCategoryID,
		arg.Name,
		arg.IsPublished,
	)
	return err
}

const getEveGroup = `-- name: GetEveGroup :one
SELECT eve_groups.id, eve_groups.eve_category_id, eve_groups.name, eve_groups.is_published, eve_categories.id, eve_categories.name, eve_categories.is_published
FROM eve_groups
JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
WHERE eve_groups.id = ?
`

type GetEveGroupRow struct {
	EveGroup    EveGroup
	EveCategory EveCategory
}

func (q *Queries) GetEveGroup(ctx context.Context, id int64) (GetEveGroupRow, error) {
	row := q.db.QueryRowContext(ctx, getEveGroup, id)
	var i GetEveGroupRow
	err := row.Scan(
		&i.EveGroup.ID,
		&i.EveGroup.EveCategoryID,
		&i.EveGroup.Name,
		&i.EveGroup.IsPublished,
		&i.EveCategory.ID,
		&i.EveCategory.Name,
		&i.EveCategory.IsPublished,
	)
	return i, err
}
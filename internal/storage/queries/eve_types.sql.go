// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: eve_types.sql

package queries

import (
	"context"
)

const createEveType = `-- name: CreateEveType :exec
INSERT INTO eve_types (
    id,
    description,
    eve_group_id,
    name,
    is_published
)
VALUES (
    ?, ?, ?, ?, ?
)
`

type CreateEveTypeParams struct {
	ID          int64
	Description string
	EveGroupID  int64
	Name        string
	IsPublished bool
}

func (q *Queries) CreateEveType(ctx context.Context, arg CreateEveTypeParams) error {
	_, err := q.db.ExecContext(ctx, createEveType,
		arg.ID,
		arg.Description,
		arg.EveGroupID,
		arg.Name,
		arg.IsPublished,
	)
	return err
}

const getEveType = `-- name: GetEveType :one
SELECT eve_types.id, eve_types.description, eve_types.eve_group_id, eve_types.name, eve_types.is_published, eve_groups.id, eve_groups.eve_category_id, eve_groups.name, eve_groups.is_published, eve_categories.id, eve_categories.name, eve_categories.is_published
FROM eve_types
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
WHERE eve_types.id = ?
`

type GetEveTypeRow struct {
	EveType     EveType
	EveGroup    EveGroup
	EveCategory EveCategory
}

func (q *Queries) GetEveType(ctx context.Context, id int64) (GetEveTypeRow, error) {
	row := q.db.QueryRowContext(ctx, getEveType, id)
	var i GetEveTypeRow
	err := row.Scan(
		&i.EveType.ID,
		&i.EveType.Description,
		&i.EveType.EveGroupID,
		&i.EveType.Name,
		&i.EveType.IsPublished,
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
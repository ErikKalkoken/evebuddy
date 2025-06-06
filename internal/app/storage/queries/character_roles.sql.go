// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: character_roles.sql

package queries

import (
	"context"
)

const createCharacterRole = `-- name: CreateCharacterRole :exec
INSERT INTO character_roles (
    character_id,
    name
)
VALUES (
    ?, ?
)
`

type CreateCharacterRoleParams struct {
	CharacterID int64
	Name        string
}

func (q *Queries) CreateCharacterRole(ctx context.Context, arg CreateCharacterRoleParams) error {
	_, err := q.db.ExecContext(ctx, createCharacterRole, arg.CharacterID, arg.Name)
	return err
}

const deleteCharacterRole = `-- name: DeleteCharacterRole :exec
DELETE FROM character_roles
WHERE character_id = ?
AND name = ?
`

type DeleteCharacterRoleParams struct {
	CharacterID int64
	Name        string
}

func (q *Queries) DeleteCharacterRole(ctx context.Context, arg DeleteCharacterRoleParams) error {
	_, err := q.db.ExecContext(ctx, deleteCharacterRole, arg.CharacterID, arg.Name)
	return err
}

const listCharacterRoles = `-- name: ListCharacterRoles :many
SELECT name
FROM character_roles
WHERE character_id = ?
`

func (q *Queries) ListCharacterRoles(ctx context.Context, characterID int64) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, listCharacterRoles, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, name)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: character_tokens.sql

package queries

import (
	"context"
	"time"
)

const addCharacterTokenScope = `-- name: AddCharacterTokenScope :exec
INSERT INTO character_token_scopes (
    character_token_id,
    scope_id
)
VALUES (
    ?, ?
)
`

type AddCharacterTokenScopeParams struct {
	CharacterTokenID int64
	ScopeID          int64
}

func (q *Queries) AddCharacterTokenScope(ctx context.Context, arg AddCharacterTokenScopeParams) error {
	_, err := q.db.ExecContext(ctx, addCharacterTokenScope, arg.CharacterTokenID, arg.ScopeID)
	return err
}

const clearCharacterTokenScopes = `-- name: ClearCharacterTokenScopes :exec
DELETE FROM character_token_scopes
WHERE character_token_id IN (
    SELECT id
    FROM character_tokens
    WHERE character_id = ?
)
`

func (q *Queries) ClearCharacterTokenScopes(ctx context.Context, characterID int64) error {
	_, err := q.db.ExecContext(ctx, clearCharacterTokenScopes, characterID)
	return err
}

const getCharacterToken = `-- name: GetCharacterToken :one
SELECT id, access_token, character_id, expires_at, refresh_token, token_type
FROM character_tokens
WHERE character_id = ?
`

func (q *Queries) GetCharacterToken(ctx context.Context, characterID int64) (CharacterToken, error) {
	row := q.db.QueryRowContext(ctx, getCharacterToken, characterID)
	var i CharacterToken
	err := row.Scan(
		&i.ID,
		&i.AccessToken,
		&i.CharacterID,
		&i.ExpiresAt,
		&i.RefreshToken,
		&i.TokenType,
	)
	return i, err
}

const listCharacterTokenScopes = `-- name: ListCharacterTokenScopes :many
SELECT scopes.id, scopes.name
FROM character_token_scopes
JOIN scopes ON scopes.id = character_token_scopes.scope_id
JOIN character_tokens ON character_tokens.id = character_token_scopes.character_token_id
WHERE character_id = ?
ORDER BY scopes.name
`

func (q *Queries) ListCharacterTokenScopes(ctx context.Context, characterID int64) ([]Scope, error) {
	rows, err := q.db.QueryContext(ctx, listCharacterTokenScopes, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Scope
	for rows.Next() {
		var i Scope
		if err := rows.Scan(&i.ID, &i.Name); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateOrCreateCharacterToken = `-- name: UpdateOrCreateCharacterToken :one
INSERT INTO character_tokens (
    character_id,
    access_token,
    expires_at,
    refresh_token,
    token_type
)
VALUES (
    ?1, ?2, ?3, ?4, ?5
)
ON CONFLICT(character_id) DO
UPDATE SET
    access_token = ?2,
    expires_at = ?3,
    refresh_token = ?4,
    token_type = ?5
WHERE character_id = ?1
RETURNING id, access_token, character_id, expires_at, refresh_token, token_type
`

type UpdateOrCreateCharacterTokenParams struct {
	CharacterID  int64
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string
	TokenType    string
}

func (q *Queries) UpdateOrCreateCharacterToken(ctx context.Context, arg UpdateOrCreateCharacterTokenParams) (CharacterToken, error) {
	row := q.db.QueryRowContext(ctx, updateOrCreateCharacterToken,
		arg.CharacterID,
		arg.AccessToken,
		arg.ExpiresAt,
		arg.RefreshToken,
		arg.TokenType,
	)
	var i CharacterToken
	err := row.Scan(
		&i.ID,
		&i.AccessToken,
		&i.CharacterID,
		&i.ExpiresAt,
		&i.RefreshToken,
		&i.TokenType,
	)
	return i, err
}
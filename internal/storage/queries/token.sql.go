// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: token.sql

package queries

import (
	"context"
	"time"
)

const createToken = `-- name: CreateToken :exec
INSERT INTO tokens (
    access_token,
    expires_at,
    refresh_token,
    token_type,
    character_id
)
VALUES (
    ?, ?, ?, ?, ?
)
`

type CreateTokenParams struct {
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string
	TokenType    string
	CharacterID  int64
}

func (q *Queries) CreateToken(ctx context.Context, arg CreateTokenParams) error {
	_, err := q.db.ExecContext(ctx, createToken,
		arg.AccessToken,
		arg.ExpiresAt,
		arg.RefreshToken,
		arg.TokenType,
		arg.CharacterID,
	)
	return err
}

const getToken = `-- name: GetToken :one
SELECT access_token, character_id, expires_at, refresh_token, token_type
FROM tokens
WHERE character_id = ?
`

func (q *Queries) GetToken(ctx context.Context, characterID int64) (Token, error) {
	row := q.db.QueryRowContext(ctx, getToken, characterID)
	var i Token
	err := row.Scan(
		&i.AccessToken,
		&i.CharacterID,
		&i.ExpiresAt,
		&i.RefreshToken,
		&i.TokenType,
	)
	return i, err
}

const updateToken = `-- name: UpdateToken :exec
UPDATE tokens
SET
    access_token = ?,
    expires_at = ?,
    refresh_token = ?,
    token_type = ?
WHERE character_id = ?
`

type UpdateTokenParams struct {
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string
	TokenType    string
	CharacterID  int64
}

func (q *Queries) UpdateToken(ctx context.Context, arg UpdateTokenParams) error {
	_, err := q.db.ExecContext(ctx, updateToken,
		arg.AccessToken,
		arg.ExpiresAt,
		arg.RefreshToken,
		arg.TokenType,
		arg.CharacterID,
	)
	return err
}
-- name: CreateToken :exec
INSERT INTO tokens (
    access_token,
    expires_at,
    refresh_token,
    token_type,
    character_id
)
VALUES (
    ?, ?, ?, ?, ?
);

-- name: GetToken :one
SELECT *
FROM tokens
WHERE character_id = ?;

-- name: UpdateToken :exec
UPDATE tokens
SET
    access_token = ?,
    expires_at = ?,
    refresh_token = ?,
    token_type = ?
WHERE character_id = ?;
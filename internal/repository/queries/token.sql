-- name: GetToken :one
SELECT *
FROM tokens
WHERE character_id = ?;

-- name: UpdateOrCreateToken :exec
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
ON CONFLICT (character_id) DO
UPDATE SET
    access_token = ?,
    expires_at = ?,
    refresh_token = ?,
    token_type = ?
;
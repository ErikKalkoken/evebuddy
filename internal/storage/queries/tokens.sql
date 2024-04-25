-- name: GetToken :one
SELECT *
FROM tokens
WHERE character_id = ?;

-- name: UpdateOrCreateToken :exec
INSERT INTO tokens (
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
WHERE character_id = ?1;

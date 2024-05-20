-- name: AddCharacterTokenScope :exec
INSERT INTO character_token_scopes (
    token_id,
    scope_id
)
VALUES (
    ?, ?
);

-- name: ClearCharacterTokenScopes :exec
DELETE FROM character_token_scopes
WHERE token_id = ?;

-- name: GetCharacterToken :one
SELECT *
FROM character_tokens
WHERE character_id = ?;

-- name: ListCharacterTokenScopes :many
SELECT scopes.*
FROM character_token_scopes
JOIN scopes ON scopes.id = character_token_scopes.scope_id
WHERE token_id = ?
ORDER BY scopes.name;

-- name: UpdateOrCreateCharacterToken :exec
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
WHERE character_id = ?1;

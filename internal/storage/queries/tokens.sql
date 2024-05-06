-- name: AddTokenScope :exec
INSERT INTO tokens_scopes (
    token_id,
    scope_id
)
VALUES (
    ?, ?
);

-- name: ClearTokenScopes :exec
DELETE FROM tokens_scopes
WHERE token_id = ?;

-- name: GetToken :one
SELECT *
FROM tokens
WHERE my_character_id = ?;

-- name: ListTokenScopes :many
SELECT scopes.*
FROM tokens_scopes
JOIN scopes ON scopes.id = tokens_scopes.scope_id
WHERE token_id = ?
ORDER BY scopes.name;

-- name: UpdateOrCreateToken :exec
INSERT INTO tokens (
    my_character_id,
    access_token,
    expires_at,
    refresh_token,
    token_type
)
VALUES (
    ?1, ?2, ?3, ?4, ?5
)
ON CONFLICT(my_character_id) DO
UPDATE SET
    access_token = ?2,
    expires_at = ?3,
    refresh_token = ?4,
    token_type = ?5
WHERE my_character_id = ?1;

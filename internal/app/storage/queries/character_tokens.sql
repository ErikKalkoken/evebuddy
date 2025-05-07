-- name: AddCharacterTokenScope :exec
INSERT INTO
    character_token_scopes (character_token_id, scope_id)
VALUES
    (?, ?);

-- name: ClearCharacterTokenScopes :exec
DELETE FROM character_token_scopes
WHERE
    character_token_id IN (
        SELECT
            id
        FROM
            character_tokens
        WHERE
            character_id = ?
    );

-- name: GetCharacterToken :one
SELECT
    *
FROM
    character_tokens
WHERE
    character_id = ?;

-- name: ListCharacterTokenForCorporation :many
SELECT
    ct.*
FROM
    character_tokens ct
    JOIN eve_characters ec ON ec.id = ct.character_id
    JOIN character_roles cr ON cr.character_id = ct.character_id
WHERE
    corporation_id = ?
    AND cr.name = ?;

-- name: ListCharacterTokenScopes :many
SELECT
    scopes.*
FROM
    character_token_scopes
    JOIN scopes ON scopes.id = character_token_scopes.scope_id
    JOIN character_tokens ON character_tokens.id = character_token_scopes.character_token_id
WHERE
    character_id = ?
ORDER BY
    scopes.name;

-- name: UpdateOrCreateCharacterToken :one
INSERT INTO
    character_tokens (
        character_id,
        access_token,
        expires_at,
        refresh_token,
        token_type
    )
VALUES
    (?1, ?2, ?3, ?4, ?5)
ON CONFLICT (character_id) DO UPDATE
SET
    access_token = ?2,
    expires_at = ?3,
    refresh_token = ?4,
    token_type = ?5 RETURNING *;

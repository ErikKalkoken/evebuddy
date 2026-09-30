-- name: AddCharacterTokenScopes :exec
INSERT INTO
    character_token_scopes (character_token_id, scope_id)
SELECT
    ?,
    id
FROM
    scopes
WHERE
    name IN (sqlc.slice('names'));

-- name: DeleteCharacterTokenScopes :exec
DELETE FROM character_token_scopes
WHERE
    character_token_id = ?
    AND scope_id IN (
        SELECT
            id
        FROM
            scopes
        WHERE
            name IN (sqlc.slice('names'))
    );

-- name: GetCharacterToken :one
SELECT
    *
FROM
    character_tokens
WHERE
    character_id = ?;

-- name: ListCharacterTokenForCorporationWithRoles :many
SELECT DISTINCT
    ct.*
FROM
    character_tokens ct
    JOIN eve_characters ec ON ec.id = ct.character_id
    JOIN character_roles cr ON cr.character_id = ct.character_id
WHERE
    ec.corporation_id = ?
    AND cr.name IN (sqlc.slice ('roles'));

-- name: ListCharacterTokenForCorporation :many
SELECT
    ct.*
FROM
    character_tokens ct
    JOIN eve_characters ec ON ec.id = ct.character_id
WHERE
    ec.corporation_id = ?;

-- name: ListCharacterTokenScopeNames :many
SELECT
    scopes.name
FROM
    character_token_scopes
    JOIN scopes ON scopes.id = character_token_scopes.scope_id
    JOIN character_tokens ON character_tokens.id = character_token_scopes.character_token_id
WHERE
    character_id = ?;

-- name: ListCharacterTokenScopesForCorporation :many
SELECT
    ct.character_id,
    s.name
FROM
    character_token_scopes cts
    JOIN scopes s ON s.id = cts.scope_id
    JOIN character_tokens ct ON ct.id = cts.character_token_id
    JOIN eve_characters ec ON ec.id = ct.character_id
WHERE
    ec.corporation_id = ?;

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
    token_type = ?5
RETURNING
    id;

-- name: CreateScopeIfMissing :exec
INSERT INTO
    scopes (name)
VALUES
    (?)
ON CONFLICT (name) DO NOTHING;

-- name: ListScopeNamesForNames :many
SELECT
    name
FROM
    scopes
WHERE
    name IN (sqlc.slice('names'));

-- name: CreateEveEntity :one
INSERT INTO
    eve_entities (id, category, name)
VALUES
    (?, ?, ?) RETURNING *;

-- name: GetEveEntity :one
SELECT
    *
FROM
    eve_entities
WHERE
    id = ?;

-- name: ListEveEntitiesForIDs :many
SELECT
    *
FROM
    eve_entities
WHERE
    id IN (sqlc.slice ('ids'));

-- name: ListEveEntityByNameAndCategory :many
SELECT
    *
FROM
    eve_entities
WHERE
    name = ?
    AND category = ?;

-- name: ListEveEntities :many
SELECT
    *
FROM
    eve_entities;

-- name: ListEveEntitiesForMailRecipients :many
SELECT
    *
FROM
    eve_entities
WHERE
    (name LIKE ? ESCAPE '!') -- sqlc can't parse ESCAPE without parentheses
    AND (
        category IN (sqlc.slice('categories'))
        OR id IN (
            SELECT
                eve_entity_id
            FROM
                character_mail_lists
            WHERE
                character_id = ?
        )
    )
ORDER BY
    name COLLATE NOCASE;

-- name: ListEveEntityIDs :many
SELECT
    id
FROM
    eve_entities;

-- name: UpdateOrCreateEveEntity :one
INSERT INTO
    eve_entities (id, category, name)
VALUES
    (?1, ?2, ?3)
ON CONFLICT (id) DO UPDATE
SET
    category = ?2,
    name = ?3 RETURNING *;

-- name: UpdateEveEntity :exec
UPDATE eve_entities
SET
    name = ?
WHERE
    id = ?;
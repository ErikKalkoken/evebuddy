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

-- name: ListEveEntitiesByName :many
SELECT
    *
FROM
    eve_entities
WHERE
    name = ?;

-- name: ListEveEntitiesByPartialName :many
SELECT
    *
FROM
    eve_entities
WHERE
    name LIKE ?
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
    name = ?3
RETURNING *;

-- name: CreateEveEntity :one
INSERT INTO eve_entities (
    id,
    category,
    name
)
VALUES (
    ?, ?, ?
)
RETURNING *;

-- name: GetEveEntity :one
SELECT *
FROM eve_entities
WHERE id = ?;

-- name: GetEveEntityByNameAndCategory :one
SELECT *
FROM eve_entities
WHERE name = ? AND category = ?;

-- name: ListEveEntitiesByName :many
SELECT *
FROM eve_entities
WHERE name = ?;

-- name: ListEveEntitiesByPartialName :many
SELECT *
FROM eve_entities
WHERE name LIKE ?
ORDER BY name
COLLATE NOCASE;

-- name: ListEveEntityIDs :many
SELECT id
FROM eve_entities;

-- name: UpdateEveEntity :exec
UPDATE eve_entities
SET
    category = ?,
    name = ?
WHERE id = ?;

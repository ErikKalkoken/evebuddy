-- name: CreateEveEntity :exec
INSERT INTO eve_entities (
    id,
    category,
    name
)
VALUES (
    ?, ?, ?
);

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

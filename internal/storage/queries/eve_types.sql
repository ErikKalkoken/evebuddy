-- name: CreateEveType :one
INSERT INTO eve_types (
    id,
    description,
    eve_group_id,
    name,
    is_published
)
VALUES (
    ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetEveType :one
SELECT *
FROM eve_types
WHERE id = ?;

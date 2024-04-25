-- name: CreateEveGroup :one
INSERT INTO eve_groups (
    id,
    eve_category_id,
    name,
    is_published
)
VALUES (
    ?, ?, ?, ?
)
RETURNING *;

-- name: GetEveGroup :one
SELECT *
FROM eve_groups
WHERE id = ?;

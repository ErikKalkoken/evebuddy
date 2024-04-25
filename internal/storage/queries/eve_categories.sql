-- name: CreateEveCategory :one
INSERT INTO eve_categories (
    id,
    name,
    is_published
)
VALUES (
    ?, ?, ?
)
RETURNING *;

-- name: GetEveCategory :one
SELECT *
FROM eve_categories
WHERE id = ?;

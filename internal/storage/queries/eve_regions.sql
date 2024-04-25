-- name: CreateEveRegion :one
INSERT INTO eve_regions (
    id,
    description,
    name
)
VALUES (
    ?, ?, ?
)
RETURNING *;

-- name: GetEveRegion :one
SELECT *
FROM eve_regions
WHERE id = ?;

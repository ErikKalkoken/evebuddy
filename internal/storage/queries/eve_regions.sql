-- name: CreateEveRegion :one
INSERT INTO eve_regions (
    id,
    name
)
VALUES (
    ?, ?
)
RETURNING *;

-- name: GetEveRegion :one
SELECT *
FROM eve_regions
WHERE id = ?;

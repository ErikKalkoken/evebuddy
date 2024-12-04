-- name: CreateEveSchematic :one
INSERT INTO eve_schematics (
    id,
    name,
    cycle_time
)
VALUES (
    ?, ?, ?
)
RETURNING *;

-- name: GetEveSchematic :one
SELECT *
FROM eve_schematics
WHERE id = ?;

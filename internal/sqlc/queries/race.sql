-- name: CreateRace :one
INSERT INTO races (
    id,
    description,
    name
)
VALUES (
    ?, ?, ?
)
RETURNING *;

-- name: GetRace :one
SELECT *
FROM races
WHERE id = ?;

-- name: ListRaceIDs :many
SELECT id
FROM races;

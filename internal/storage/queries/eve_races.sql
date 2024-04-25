-- name: CreateRace :one
INSERT INTO eve_races (
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
FROM eve_races
WHERE id = ?;

-- name: ListRaceIDs :many
SELECT id
FROM eve_races;

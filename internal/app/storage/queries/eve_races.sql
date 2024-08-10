-- name: CreateEveRace :one
INSERT INTO eve_races (
    id,
    description,
    name
)
VALUES (
    ?, ?, ?
)
RETURNING *;

-- name: GetEveRace :one
SELECT *
FROM eve_races
WHERE id = ?;

-- name: ListEveRaceIDs :many
SELECT id
FROM eve_races;

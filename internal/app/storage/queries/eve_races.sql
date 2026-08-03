-- name: CreateEveRace :exec
INSERT INTO
    eve_races (id, description, name, faction_id)
VALUES
    (?, ?, ?, ?);

-- name: GetEveRace :one
SELECT
    sqlc.embed(er),
    ee.name as alliance_name
FROM
    eve_races er
    LEFT JOIN eve_entities ee ON ee.id = er.faction_id
WHERE
    er.id = ?;

-- name: ListEveRaceIDs :many
SELECT
    id
FROM
    eve_races;

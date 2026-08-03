-- name: UpdateOrCreateEveRace :exec
INSERT INTO
    eve_races (id, description, faction_id, name)
VALUES
    (?1, ?2, ?3, ?4)
ON CONFLICT (id) DO UPDATE
SET
    description = ?2,
    faction_id = ?3,
    name = ?4;

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

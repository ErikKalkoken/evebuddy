-- name: CreateEveFaction :exec
INSERT INTO
    eve_factions (
        id,
        corporation_id,
        description,
        is_unique,
        militia_corporation_id,
        name,
        size_factor,
        solar_system_id,
        station_count,
        station_system_count
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetEveFaction :one
SELECT
    sqlc.embed(ef),
    eec.name as corporation_name,
    eem.name as militia_corporation_name,
    ees.name as solar_system_name
FROM
    eve_factions ef
    LEFT JOIN eve_entities eec ON eec.id = ef.corporation_id
    LEFT JOIN eve_entities eem ON eem.id = ef.militia_corporation_id
    LEFT JOIN eve_solar_systems ees ON ees.id = ef.solar_system_id
WHERE
    ef.id = ?;

-- name: ListEveFactionIDs :many
SELECT
    id
FROM
    eve_factions;

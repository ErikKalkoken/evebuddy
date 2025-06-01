-- name: GetEveLocation :one
SELECT
    *
FROM
    eve_locations
WHERE
    id = ?;

-- name: ListEveLocationIDs :many
SELECT
    id
FROM
    eve_locations;

-- name: ListEveLocations :many
SELECT
    *
FROM
    eve_locations;

-- name: ListEveLocationsInSolarSystem :many
SELECT
    *
FROM
    eve_locations
WHERE
    eve_solar_system_id = ?
ORDER BY
    name;

-- name: UpdateOrCreateEveLocation :exec
INSERT INTO
    eve_locations (
        id,
        eve_solar_system_id,
        eve_type_id,
        name,
        owner_id,
        updated_at
    )
VALUES
    (?1, ?2, ?3, ?4, ?5, ?6)
ON CONFLICT (id) DO UPDATE
SET
    eve_solar_system_id = ?2,
    eve_type_id = ?3,
    name = ?4,
    owner_id = ?5,
    updated_at = ?6;
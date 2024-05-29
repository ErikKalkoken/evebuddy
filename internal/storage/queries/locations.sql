-- name: GetLocation :one
SELECT *
FROM locations
WHERE id = ?;

-- name: ListLocationIDs :many
SELECT id
FROM locations;

-- name: UpdateOrCreateLocation :exec
INSERT INTO locations (
    id,
    eve_solar_system_id,
    eve_type_id,
    name,
    owner_id,
    updated_at
)
VALUES (
    ?1, ?2, ?3, ?4, ?5, ?6
)
ON CONFLICT(id) DO
UPDATE SET
    eve_solar_system_id = ?2,
    eve_type_id = ?3,
    name = ?4,
    owner_id = ?5,
    updated_at = ?6
WHERE id = ?1;

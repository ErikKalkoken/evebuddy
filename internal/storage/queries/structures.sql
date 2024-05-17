-- name: CreateStructure :exec
INSERT INTO structures (
    id,
    eve_solar_system_id,
    eve_type_id,
    name,
    owner_id,
    position_x,
    position_y,
    position_z
)
VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: GetStructure :one
SELECT
    sqlc.embed(structures),
    sqlc.embed(owners),
    sqlc.embed(eve_solar_systems),
    sqlc.embed(eve_constellations),
    sqlc.embed(eve_regions),
    eve_type_id
FROM structures
JOIN eve_entities AS owners ON owners.id = structures.owner_id
JOIN eve_solar_systems ON eve_solar_systems.id = structures.eve_solar_system_id
JOIN eve_constellations ON eve_constellations.id = eve_solar_systems.eve_constellation_id
JOIN eve_regions ON eve_regions.id = eve_constellations.eve_region_id
WHERE structures.id = ?;

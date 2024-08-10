-- name: CreateEveSolarSystem :exec
INSERT INTO eve_solar_systems (
    id,
    eve_constellation_id,
    name,
    security_status
)
VALUES (
    ?, ?, ?, ?
);

-- name: GetEveSolarSystem :one
SELECT sqlc.embed(eve_solar_systems), sqlc.embed(eve_constellations), sqlc.embed(eve_regions)
FROM eve_solar_systems
JOIN eve_constellations ON eve_constellations.id = eve_solar_systems.eve_constellation_id
JOIN eve_regions ON eve_regions.id = eve_constellations.eve_region_id
WHERE eve_solar_systems.id = ?;

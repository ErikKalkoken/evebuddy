-- name: CreateEvePlanet :exec
INSERT INTO eve_planets (
    id,
    name,
    eve_solar_system_id,
    eve_type_id
)
VALUES (
    ?, ?, ?, ?
);

-- name: GetEvePlanet :one
SELECT sqlc.embed(ep), sqlc.embed(ess), sqlc.embed(ecn), sqlc.embed(er), sqlc.embed(et), sqlc.embed(eg), sqlc.embed(ect)
FROM eve_planets ep
JOIN eve_solar_systems ess ON ess.id = ep.eve_solar_system_id
JOIN eve_constellations ecn ON ecn.id = ess.eve_constellation_id
JOIN eve_regions er ON er.id = ecn.eve_region_id
JOIN eve_types et ON et.id = ep.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ect ON ect.id = eg.eve_category_id
WHERE ep.id = ?;

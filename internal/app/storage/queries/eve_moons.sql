-- name: CreateEveMoon :exec
INSERT INTO eve_moons (
    id,
    name,
    eve_solar_system_id
)
VALUES (
    ?, ?, ?
);

-- name: GetEveMoon :one
SELECT sqlc.embed(em), sqlc.embed(ess), sqlc.embed(ecn), sqlc.embed(er)
FROM eve_moons em
JOIN eve_solar_systems ess ON ess.id = em.eve_solar_system_id
JOIN eve_constellations ecn ON ecn.id = ess.eve_constellation_id
JOIN eve_regions er ON er.id = ecn.eve_region_id
WHERE em.id = ?;

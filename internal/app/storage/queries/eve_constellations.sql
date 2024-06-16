-- name: CreateEveConstellation :exec
INSERT INTO eve_constellations (
    id,
    eve_region_id,
    name
)
VALUES (
    ?, ?, ?
);

-- name: GetEveConstellation :one
SELECT sqlc.embed(eve_constellations), sqlc.embed(eve_regions)
FROM eve_constellations
JOIN eve_regions ON eve_regions.id = eve_constellations.eve_region_id
WHERE eve_constellations.id = ?;

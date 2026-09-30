-- name: CreatePlanetRoute :exec
INSERT INTO
    planet_routes (
        character_planet_id,
        content_type_id,
        destination_pin_id,
        quantity,
        route_id,
        source_pin_id
    )
VALUES
    (?, ?, ?, ?, ?, ?);

-- name: DeletePlanetRoutes :exec
DELETE FROM
    planet_routes
WHERE
    character_planet_id = ?;

-- name: ListPlanetRoutesForCharacterPlanetIDs :many
SELECT
    sqlc.embed(pr),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec)
FROM
    planet_routes pr
    JOIN eve_types et ON et.id = pr.content_type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    JOIN eve_categories ec ON ec.id = eg.eve_category_id
WHERE
    pr.character_planet_id IN (sqlc.slice('ids'))
ORDER BY
    pr.route_id;

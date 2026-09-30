-- name: CreatePlanetPinContent :exec
INSERT INTO
    planet_pin_contents (planet_pin_id, type_id, amount)
VALUES
    (?, ?, ?);

-- name: ListPlanetPinContentsForPlanetPinIDs :many
SELECT
    ppc.planet_pin_id,
    ppc.amount,
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec)
FROM
    planet_pin_contents ppc
    JOIN eve_types et ON et.id = ppc.type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    JOIN eve_categories ec ON ec.id = eg.eve_category_id
WHERE
    ppc.planet_pin_id IN (sqlc.slice('ids'))
ORDER BY
    et.name;

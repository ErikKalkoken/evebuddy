-- name: CreatePlanetPin :exec
INSERT INTO
    planet_pins (
        character_planet_id,
        extractor_product_type_id,
        factory_schema_id,
        schematic_id,
        type_id,
        expiry_time,
        install_time,
        last_cycle_start,
        pin_id
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeletePlanetPins :exec
DELETE FROM
    planet_pins
WHERE
    character_planet_id = ?;

-- name: GetPlanetPin :one
SELECT
    sqlc.embed(pp),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec),
    es.name as schematic_name,
    es.cycle_time as schematic_cycle,
    fes.name as factory_schematic_name,
    fes.cycle_time as factory_schematic_cycle
FROM
    planet_pins pp
    JOIN eve_types et ON et.id = pp.type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    JOIN eve_categories ec ON ec.id = eg.eve_category_id
    LEFT JOIN eve_schematics es ON es.id = pp.schematic_id
    LEFT JOIN eve_schematics fes ON fes.id = pp.factory_schema_id
WHERE
    character_planet_id = ?
    AND pin_id = ?;

-- name: ListPlanetPins :many
SELECT
    sqlc.embed(pp),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec),
    es.name as schematic_name,
    es.cycle_time as schematic_cycle,
    fes.name as factory_schematic_name,
    fes.cycle_time as factory_schematic_cycle
FROM
    planet_pins pp
    JOIN eve_types et ON et.id = pp.type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    JOIN eve_categories ec ON ec.id = eg.eve_category_id
    LEFT JOIN eve_schematics es ON es.id = pp.schematic_id
    LEFT JOIN eve_schematics fes ON fes.id = pp.factory_schema_id
WHERE
    character_planet_id = ?;